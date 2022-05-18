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
	"github.com/jmoiron/sqlx"
	"github.com/package-url/packageurl-go"
	zlog "scanoss.com/dependencies/pkg/logger"
	"scanoss.com/dependencies/pkg/utils"
)

type GolangProjects struct {
	ctx  context.Context
	conn *sqlx.Conn
}

func NewGolangProjectModel(ctx context.Context, conn *sqlx.Conn) *GolangProjects {
	return &GolangProjects{ctx: ctx, conn: conn}
}

func (m *GolangProjects) GetGoLangUrlByPurlString(purlString, purlReq string) (AllUrl, error) {
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
	return m.GetGoLangUrlByPurl(purl, purlName, purlReq)
}

func (m *GolangProjects) GetGoLangUrlByPurl(purl packageurl.PackageURL, purlName, purlReq string) (AllUrl, error) {
	if len(purl.Version) > 0 {
		return m.GetGolangUrlsByPurlNameTypeVersion(purlName, purl.Type, purl.Version)
	}
	return m.GetGolangUrlsByPurlNameType(purlName, purl.Type, purlReq)
}

func (m *GolangProjects) GetGolangUrlsByPurlNameType(purlName, purlType, purlReq string) (AllUrl, error) {
	if len(purlName) == 0 {
		zlog.S.Errorf("Please specify a valid Purl Name to query")
		return AllUrl{}, errors.New("please specify a valid Purl Name to query")
	}
	if len(purlType) == 0 {
		zlog.S.Errorf("Please specify a valid Purl Type to query: %v", purlName)
		return AllUrl{}, errors.New("please specify a valid Purl Type to query")
	}
	var golangUrls []AllUrl
	err := m.conn.SelectContext(m.ctx, &golangUrls,
		"SELECT component, v.version_name AS version, v.semver AS semver,"+
			" l.license_name AS license, l.spdx_id AS license_id, l.is_spdx AS is_spdx,"+
			" purl_name, mine_id FROM golang_projects u"+
			" LEFT JOIN mines m ON u.mine_id = m.id"+
			" LEFT JOIN licenses l ON u.license_id = l.id"+
			" LEFT JOIN versions v ON u.version_id = v.id"+
			" WHERE m.purl_type = $1 AND u.purl_name = $2 AND is_indexed = True"+
			" ORDER BY version_date DESC",
		purlType, purlName)
	if err != nil {
		zlog.S.Errorf("Failed to query golang projects table for %v - %v: %v", purlType, purlName, err)
		return AllUrl{}, fmt.Errorf("failed to query the golang projects table: %v", err)
	}
	zlog.S.Debugf("Found %v results for %v, %v.", len(golangUrls), purlType, purlName)
	// Pick the most appropriate version to return
	return pickOneUrl(nil, golangUrls, purlName, purlType, purlReq)
}

func (m *GolangProjects) GetGolangUrlsByPurlNameTypeVersion(purlName, purlType, purlVersion string) (AllUrl, error) {
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
			" purl_name, mine_id FROM golang_projects u"+
			" LEFT JOIN mines m ON u.mine_id = m.id"+
			" LEFT JOIN licenses l ON u.license_id = l.id"+
			" LEFT JOIN versions v ON u.version_id = v.id"+
			" WHERE m.purl_type = $1 AND u.purl_name = $2 AND v.version_name = $3 AND is_indexed = True"+
			" ORDER BY version_date DESC",
		purlType, purlName, purlVersion)
	if err != nil {
		zlog.S.Errorf("Failed to query golang projects table for %v - %v: %v", purlType, purlName, err)
		return AllUrl{}, fmt.Errorf("failed to query the golang projects table: %v", err)
	}
	zlog.S.Debugf("Found %v results for %v, %v.", len(allUrls), purlType, purlName)
	// Pick the most appropriate version to return
	return pickOneUrl(nil, allUrls, purlName, purlType, "")
}
