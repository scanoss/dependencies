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

// Handle all interaction with the licenses table

package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"scanoss.com/dependencies/pkg/shared"
)

type DependencyModel struct {
	ctx context.Context
	s   *zap.SugaredLogger
	db  *sqlx.DB
}

type UnresolvedDependency struct {
	Purl        string `json:"dep_purl_name"`
	Requirement string `json:"dep_ver"`
}

// NewDependencyModel create a new instance of the Dependency Model.
func NewDependencyModel(ctx context.Context, s *zap.SugaredLogger, db *sqlx.DB) *DependencyModel {
	return &DependencyModel{ctx: ctx, s: s, db: db}
}

func (m *DependencyModel) GetDependencies(purl string, version string, ecosystem string) ([]UnresolvedDependency, error) {
	// Check if ecosystem is supported
	// (already verified in the constructor but there is no harm to check it again and avoid SQL injection)
	if _, isEcosystemSupported := shared.SupportedEcosystems[ecosystem]; !isEcosystemSupported {
		return nil, errors.New("ecosystem not supported")
	}

	// Get a connection from the pool for this operation
	conn, err := m.db.Connx(m.ctx)
	if err != nil {
		m.s.Errorf("Failed to get database connection: %v", err)

		return nil, fmt.Errorf("database connection error: %v", err)
	}
	defer conn.Close() // Return the connection to the pool when done

	var dependencies []UnresolvedDependency

	// Build query with table name based on ecosystem
	query := fmt.Sprintf("SELECT dep_data FROM %s_dependencies WHERE purl_name = $1 AND version = $2", shared.EcosystemDBMapper[ecosystem])

	// Execute query and scan result into byte slice
	var jsonData []byte
	err = conn.QueryRowxContext(m.ctx, query, purl, version).Scan(&jsonData)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []UnresolvedDependency{}, nil
		}
		m.s.Errorf("Error: Failed to query dependency table for %v_dependencies, purl: %v, version: %v:. Error:%#v", ecosystem, purl, version, err)
		return dependencies, err
	}

	// Unmarshal JSON array directly into slice of UnresolvedDependency
	err = json.Unmarshal(jsonData, &dependencies)
	if err != nil {
		return dependencies, fmt.Errorf("failed to unmarshal dependency data: %v", err)
	}

	return dependencies, nil
}
