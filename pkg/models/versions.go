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

// Handle all interaction with the versions table

package models

import (
	"context"
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
	zlog "scanoss.com/dependencies/pkg/logger"
)

type versionModel struct {
	ctx  context.Context
	conn *sqlx.Conn
}

type Version struct {
	Id          int32  `db:"id"`
	VersionName string `db:"version_name"`
	SemVer      string `db:"semver"`
}

// TODO add cache for versions already searched for?

func NewVersionModel(ctx context.Context, conn *sqlx.Conn) *versionModel {
	return &versionModel{ctx: ctx, conn: conn}
}

// GetVersionByName gets the given version from the versions table
func (m *versionModel) GetVersionByName(name string) (Version, error) {
	if len(name) == 0 {
		zlog.S.Error("Please specify a valid Version Name to query")
		return Version{}, errors.New("please specify a valid Version Name to query")
	}
	rows, err := m.conn.QueryxContext(m.ctx,
		"SELECT id, version_name, semver FROM versions"+
			" WHERE version_name = $1",
		name)
	defer CloseRows(rows)
	if err != nil {
		zlog.S.Errorf("Error: Failed to query versions table for %v: %v", name, err)
		return Version{}, fmt.Errorf("failed to query the versions table: %v", err)
	}
	var version Version
	for rows.Next() {
		err = rows.StructScan(&version)
		if err != nil {
			zlog.S.Errorf("Failed to parse versions table results for %#v: %v", rows, err)
			zlog.S.Errorf("Query failed for version_name = %v", name)
			return Version{}, fmt.Errorf("failed to query the versions table: %v", err)
		}
		break // Only process the first row
	}
	return version, nil
}
