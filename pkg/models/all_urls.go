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
	"errors"
	"github.com/hashicorp/go-version"
	"github.com/jmoiron/sqlx"
	"log"
	"scanoss.com/dependencies/pkg/utils"
)

type allUrlsModel struct {
	db      *sqlx.DB
	mine    *mineModel
	project *projectModel
}

type AllUrl struct {
	Component string           `db:"component"`
	Version   string           `db:"version"`
	License   string           `db:"license"`
	PurlName  string           `db:"purl_name"`
	SemVer    *version.Version `db:-`
}

func NewAllUrlModel(db *sqlx.DB, mine *mineModel, project *projectModel) *allUrlsModel {
	return &allUrlsModel{db: db, mine: mine, project: project}
}

func (m *allUrlsModel) GetUrlsByPurlString(purlString string) ([]AllUrl, error) {
	if len(purlString) == 0 {
		log.Printf("Please specify a valid Purl String to query: %v", purlString)
		return nil, errors.New("please specify a valid Purl String to query")
	}
	purl, err := utils.PurlFromString(purlString)
	if err != nil {
		return nil, err
	}
	mineId, err := m.mine.GetMineIdByPurlType(purl.Type)
	if err != nil {
		return nil, err
	}
	return m.GetUrlsByPurlName(purl.Name, mineId)
}

func (m *allUrlsModel) GetUrlsByPurlName(purlName string, mineId int) ([]AllUrl, error) {
	if mineId < 0 {
		log.Printf("Please specify a valid Mine ID to query: %v", mineId)
		return nil, errors.New("please specify a valid Mine ID to query")
	}
	if len(purlName) == 0 {
		log.Printf("Please specify a valid Purl Name to query: %v", mineId)
		return nil, errors.New("please specify a valid Purl Name to query")
	}
	var allUrls []AllUrl
	err := m.db.Select(&allUrls,
		"SELECT component, version, license, purl_name FROM all_urls WHERE mine_id = ? AND purl_name = ?",
		mineId, purlName)
	if err != nil {
		log.Printf("Error: Failed to query all urls table for %v, %v: %v", purlName, mineId, err)
		return nil, errors.New("failed to query the all urls table")
	}
	// Check if any of the URL entries is missing a license. If so, search for it in the projects table
	if m.project != nil {
		var projects []Project
		for i, url := range allUrls {
			allUrls[i].SemVer, err = version.NewVersion(url.Version)
			if err != nil {
				log.Printf("Warning: Problem parsing version from string: %v", url)
			}
			if len(url.License) == 0 {
				if len(projects) == 0 { // Only search for the project data once
					projects, err = m.project.GetProjectsByPurlName(purlName, mineId)
					if err != nil {
						log.Printf("Warning: Problem searching projects table for %v, %v", purlName, mineId)
						break // Stop search the rest of the URL entries
					}
				}
				if len(projects) > 0 { // TODO which version of the project entry should be used?
					log.Printf("Adding license data to %v from %v", url, projects[0])
					allUrls[i].License = projects[0].License // Assign the first license found
				}
			}
		}
	}
	return allUrls, nil
}
