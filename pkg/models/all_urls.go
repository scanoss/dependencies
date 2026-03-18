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

	"github.com/jmoiron/sqlx"
	componentHelper "github.com/scanoss/go-component-helper/componenthelper"
	"github.com/scanoss/go-grpc-helper/pkg/grpc/database"
	purlutils "github.com/scanoss/go-purl-helper/pkg"
	"go.uber.org/zap"
)

type AllUrlsModel struct {
	ctx        context.Context
	s          *zap.SugaredLogger
	db         *sqlx.DB
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
	db *sqlx.DB,
	project *ProjectModel,
	golangProj *GolangProjects,
	mineModel *MineModel,
	q *database.DBQueryContext,
) *AllUrlsModel {
	return &AllUrlsModel{
		ctx:        ctx,
		s:          s,
		db:         db,
		project:    project,
		golangProj: golangProj,
		mineModel:  mineModel,
		q:          q,
	}
}

// GetURLsByPurlString searches for component details of the specified Purl string (and optional requirement).
func (m *AllUrlsModel) GetURLsByPurlString(component componentHelper.Component) (AllURL, error) {
	if len(component.Version) > 0 {
		result, err := m.GetURLsByPurlNameTypeVersion(component.Name, component.PurlType, component.Version)
		if err != nil {
			return AllURL{}, err
		}
		if result.PurlName == "" && component.PurlType == "golang" {
			return m.getURLsByGolangPurl(component)
		}
		return result, nil
	}
	result, err := m.GetURLsByPurlNameType(component.Name, component.PurlType)
	if err != nil {
		return AllURL{}, err
	}
	if result.PurlName == "" && component.PurlType == "golang" {
		return m.getURLsByGolangPurl(component)
	}
	return result, nil
}

// getURLsByGolangPurl searches golang_projects table first, then converts a golang purl to a GitHub purl and searches all_urls.
func (m *AllUrlsModel) getURLsByGolangPurl(component componentHelper.Component) (AllURL, error) {
	// First try the golang_projects table
	if m.golangProj != nil {
		result, err := m.golangProj.GetGoLangURLByPurlString(component.Purl, component.Requirement)
		if err == nil && len(result.PurlName) > 0 {
			return result, nil
		}
	}
	// Fall back to converting to GitHub purl and searching all_urls
	purlString := purlutils.ConvertGoPurlStringToGithub(component.Purl) // Convert to GitHub purl
	purl, err := purlutils.PurlFromString(purlString)
	if err != nil {
		return AllURL{}, err
	}
	purlName, err := purlutils.PurlNameFromString(purlString) // Make sure we just have the bare minimum for a Purl Name
	if err != nil {
		return AllURL{}, err
	}
	if len(component.Version) > 0 {
		return m.GetURLsByPurlNameTypeVersion(purlName, purl.Type, component.Version)
	}
	return m.GetURLsByPurlNameType(purlName, purl.Type)
}

// GetURLsByPurlNameType searches for component details of the specified Purl Name/Type (and optional requirement).
func (m *AllUrlsModel) GetURLsByPurlNameType(purlName, purlType string) (AllURL, error) {
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
	return pickOneURL(m.s, m.project, m.mineModel, allUrls, purlName, purlType)
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
	err := m.q.SelectContext(m.ctx, &allUrls, query, purlType, purlName, purlVersion)
	if err != nil {
		m.s.Errorf("Failed to query all urls table for %v - %v: %v", purlType, purlName, err)
		return AllURL{}, fmt.Errorf("failed to query the all urls table: %v", err)
	}
	m.s.Debugf("Found %v results for %v, %v.", len(allUrls), purlType, purlName)
	// Pick one URL to return (checking for license details also)
	return pickOneURL(m.s, m.project, m.mineModel, allUrls, purlName, purlType)
}

// pickOneURL takes the potential matching component/versions and selects the most appropriate one.
func pickOneURL(s *zap.SugaredLogger, projModel *ProjectModel, mineModel *MineModel, allUrls []AllURL, purlName string, purlType string) (AllURL, error) {
	if len(allUrls) == 0 {
		s.Infof("No component match (in urls) found for %v, %v,", purlName, purlType)
		return buildFallbackURL(s, projModel, mineModel, purlName, purlType), nil
	}
	url := allUrls[0]
	url.URL, _ = purlutils.ProjectUrl(purlName, purlType)
	if len(url.License) == 0 && projModel != nil { // Check for a project license if we don't have a component one
		GetURLFromProject(s, projModel, &url, purlName, purlType)
	}
	return url, nil // Return the best component match
}

// buildFallbackURL creates an empty AllURL populated with mine ID and project license info
// when no component match is found in the all_urls table.
func buildFallbackURL(s *zap.SugaredLogger, projModel *ProjectModel, mineModel *MineModel, purlName, purlType string) AllURL {
	url := AllURL{}
	projectURL, err := purlutils.ProjectUrl(purlName, purlType)
	if err != nil {
		s.Errorf("Failed to retrieve project URL for %v, %v: %v", purlName, purlType, err)
	}
	url.URL = projectURL
	if projModel == nil && mineModel == nil {
		return url
	}
	mineIds, err := mineModel.GetMineIdsByPurlType(purlType)
	if err != nil {
		s.Errorf("No component match (in urls) found for %v, %v: %v", purlName, purlType, err)
		return url
	}
	url.MineID = mineIds[0]
	for _, m := range mineIds {
		url.MineID = m
		GetURLFromProject(s, projModel, &url, purlName, purlType)
		if len(url.License) > 0 {
			break
		}
	}
	return url
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
