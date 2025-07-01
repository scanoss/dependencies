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
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/scanoss/go-grpc-helper/pkg/grpc/database"
	"github.com/scanoss/go-models-helper/pkg/helpers"
	"github.com/scanoss/go-models-helper/pkg/models"
	purlutils "github.com/scanoss/go-purl-helper/pkg"
	"go.uber.org/zap"
)

type AllUrlsModel struct {
	ctx        context.Context
	s          *zap.SugaredLogger
	conn       *sqlx.Conn
	project    *models.ProjectModel
	golangProj *GolangProjects
	q          *database.DBQueryContext
}

// Use the AllURL struct from the shared models library.
type AllURL = models.AllURL

// SQL Query constants.
const (
	purlSQLQuerySelect = "SELECT component, v.version_name AS version, v.semver AS semver,"
	licSpdxSQLQuery    = " l.license_name AS license, l.spdx_id AS license_id, l.is_spdx AS is_spdx,"
	verLeftJoinSQL     = " LEFT JOIN versions v ON u.version_id = v.id"
	licLeftJoinSQL     = " LEFT JOIN licenses l ON u.license_id = l.id"
	mineLeftJoinSQL    = " LEFT JOIN mines m ON u.mine_id = m.id"
)

// NewAllURLModel creates a new instance of the 'All URL' Model.
func NewAllURLModel(ctx context.Context, s *zap.SugaredLogger, conn *sqlx.Conn, project *models.ProjectModel, golangProj *GolangProjects, q *database.DBQueryContext) *AllUrlsModel {
	return &AllUrlsModel{ctx: ctx, s: s, conn: conn, project: project, golangProj: golangProj, q: q}
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
	// Convert to helpers.AllURL slice and call shared helper
	helperURLs := convertToHelperAllURLs(allUrls)
	var projRepo helpers.ProjectRepository
	if m.project != nil {
		projRepo = m.project
	}
	result, err := helpers.PickOneURL(m.s, projRepo, helperURLs, purlName, purlType, purlReq)
	if err != nil {
		return AllURL{}, err
	}
	// Convert back to local AllURL
	return convertFromHelperAllURL(result), nil
}

// convertToHelperAllURLs converts local AllURL slice to helpers.AllURL slice.
func convertToHelperAllURLs(localURLs []AllURL) []helpers.AllURL {
	helperURLs := make([]helpers.AllURL, len(localURLs))
	for i, url := range localURLs {
		helperURLs[i] = helpers.AllURL{
			Component: url.Component,
			Version:   url.Version,
			SemVer:    url.SemVer,
			License:   url.License,
			LicenseID: url.LicenseID,
			IsSpdx:    url.IsSpdx,
			PurlName:  url.PurlName,
			MineID:    url.MineID,
			URL:       url.URL,
		}
	}
	return helperURLs
}

// convertFromHelperAllURL converts helpers.AllURL to local AllURL.
func convertFromHelperAllURL(helperURL helpers.AllURL) AllURL {
	return AllURL{
		Component: helperURL.Component,
		Version:   helperURL.Version,
		SemVer:    helperURL.SemVer,
		License:   helperURL.License,
		LicenseID: helperURL.LicenseID,
		IsSpdx:    helperURL.IsSpdx,
		PurlName:  helperURL.PurlName,
		MineID:    helperURL.MineID,
		URL:       helperURL.URL,
	}
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
	// Convert to helpers.AllURL slice and call shared helper
	helperURLs := convertToHelperAllURLs(allUrls)
	var projRepo helpers.ProjectRepository
	if m.project != nil {
		projRepo = m.project
	}
	result, err := helpers.PickOneURL(m.s, projRepo, helperURLs, purlName, purlType, "")
	if err != nil {
		return AllURL{}, err
	}
	// Convert back to local AllURL
	return convertFromHelperAllURL(result), nil
}
