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
	"github.com/hashicorp/go-version"
	"github.com/jmoiron/sqlx"
	zlog "scanoss.com/dependencies/pkg/logger"
	"scanoss.com/dependencies/pkg/utils"
)

type AllUrlsModel struct {
	ctx     context.Context
	conn    *sqlx.Conn
	project *projectModel
}

type AllUrl struct {
	Component string `db:"component"`
	Version   string `db:"version"`
	//SemVer    string           `db:"semver"` // TODO update database to always have a semver value?
	License   string           `db:"license"`
	LicenseId string           `db:"license_id"`
	IsSpdx    bool             `db:"is_spdx"`
	PurlName  string           `db:"purl_name"`
	MineId    int32            `db:"mine_id"`
	semVer    *version.Version `db:"-"` // TODO what semver should we use?
}

func NewAllUrlModel(ctx context.Context, conn *sqlx.Conn, project *projectModel) *AllUrlsModel {
	return &AllUrlsModel{ctx: ctx, conn: conn, project: project}
}

func (m *AllUrlsModel) GetUrlsByPurlString(purlString string) ([]AllUrl, error) {
	if len(purlString) == 0 {
		zlog.S.Errorf("Please specify a valid Purl String to query: %v", purlString)
		return nil, errors.New("please specify a valid Purl String to query")
	}
	purl, err := utils.PurlFromString(purlString)
	if err != nil {
		return nil, err
	}
	return m.GetUrlsByPurlNameType(purl.Name, purl.Type)
}

func (m *AllUrlsModel) GetUrlsByPurlNameType(purlName string, purlType string) ([]AllUrl, error) {
	if len(purlName) == 0 {
		zlog.S.Errorf("Please specify a valid Purl Name to query")
		return nil, errors.New("please specify a valid Purl Name to query")
	}
	if len(purlType) == 0 {
		zlog.S.Errorf("Please specify a valid Purl Type to query")
		return nil, errors.New("please specify a valid Purl Type to query")
	}
	var allUrls []AllUrl
	err := m.conn.SelectContext(m.ctx, &allUrls,
		"SELECT component, v.version_name AS version,"+ // TODO add v.semver AS semver,
			" l.license_name AS license, l.spdx_id AS license_id, l.is_spdx AS is_spdx,"+
			" purl_name, mine_id FROM all_urls u"+
			" LEFT JOIN mines m ON u.mine_id = m.id"+
			" LEFT JOIN licenses l ON u.license_id = l.id"+
			" LEFT JOIN versions v ON u.version_id = v.id"+
			" WHERE m.purl_type = $1 AND u.purl_name = $2",
		purlType, purlName)
	if err != nil {
		zlog.S.Errorf("Failed to query all urls table for %v - %v: %v", purlType, purlName, err)
		return nil, fmt.Errorf("failed to query the all urls table: %v", err)
	}
	// Check if any of the URL entries is missing a license. If so, search for it in the projects table
	if m.project != nil { // TODO should this not be done when loading the URLs table (mining) - i.e. maybe store the project id?
		var projects = make(map[int32]Project)
		for i, url := range allUrls {
			// TODO Add semver here?
			//allUrls[i].semVer, err = version.NewVersion(url.Version)
			//if err != nil {
			//	zlog.S.Warnf("Problem parsing version from string: %v", url)
			//}
			if len(url.License) == 0 {
				project, ok := projects[url.MineId]    // Check if it's already cached
				if !ok || len(project.PurlName) == 0 { // Only search for the project data once
					zlog.S.Debugf("Caching project data for %v - %v", purlName, url.MineId)
					project, err = m.project.GetProjectByPurlName(purlName, url.MineId)
					if err != nil {
						zlog.S.Warnf("Problem searching projects table for %v, %v", purlName, purlType)
						projects[url.MineId] = Project{PurlName: purlName, License: ""} // Cache an empty license id string. no need to search again for the same entry
					} else {
						projects[url.MineId] = project
					}
				} else {
					zlog.S.Debugf("Project data already cached for %v - %v", purlName, url.MineId)
				}
				project, ok = projects[url.MineId] // Do we have a match?
				if ok && len(project.License) > 0 {
					zlog.S.Debugf("Adding license data to %v from %v", url, project)
					allUrls[i].License = project.License
				}
			}
		}
	}
	return allUrls, nil
}
