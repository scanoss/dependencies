// SPDX-License-Identifier: GPL-2.0-or-later
/*
 * Copyright (C) 2018-2023 SCANOSS.COM
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

// This file common tasks for the models package

package models

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jmoiron/sqlx"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	_ "modernc.org/sqlite"
)

// loadSQLData Load the specified SQL files into the supplied DB.
func loadSQLData(db *sqlx.DB, ctx context.Context, conn *sqlx.Conn, filename string) error {
	fmt.Printf("Loading test data file: %v\n", filename)
	file, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return execSQL(db, ctx, conn, string(file))
}

// execSQL runs the given SQL against either the supplied connection or the DB pool.
func execSQL(db *sqlx.DB, ctx context.Context, conn *sqlx.Conn, sql string) error {
	var err error
	if conn != nil {
		_, err = conn.ExecContext(ctx, sql)
	} else {
		_, err = db.Exec(sql)
	}
	return err
}

// idempotentDDL rewrites CREATE TABLE/INDEX into their IF NOT EXISTS form so a test
// can load the schema more than once. testSchemaDDL transcribes production and must not
// carry test-only clauses, so the rewrite happens here instead of in the constant.
func idempotentDDL(sql string) string {
	for _, kind := range []string{"TABLE", "INDEX"} {
		bare := "CREATE " + kind + " "
		guarded := bare + "IF NOT EXISTS "
		// Normalise first, so an already-guarded statement is not double-guarded.
		sql = strings.ReplaceAll(sql, guarded, bare)
		sql = strings.ReplaceAll(sql, bare, guarded)
	}
	return sql
}

// testDataFiles are the data fixtures loaded on top of the schema. They carry data
// only; the tables come from testSchemaDDL in test_schema.go. Paths are relative to a
// package directory under pkg/.
var testDataFiles = []string{
	"../models/tests/component.sql",
	"../models/tests/component_version.sql",
	"../models/tests/license.sql",
	"../models/tests/component_version_license.sql",
	"../models/tests/component_version_dependency.sql",
}

// LoadTestSchema creates the production schema in the supplied DB. Call this before
// loading any data fixture, since the fixtures do not define their own tables.
// It is safe to call more than once on the same DB.
func LoadTestSchema(db *sqlx.DB, ctx context.Context, conn *sqlx.Conn) error {
	if err := execSQL(db, ctx, conn, idempotentDDL(testSchemaDDL)); err != nil {
		return fmt.Errorf("failed to load the test schema: %v", err)
	}
	return nil
}

// LoadTestSQLData loads the production schema plus all the test data fixtures.
//
// Load through the pool (conn nil) when the database is an in-memory SQLite one
// capped at a single connection: a checked-out connection held by the caller would
// leave the models nothing to query with.
func LoadTestSQLData(db *sqlx.DB, ctx context.Context, conn *sqlx.Conn) error {
	if err := LoadTestSchema(db, ctx, conn); err != nil {
		return err
	}
	return loadTestSQLDataFiles(db, ctx, conn, testDataFiles)
}

// loadTestSQLDataFiles loads a list of test SQL files.
func loadTestSQLDataFiles(db *sqlx.DB, ctx context.Context, conn *sqlx.Conn, files []string) error {
	for _, file := range files {
		err := loadSQLData(db, ctx, conn, file)
		if err != nil {
			return err
		}
	}
	return nil
}

// CloseDB closes the specified DB and logs any errors.
func CloseDB(db *sqlx.DB) {
	if db != nil {
		// zlog.S.Debugf("Closing DB...")
		err := db.Close()
		if err != nil {
			zlog.S.Warnf("Problem closing DB: %v", err)
		}
	}
}

// CloseConn closes the specified DB connection and logs any errors.
func CloseConn(conn *sqlx.Conn) {
	if conn != nil {
		// zlog.S.Debugf("Closing Connection...")
		err := conn.Close()
		if err != nil {
			zlog.S.Warnf("Problem closing DB connection: %v", err)
		}
	}
}

// CloseRows closes the specified DB query row and logs any errors.
func CloseRows(rows *sqlx.Rows) {
	if rows != nil {
		err := rows.Close()
		if err != nil {
			zlog.S.Warnf("Problem closing Rows: %v", err)
		}
	}
}
