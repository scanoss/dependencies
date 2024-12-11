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

// Handle all interaction with the mines table

package models

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type TransientDependencyModel struct {
	ctx  context.Context
	s    *zap.SugaredLogger
	conn *sqlx.Conn
}

type Transient struct {
	Purl    string          `db:"purl_name"`
	Version string          `db:"version"`
	Data    json.RawMessage `db:"dep_data"`
}

// NewMineModel creates a new instance of the 'Mine' Model.
func NewTransientModel(ctx context.Context, s *zap.SugaredLogger, conn *sqlx.Conn) *TransientDependencyModel {
	return &TransientDependencyModel{ctx: ctx, s: s, conn: conn}
}

// GetMineIdsByPurlType retrieves a list of the Purl Type IDs associated with the given Purl Type (string).
func (m *TransientDependencyModel) GetTransientDependencies(purl []string, version []string) ([]Transient, error) {
	if len(purl) == 0 {
		m.s.Error("Please specify a Purl Type to query")
		return nil, errors.New("please specify a Purl Type to query")
	}
	var deps []Transient
	// Option 2: Alternative approach using string formatting (be careful with SQL injection!)
	fmt.Printf("Executing query")
	query := fmt.Sprintf(
		"SELECT purl_name, version, dep_data FROM npmjs_dependencies WHERE purl_name IN ('%s') AND version IN ('%s')",
		strings.Join(purl, "','"),
		strings.Join(version, "','"))

	err := m.conn.SelectContext(m.ctx, &deps,
		query)
	if err != nil {
		m.s.Errorf("Error: Failed to query mines table for %v: %v", purl, err)
		return []Transient{}, fmt.Errorf("failed to query the mines table: %v", err)
	}

	if len(deps) > 0 {
		return deps, nil
	}
	return deps, nil

}
