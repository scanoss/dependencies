// SPDX-License-Identifier: GPL-2.0-or-later
/*
 * Copyright (C) 2018-2023 SCANOSS.COM
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 2 of the License, or
 * (at your option) any later version.
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package models

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/scanoss/go-grpc-helper/pkg/grpc/database"

	"github.com/Masterminds/semver/v3"
	"github.com/jmoiron/sqlx"
	purlutils "github.com/scanoss/go-purl-helper/pkg"
	"go.uber.org/zap"
)

type AllUrlsModel struct {
	ctx        context.Context
	s          *zap.SugaredLogger
	conn       *sqlx.Conn
	project    *ProjectModel
	golangProj *GolangProjects
	mineModel  *MineModel
	q          *database.DBQueryContext
}

type AllURL struct {
	Component string `db:"component"`
	Version   string `db:"version"`
	SemVer    string `db:"semver"`
	License   string `db:"license"`
	LicenseID string `db:"license_id"`
	IsSpdx    bool   `db:"is_spdx"`
	PurlName  string `db:"purl_name"`
	MineID    int32  `db:"mine_id"`
	URL       string `db:"-"`
}

// SQL Query constants.
const (
	purlSQLQuerySelect = "SELECT component, v.version_name AS version, v.semver AS semver,"
	licSpdxSQLQuery    = " l.license_name AS license, l.spdx_id AS license_id, l.is_spdx AS is_spdx,"
	verLeftJoinSQL     = " LEFT JOIN versions v ON u.version_id = v.id"
	licLeftJoinSQL     = " LEFT JOIN licenses l ON u.license_id = l.id"
	mineLeftJoinSQL    = " LEFT JOIN mines m ON u.mine_id = m.id"
)

// NewAllURLModel creates a new instance of the 'All URL' Model.
func NewAllURLModel(ctx context.Context,
	s *zap.SugaredLogger,
	conn *sqlx.Conn,
	project *ProjectModel,
	golangProj *GolangProjects,
	mineModel *MineModel,
	q *database.DBQueryContext,
) *AllUrlsModel {
	return &AllUrlsModel{
		ctx:        ctx,
		s:          s,
		conn:       conn,
		project:    project,
		golangProj: golangProj,
		mineModel:  mineModel,
		q:          q,
	}
}

// GetURLsByPurlString searches for component details of the specified Purl string (and optional requirement).
func (m *AllUrlsModel) GetURLsByPurlString(purlString, purlReq string) (AllURL, error) {
	if len(purlString) == 0 {
		m.s.Error("Please specify a valid Purl String to query")
		return AllURL{}, errors.New("please specify a valid Purl String to query")
	}
	purl, err := purlutils.PurlFromString(purlString)
	if err != nil {
		return AllURL{}, err
	}
	purlName, err := purlutils.PurlNameFromString(purlString) // Make sure we just have the bare minimum for a Purl Name
	if err != nil {
		return AllURL{}, err
	}
	// TODO check what to do if we get a "file" requirement
	if len(purlReq) > 0 && strings.HasPrefix(purlReq, "file:") { // internal dependency requirement. Assume latest
		m.s.Debugf("Removing 'local' requirement for purl: %v (req: %v)", purlString, purlReq)
		purlReq = ""
	}
	if len(purl.Version) == 0 && len(purlReq) > 0 { // No version specified, but we might have a specific version in the Requirement
		ver := purlutils.GetVersionFromReq(purlReq)
		if len(ver) > 0 {
			purl.Version = ver // Switch to exact version search (faster)
			purlReq = ""
		}
	}
	if purl.Type == "golang" {
		allURL, err := m.golangProj.GetGoLangURLByPurl(purl, purlName, purlReq) // Search a separate table for golang dependencies
		// If no golang package/license is found, but it's a GitHub component, search GitHub for it
		if err == nil && (len(allURL.Component) == 0 || len(allURL.License) == 0) && strings.HasPrefix(purlString, "pkg:golang/github.com/") {
			if len(allURL.Component) == 0 {
				m.s.Debugf("Didn't find component in golang projects table for %v. Checking all urls...", purlString)
			} else if len(allURL.License) == 0 {
				m.s.Debugf("Didn't find license in golang projects table for %v. Checking all urls...", purlString)
			}
			purlString = purlutils.ConvertGoPurlStringToGithub(purlString) // Convert to GitHub purl
			purl, err = purlutils.PurlFromString(purlString)
			if err != nil {
				return AllURL{}, err
			}
			purlName, err = purlutils.PurlNameFromString(purlString) // Make sure we just have the bare minimum for a Purl Name
			if err != nil {
				return AllURL{}, err
			}
			m.s.Debugf("Now searching All Urls for Purl: %#v, PurlName: %v", purl, purlName)
		} else {
			return allURL, err
		}
	}
	if len(purl.Version) > 0 {
		return m.GetURLsByPurlNameTypeVersion(purlName, purl.Type, purl.Version)
	}
	return m.GetURLsByPurlNameType(purlName, purl.Type, purlReq)
}

// GetURLsByPurlNameType searches for component details of the specified Purl Name/Type (and optional requirement).
func (m *AllUrlsModel) GetURLsByPurlNameType(purlName, purlType, purlReq string) (AllURL, error) {
	if len(purlName) == 0 {
		m.s.Error("Please specify a valid Purl Name to query")
		return AllURL{}, errors.New("please specify a valid Purl Name to query")
	}
	if len(purlType) == 0 {
		m.s.Errorf("Please specify a valid Purl Type to query: %v", purlName)
		return AllURL{}, errors.New("please specify a valid Purl Type to query")
	}
	query := purlSQLQuerySelect + licSpdxSQLQuery + " purl_name, mine_id FROM all_urls u" +
		mineLeftJoinSQL + licLeftJoinSQL + verLeftJoinSQL +
		" WHERE m.purl_type = $1 AND u.purl_name = $2 ORDER BY date DESC"
	var allUrls []AllURL
	err := m.q.SelectContext(m.ctx, &allUrls, query, purlType, purlName)
	if err != nil {
		m.s.Errorf("Failed to query all urls table for %v - %v: %v", purlType, purlName, err)
		return AllURL{}, fmt.Errorf("failed to query the all urls table: %v", err)
	}
	m.s.Debugf("Found %v results for %v, %v.", len(allUrls), purlType, purlName)
	// Pick one URL to return (checking for license details also)
	return pickOneURL(m.s, m.project, m.mineModel, allUrls, purlName, purlType, purlReq)
}

// GetURLsByPurlNameTypeVersion searches for component details of the specified Purl Name/Type and version.
func (m *AllUrlsModel) GetURLsByPurlNameTypeVersion(purlName, purlType, purlVersion string) (AllURL, error) {
	if len(purlName) == 0 {
		m.s.Error("Please specify a valid Purl Name to query")
		return AllURL{}, errors.New("please specify a valid Purl Name to query")
	}
	if len(purlType) == 0 {
		m.s.Error("Please specify a valid Purl Type to query")
		return AllURL{}, errors.New("please specify a valid Purl Type to query")
	}
	if len(purlVersion) == 0 {
		m.s.Error("Please specify a valid Purl Version to query")
		return AllURL{}, errors.New("please specify a valid Purl Version to query")
	}
	query := purlSQLQuerySelect + licSpdxSQLQuery + " purl_name, mine_id FROM all_urls u" +
		mineLeftJoinSQL + licLeftJoinSQL + verLeftJoinSQL +
		" WHERE m.purl_type = $1 AND u.purl_name = $2 AND v.version_name = $3 ORDER BY date DESC"
	var allUrls []AllURL
	fmt.Printf("Query: %v\n", query)
	fmt.Printf("PurlType: %v\n", purlType)
	fmt.Printf("PurlName: %v\n", purlName)
	fmt.Printf("PurlVersion: %v\n", purlVersion)
	err := m.q.SelectContext(m.ctx, &allUrls, query, purlType, purlName, purlVersion)
	if err != nil {
		m.s.Errorf("Failed to query all urls table for %v - %v: %v", purlType, purlName, err)
		return AllURL{}, fmt.Errorf("failed to query the all urls table: %v", err)
	}
	m.s.Debugf("Found %v results for %v, %v.", len(allUrls), purlType, purlName)
	// Pick one URL to return (checking for license details also)
	return pickOneURL(m.s, m.project, m.mineModel, allUrls, purlName, purlType, "")
}

// pickOneURL takes the potential matching component/versions and selects the most appropriate one.
func pickOneURL(s *zap.SugaredLogger, projModel *ProjectModel, mineModel *MineModel, allUrls []AllURL, purlName, purlType, purlReq string) (AllURL, error) {
	if len(allUrls) == 0 {
		s.Infof("No component match (in urls) found for %v, %v,", purlName, purlType)
		url := AllURL{}

		if projModel == nil && mineModel == nil {
			return url, nil
		}

		mineIds, err := mineModel.GetMineIdsByPurlType(purlType)
		if err != nil {
			s.Errorf("No component match (in urls) found for %v, %v: %v", purlName, purlType, err)
			return url, nil
		}
		url.MineID = mineIds[0]
		GetURLFromProject(s, projModel, &url, purlName, purlType)

		return url, nil
	}

	// s.Debugf("Potential Matches: %v", allUrls)
	var c *semver.Constraints
	var urlMap = make(map[*semver.Version]AllURL)
	if len(purlReq) > 0 {
		s.Debugf("Building version constraint for %v: %v", purlName, purlReq)
		var err error
		c, err = semver.NewConstraint(purlReq)
		if err != nil {
			s.Warnf("Encountered an issue parsing version constraint string '%v' (%v,%v): %v", purlReq, purlName, purlType, err)
		}
	}
	s.Debugf("Checking versions...")
	for _, url := range allUrls {
		if len(url.SemVer) > 0 || len(url.Version) > 0 {
			v, err := semver.NewVersion(url.Version)
			if err != nil && len(url.SemVer) > 0 {
				s.Debugf("Failed to parse SemVer: '%v'. Trying Version instead: %v (%v)", url.Version, url.SemVer, err)
				v, err = semver.NewVersion(url.SemVer) // Semver failed, try the normal version
			}
			if err != nil {
				s.Warnf("Encountered an issue parsing version string '%v' (%v) for %v: %v. Using v0.0.0", url.Version, url.SemVer, url, err)
				v, err = semver.NewVersion("v0.0.0") // Semver failed, just use a standard version zero (for now)
			}
			if err == nil {
				if c == nil || c.Check(v) {
					_, ok := urlMap[v]
					if !ok {
						urlMap[v] = url // fits inside the constraint and hasn't already been stored
					}
				}
			}
		} else {
			s.Infof("Skipping match as it doesn't have a version: %#v", url)
		}
	}
	if len(urlMap) == 0 { // TODO should we return the latest version anyway?
		s.Warnf("No component match found for %v, %v after filter %v", purlName, purlType, purlReq)
		return AllURL{}, nil
	}
	var versions = make([]*semver.Version, len(urlMap))
	var vi = 0
	for version := range urlMap { // Save the list of versions so they can be sorted
		versions[vi] = version
		vi++
	}
	sort.Sort(semver.Collection(versions))
	version := versions[len(versions)-1] // Get the latest (acceptable) URL version
	s.Debugf("Sorted versions: %v. Highest: %v", versions, version)

	url, ok := urlMap[version] // Retrieve the latest accepted URL version
	if !ok {
		s.Errorf("Problem retrieving URL data for %v (%v, %v)", version, purlName, purlType)
		return AllURL{}, fmt.Errorf("failed to retrieve specific URL version: %v", version)
	}
	url.URL, _ = purlutils.ProjectUrl(purlName, purlType)

	s.Debugf("Selected version: %#v", url)
	if len(url.License) == 0 && projModel != nil { // Check for a project license if we don't have a component one
		GetURLFromProject(s, projModel, &url, purlName, purlType)
	}
	return url, nil // Return the best component match
}

func GetURLFromProject(s *zap.SugaredLogger, projModel *ProjectModel, url *AllURL, purlName, purlType string) {
	project, err := projModel.GetProjectByPurlName(purlName, url.MineID)
	switch {
	case err != nil:
		s.Warnf("Problem searching projects table for %v, %v", purlName, purlType)
	case len(project.License) > 0:
		s.Debugf("Adding project license data to %v from %v", url, project)
		url.License = project.License
		url.IsSpdx = project.IsSpdx
		url.LicenseID = project.LicenseID
	case len(project.GitLicense) > 0:
		s.Debugf("Adding project git license data to %v from %v", url, project)
		url.License = project.GitLicense
		url.IsSpdx = project.GitIsSpdx
		url.LicenseID = project.GitLicenseID
	}
}
