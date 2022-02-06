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
	zlog "scanoss.com/dependencies/pkg/logger"
)

type projectModel struct {
	ctx  context.Context
	conn *sqlx.Conn
}

type Project struct {
	Component string `db:"component"`
	Versions  int    `db:"versions"`
	License   string `db:"license"`
	PurlName  string `db:"purl_name"`
}

func NewProjectModel(ctx context.Context, conn *sqlx.Conn) *projectModel {
	return &projectModel{ctx: ctx, conn: conn}
}

// GetProjectsByPurlName searches the projects' table for details about Purl Name and Type
func (m *projectModel) GetProjectsByPurlName(purlName string, purlType string) ([]Project, error) {
	if len(purlName) == 0 {
		zlog.S.Error("Please specify a valid Purl Name to query")
		return nil, errors.New("please specify a valid Purl Name to query")
	}
	if len(purlType) == 0 {
		zlog.S.Error("Please specify a valid Purl Type to query")
		return nil, errors.New("please specify a valid Purl Type to query")
	}
	var allProjects []Project
	err := m.conn.SelectContext(m.ctx, &allProjects,
		"SELECT component, versions, license, purl_name FROM projects p LEFT JOIN mines m ON p.mine_id = m.id"+
			" WHERE m.purl_type = $1 AND p.purl_name = $2",
		purlType, purlName)
	if err != nil {
		zlog.S.Errorf("Error: Failed to query projects table for %v, %v: %v", purlName, purlType, err)
		return nil, fmt.Errorf("failed to query the projects table: %v", err)
	}
	return allProjects, nil
}

// GetProjectByPurlName searches the projects' table for details about a Purl Name and Mine ID
func (m *projectModel) GetProjectByPurlName(purlName string, mineId int32) (Project, error) {
	if len(purlName) == 0 {
		zlog.S.Error("Please specify a valid Purl Name to query")
		return Project{}, errors.New("please specify a valid Purl Name to query")
	}
	if mineId < 0 {
		zlog.S.Error("Please specify a valid Mine ID to query")
		return Project{}, errors.New("please specify a valid Mine ID to query")
	}
	rows, err := m.conn.QueryxContext(m.ctx,
		"SELECT component, versions, license, purl_name FROM projects WHERE purl_name = $1 AND mine_id = $2",
		purlName, mineId)
	if err != nil {
		zlog.S.Errorf("Error: Failed to query projects table for %v, %v: %v", purlName, mineId, err)
		return Project{}, fmt.Errorf("failed to query the projects table: %v", err)
	}
	var project Project
	for rows.Next() {
		err = rows.StructScan(&project)
		if err != nil {
			zlog.S.Errorf("Error: Failed to parse projects table results for %v: %v", rows, err)
			return Project{}, fmt.Errorf("failed to query the projects table: %v", err)
		}
	}
	return project, nil
}
