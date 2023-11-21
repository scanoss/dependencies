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
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type VersionModel struct {
	ctx  context.Context
	s    *zap.SugaredLogger
	conn *sqlx.Conn
}

type Version struct {
	ID          int32  `db:"id"`
	VersionName string `db:"version_name"`
	SemVer      string `db:"semver"`
}

// TODO add cache for versions already searched for?

// NewVersionModel creates a new instance of the Version Model.
func NewVersionModel(ctx context.Context, s *zap.SugaredLogger, conn *sqlx.Conn) *VersionModel {
	return &VersionModel{ctx: ctx, s: s, conn: conn}
}

// GetVersionByName gets the given version from the versions table.
func (m *VersionModel) GetVersionByName(name string, create bool) (Version, error) {
	if len(name) == 0 {
		m.s.Error("Please specify a valid Version Name to query")
		return Version{}, errors.New("please specify a valid Version Name to query")
	}
	var version Version
	err := m.conn.QueryRowxContext(m.ctx,
		"SELECT id, version_name, semver FROM versions"+
			" WHERE version_name = $1",
		name).StructScan(&version)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		m.s.Errorf("Error: Failed to query versions table for %v: %v", name, err)
		return Version{}, fmt.Errorf("failed to query the versions table: %v", err)
	}
	if create && len(version.VersionName) == 0 { // No version found and requested to create an entry
		return m.saveVersion(name)
	}

	return version, nil
}

// saveVersion writes the given version name to the versions table.
func (m *VersionModel) saveVersion(name string) (Version, error) {
	if len(name) == 0 {
		m.s.Error("Please specify a valid version Name to save")
		return Version{}, errors.New("please specify a valid Version Name to save")
	}
	m.s.Debugf("Attempting to save '%v' to the versions table...", name)
	var version Version
	err := m.conn.QueryRowxContext(m.ctx,
		"INSERT INTO versions (version_name, semver) VALUES($1, $2)"+
			" RETURNING id, version_name, semver",
		name, "", false, false,
	).StructScan(&version)
	if err != nil {
		m.s.Errorf("Error: Failed to insert new version name into versions table for %v: %v", name, err)
		return m.GetVersionByName(name, false) // Search one more time for it, just in case someone else added it
	}
	return version, nil
}
