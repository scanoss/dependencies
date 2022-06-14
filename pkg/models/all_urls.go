// SPDX-License-Identifier: GPL-2.0-or-later
/*
 * Copyright (C) 2018-2022 SCANOSS.COM
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
	semver "github.com/Masterminds/semver/v3"
	"github.com/jmoiron/sqlx"
	zlog "scanoss.com/dependencies/pkg/logger"
	"scanoss.com/dependencies/pkg/utils"
	"sort"
	"strings"
)

type AllUrlsModel struct {
	ctx        context.Context
	conn       *sqlx.Conn
	project    *projectModel
	golangProj *GolangProjects
}

type AllUrl struct {
	Component string `db:"component"`
	Version   string `db:"version"`
	SemVer    string `db:"semver"`
	License   string `db:"license"`
	LicenseId string `db:"license_id"`
	IsSpdx    bool   `db:"is_spdx"`
	PurlName  string `db:"purl_name"`
	MineId    int32  `db:"mine_id"`
	Url       string `db:"-"`
}

// NewAllUrlModel creates a new instance of the All URL Model
func NewAllUrlModel(ctx context.Context, conn *sqlx.Conn, project *projectModel, golangProj *GolangProjects) *AllUrlsModel {
	return &AllUrlsModel{ctx: ctx, conn: conn, project: project, golangProj: golangProj}
}

// GetUrlsByPurlString searches for component details of the specified Purl string (and optional requirement)
func (m *AllUrlsModel) GetUrlsByPurlString(purlString, purlReq string) (AllUrl, error) {
	if len(purlString) == 0 {
		zlog.S.Errorf("Please specify a valid Purl String to query")
		return AllUrl{}, errors.New("please specify a valid Purl String to query")
	}
	purl, err := utils.PurlFromString(purlString)
	if err != nil {
		return AllUrl{}, err
	}
	purlName, err := utils.PurlNameFromString(purlString) // Make sure we just have the bare minimum for a Purl Name
	if err != nil {
		return AllUrl{}, err
	}
	if len(purl.Version) == 0 && len(purlReq) > 0 { // No version specified, but we might have a specific version in the Requirement
		ver := utils.GetVersionFromReq(purlReq)
		if len(ver) > 0 {
			// TODO check what to do if we get a "file" requirement
			purl.Version = ver // Switch to exact version search (faster)
			purlReq = ""
		}
	}
	if purl.Type == "golang" {
		allUrl, err := m.golangProj.GetGoLangUrlByPurl(purl, purlName, purlReq) // Search a separate table for golang dependencies
		// If no golang package is found, but it's a GitHub component, search GitHub for it
		if err == nil && allUrl.Component == "" && strings.HasPrefix(purlString, "pkg:golang/github.com/") {
			zlog.S.Debugf("Didn't find golang component in projects table for %v. Checking all urls...", purlString)
			purlString = utils.ConvertPurlString(purlString) // Convert to GitHub purl
			purl, err = utils.PurlFromString(purlString)
			if err != nil {
				return AllUrl{}, err
			}
			purlName, err = utils.PurlNameFromString(purlString) // Make sure we just have the bare minimum for a Purl Name
			if err != nil {
				return AllUrl{}, err
			}
			zlog.S.Debugf("Now searching All Urls for Purl: %#v, PurlName: %v", purl, purlName)
		} else {
			return allUrl, err
		}
	}
	if len(purl.Version) > 0 {
		return m.GetUrlsByPurlNameTypeVersion(purlName, purl.Type, purl.Version)
	}
	return m.GetUrlsByPurlNameType(purlName, purl.Type, purlReq)
}

// GetUrlsByPurlNameType searches for component details of the specified Purl Name/Type (and optional requirement)
func (m *AllUrlsModel) GetUrlsByPurlNameType(purlName, purlType, purlReq string) (AllUrl, error) {
	if len(purlName) == 0 {
		zlog.S.Errorf("Please specify a valid Purl Name to query")
		return AllUrl{}, errors.New("please specify a valid Purl Name to query")
	}
	if len(purlType) == 0 {
		zlog.S.Errorf("Please specify a valid Purl Type to query: %v", purlName)
		return AllUrl{}, errors.New("please specify a valid Purl Type to query")
	}
	var allUrls []AllUrl
	err := m.conn.SelectContext(m.ctx, &allUrls,
		"SELECT component, v.version_name AS version, v.semver AS semver,"+
			" l.license_name AS license, l.spdx_id AS license_id, l.is_spdx AS is_spdx,"+
			" purl_name, mine_id FROM all_urls u"+
			" LEFT JOIN mines m ON u.mine_id = m.id"+
			" LEFT JOIN licenses l ON u.license_id = l.id"+
			" LEFT JOIN versions v ON u.version_id = v.id"+
			" WHERE m.purl_type = $1 AND u.purl_name = $2"+
			" ORDER BY date DESC",
		purlType, purlName)
	if err != nil {
		zlog.S.Errorf("Failed to query all urls table for %v - %v: %v", purlType, purlName, err)
		return AllUrl{}, fmt.Errorf("failed to query the all urls table: %v", err)
	}
	zlog.S.Debugf("Found %v results for %v, %v.", len(allUrls), purlType, purlName)
	// Pick one URL to return (checking for license details also)
	return pickOneUrl(m.project, allUrls, purlName, purlType, purlReq)
}

// GetUrlsByPurlNameTypeVersion searches for component details of the specified Purl Name/Type and version
func (m *AllUrlsModel) GetUrlsByPurlNameTypeVersion(purlName, purlType, purlVersion string) (AllUrl, error) {
	if len(purlName) == 0 {
		zlog.S.Errorf("Please specify a valid Purl Name to query")
		return AllUrl{}, errors.New("please specify a valid Purl Name to query")
	}
	if len(purlType) == 0 {
		zlog.S.Errorf("Please specify a valid Purl Type to query")
		return AllUrl{}, errors.New("please specify a valid Purl Type to query")
	}
	if len(purlVersion) == 0 {
		zlog.S.Errorf("Please specify a valid Purl Version to query")
		return AllUrl{}, errors.New("please specify a valid Purl Version to query")
	}
	var allUrls []AllUrl
	err := m.conn.SelectContext(m.ctx, &allUrls,
		"SELECT component, v.version_name AS version, v.semver AS semver,"+
			" l.license_name AS license, l.spdx_id AS license_id, l.is_spdx AS is_spdx,"+
			" purl_name, mine_id FROM all_urls u"+
			" LEFT JOIN mines m ON u.mine_id = m.id"+
			" LEFT JOIN licenses l ON u.license_id = l.id"+
			" LEFT JOIN versions v ON u.version_id = v.id"+
			" WHERE m.purl_type = $1 AND u.purl_name = $2 AND v.version_name = $3"+
			" ORDER BY date DESC",
		purlType, purlName, purlVersion)
	if err != nil {
		zlog.S.Errorf("Failed to query all urls table for %v - %v: %v", purlType, purlName, err)
		return AllUrl{}, fmt.Errorf("failed to query the all urls table: %v", err)
	}
	zlog.S.Debugf("Found %v results for %v, %v.", len(allUrls), purlType, purlName)
	// Pick one URL to return (checking for license details also)
	return pickOneUrl(m.project, allUrls, purlName, purlType, "")
}

// pickOneUrl takes the potential matching component/versions and selects the most appropriate one
func pickOneUrl(projModel *projectModel, allUrls []AllUrl, purlName, purlType, purlReq string) (AllUrl, error) {

	if len(allUrls) == 0 {
		zlog.S.Infof("No component match (in urls) found for %v, %v", purlName, purlType)
		return AllUrl{}, nil
	}
	zlog.S.Debugf("Potential Matches: %v", allUrls)
	var c *semver.Constraints
	var urlMap = make(map[*semver.Version]AllUrl)
	if len(purlReq) > 0 {
		zlog.S.Debugf("Building version constraint for %v: %v", purlName, purlReq)
		var err error
		c, err = semver.NewConstraint(purlReq)
		if err != nil {
			zlog.S.Warnf("Encountered an issue parsing version constraint string '%v' (%v,%v): %v", purlReq, purlName, purlType, err)
		}
	}
	zlog.S.Debugf("Checking versions...")
	for _, url := range allUrls {
		if len(url.SemVer) > 0 || len(url.Version) > 0 {
			v, err := semver.NewVersion(url.Version)
			if err != nil && len(url.SemVer) > 0 {
				zlog.S.Debugf("Failed to parse SemVer: '%v'. Trying Version instead: %v (%v)", url.Version, url.SemVer, err)
				v, err = semver.NewVersion(url.SemVer) // Semver failed, try the normal version
			}
			if err != nil {
				zlog.S.Warnf("Encountered an issue parsing version string '%v' (%v) for %v: %v. Using v0.0.0", url.Version, url.SemVer, url, err)
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
			zlog.S.Warnf("Skipping match as it doesn't have a version: %#v", url)
		}
	}
	if len(urlMap) == 0 { // TODO should we return the latest version anyway?
		zlog.S.Warnf("No component match found for %v, %v after filter %v", purlName, purlType, purlReq)
		return AllUrl{}, nil
	}
	var versions = make([]*semver.Version, len(urlMap))
	var vi = 0
	for version := range urlMap { // Save the list of versions so they can be sorted
		versions[vi] = version
		vi++
	}
	zlog.S.Debugf("Version List: %v", versions)
	sort.Sort(semver.Collection(versions))
	version := versions[len(versions)-1] // Get the latest (acceptable) URL version
	zlog.S.Debugf("Sorted versions: %v. Highest: %v", versions, version)

	url, ok := urlMap[version] // Retrieve the latest accepted URL version
	if !ok {
		zlog.S.Errorf("Problem retrieving URL data for %v (%v, %v)", version, purlName, purlType)
		return AllUrl{}, fmt.Errorf("failed to retrieve specific URL version: %v", version)
	}
	url.Url, _ = utils.ProjectUrl(purlName, purlType)

	zlog.S.Debugf("Selected version: %#v", url)
	if len(url.License) == 0 && projModel != nil { // Check for a project license if we don't have a component one
		zlog.S.Debugf("Searching for project license for")
		project, err := projModel.GetProjectByPurlName(purlName, url.MineId)
		if err != nil {
			zlog.S.Warnf("Problem searching projects table for %v, %v", purlName, purlType)
		}
		if ok && len(project.License) > 0 {
			zlog.S.Debugf("Adding license data to %v from %v", url, project)
			url.License = project.License
			url.IsSpdx = project.IsSpdx
			url.LicenseId = project.LicenseId
		}
	}
	return url, nil // Return the best component match
}
