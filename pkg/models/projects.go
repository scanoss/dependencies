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
	"context"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type ProjectModel struct {
	ctx  context.Context
	s    *zap.SugaredLogger
	conn *sqlx.Conn
}

type Project struct {
	PurlName     string `db:"purl_name"`
	Component    string `db:"component"`
	License      string `db:"license"`
	LicenseID    string `db:"license_id"`
	IsSpdx       bool   `db:"is_spdx"`
	GitLicense   string `db:"g_license"`
	GitLicenseID string `db:"g_license_id"`
	GitIsSpdx    bool   `db:"g_is_spdx"`
}

// NewProjectModel creates a new instance of the Project Model.
func NewProjectModel(ctx context.Context, s *zap.SugaredLogger, conn *sqlx.Conn) *ProjectModel {
	return &ProjectModel{ctx: ctx, s: s, conn: conn}
}

// GetProjectsByPurlName searches the projects' table for details about Purl Name and Type.
func (m *ProjectModel) GetProjectsByPurlName(purlName string, purlType string) ([]Project, error) {
	if len(purlName) == 0 {
		m.s.Error("Please specify a valid Purl Name to query")
		return nil, errors.New("please specify a valid Purl Name to query")
	}
	if len(purlType) == 0 {
		m.s.Error("Please specify a valid Purl Type to query")
		return nil, errors.New("please specify a valid Purl Type to query")
	}
	var allProjects []Project
	err := m.conn.SelectContext(m.ctx, &allProjects,
		"SELECT purl_name, component,"+
			" l.license_name AS   license, l.spdx_id AS   license_id, l.is_spdx AS   is_spdx,"+
			" g.license_name AS g_license, g.spdx_id AS g_license_id, g.is_spdx AS g_is_spdx"+
			" FROM projects p"+
			" LEFT JOIN mines m ON p.mine_id = m.id"+
			" LEFT JOIN licenses l ON p.license_id = l.id"+
			" LEFT JOIN licenses g ON p.git_license_id = g.id"+
			" WHERE m.purl_type = $1 AND p.purl_name = $2",
		purlType, purlName)
	if err != nil {
		m.s.Errorf("Failed to query projects table for %v, %v: %v", purlName, purlType, err)
		return nil, fmt.Errorf("failed to query the projects table: %v", err)
	}
	return allProjects, nil
}

// GetProjectByPurlName searches the projects' table for details about a Purl Name and Mine ID.
func (m *ProjectModel) GetProjectByPurlName(purlName string, mineID int32) (Project, error) {
	if len(purlName) == 0 {
		m.s.Error("Please specify a valid Purl Name to query")
		return Project{}, errors.New("please specify a valid Purl Name to query")
	}
	if mineID < 0 {
		m.s.Error("Please specify a valid Mine ID to query")
		return Project{}, errors.New("please specify a valid Mine ID to query")
	}
	rows, err := m.conn.QueryxContext(m.ctx,
		"SELECT purl_name, component,"+
			" l.license_name AS   license, l.spdx_id AS   license_id, l.is_spdx AS   is_spdx,"+
			" g.license_name AS g_license, g.spdx_id AS g_license_id, g.is_spdx AS g_is_spdx"+
			" FROM projects p"+
			" LEFT JOIN licenses l ON p.license_id = l.id"+
			" LEFT JOIN licenses g ON p.git_license_id = g.id"+
			" WHERE purl_name = $1 AND mine_id = $2",
		purlName, mineID)
	defer CloseRows(rows)
	if err != nil {
		m.s.Errorf("Error: Failed to query projects table for %v, %v: %v", purlName, mineID, err)
		return Project{}, fmt.Errorf("failed to query the projects table: %v", err)
	}
	var project Project
	if rows.Next() {
		err = rows.StructScan(&project)
		if err != nil {
			m.s.Errorf("Failed to parse projects table results for %#v: %v", rows, err)
			m.s.Errorf("Query failed for purl_name = %v, mine_id = %v", purlName, mineID)
			return Project{}, fmt.Errorf("failed to query the projects table: %v", err)
		}
	}
	return project, nil
}
