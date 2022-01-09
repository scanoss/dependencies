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
	"log"
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

func (m *projectModel) GetProjectsByPurlName(purlName string, purlType string) ([]Project, error) {
	if len(purlName) == 0 {
		log.Printf("Please specify a valid Purl Name to query")
		return nil, errors.New("please specify a valid Purl Name to query")
	}
	if len(purlType) == 0 {
		log.Printf("Please specify a valid Purl Type to query")
		return nil, errors.New("please specify a valid Purl Type to query")
	}
	var allProjects []Project
	err := m.conn.SelectContext(m.ctx, &allProjects,
		"SELECT component, versions, license, purl_name FROM projects p LEFT JOIN mines m ON p.mine_id = m.id"+
			" WHERE m.purl_type = ? AND p.purl_name = ?",
		purlType, purlName)
	if err != nil {
		log.Printf("Error: Failed to query projects table for %v, %v: %v", purlName, purlType, err)
		return nil, fmt.Errorf("failed to query the projects table: %v", err)
	}
	return allProjects, nil
}
