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
)

type AllUrlsModel struct {
	ctx     context.Context
	conn    *sqlx.Conn
	project *projectModel
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
}

func NewAllUrlModel(ctx context.Context, conn *sqlx.Conn, project *projectModel) *AllUrlsModel {
	return &AllUrlsModel{ctx: ctx, conn: conn, project: project}
}

func (m *AllUrlsModel) GetUrlsByPurlString(purlString, purlReq string) (AllUrl, error) {
	if len(purlString) == 0 {
		zlog.S.Errorf("Please specify a valid Purl String to query")
		return AllUrl{}, errors.New("please specify a valid Purl String to query")
	}
	purl, err := utils.PurlFromString(purlString)
	if err != nil {
		return AllUrl{}, err
	}
	purlName, err := utils.PurlNameFromString(purlString)
	if err != nil {
		return AllUrl{}, err
	}
	if len(purl.Version) > 0 {
		return m.GetUrlsByPurlNameTypeVersion(purlName, purl.Type, purl.Version)
	}
	return m.GetUrlsByPurlNameType(purlName, purl.Type, purlReq)
}

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
	// Check if any of the URL entries is missing a license. If so, search for it in the projects table
	return m.pickOneUrl(allUrls, purlName, purlType, purlReq)
}

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
	// Check if any of the URL entries is missing a license. If so, search for it in the projects table
	return m.pickOneUrl(allUrls, purlName, purlType, "")
}

func (m *AllUrlsModel) pickOneUrl(allUrls []AllUrl, purlName, purlType, purlReq string) (AllUrl, error) {

	if len(allUrls) == 0 {
		zlog.S.Infof("No component match (in allurls) found for %v, %v", purlName, purlType)
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
				zlog.S.Warnf("Encountered an issue parsing version string '%v' (%v) for %v: %v", url.Version, url.SemVer, url, err)
			} else {
				if c == nil || c.Check(v) {
					//zlog.S.Debugf("Saving URL version %v: %v", v, url)
					urlMap[v] = url // fits inside the constraint
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
	zlog.S.Debugf("Selected version: %#v", url)
	if len(url.License) == 0 && m.project != nil { // Check for a project license if we don't have a component one
		zlog.S.Debugf("Searching for project license for")
		project, err := m.project.GetProjectByPurlName(purlName, url.MineId)
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
