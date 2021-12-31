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

// Handle all interaction with the projects table

package models

import (
	"errors"
	"github.com/jmoiron/sqlx"
	"log"
)

type projectModel struct {
	db *sqlx.DB
}

type Project struct {
	Component string `db:"component"`
	Versions  int    `db:"versions"`
	License   string `db:"license"`
	PurlName  string `db:"purl_name"`
}

func NewProjectModel(db *sqlx.DB) *projectModel {
	return &projectModel{db: db}
}

func (m *projectModel) GetProjectsByPurlName(purlName string, mineId int) ([]Project, error) {
	if mineId < 0 {
		log.Printf("Please specify a valid Mine ID to query: %v", mineId)
		return nil, errors.New("please specify a valid Mine ID to query")
	}
	if len(purlName) == 0 {
		log.Printf("Please specify a valid Purl Name to query: %v", mineId)
		return nil, errors.New("please specify a valid Purl Name to query")
	}
	var allProjects []Project
	err := m.db.Select(&allProjects,
		"SELECT component, versions, license, purl_name FROM projects WHERE mine_id = ? AND purl_name = ?",
		mineId, purlName)
	if err != nil {
		log.Printf("Error: Failed to query projects table for %v, %v: %v", purlName, mineId, err)
		return nil, errors.New("failed to query the projects table")
	}
	return allProjects, nil
}
