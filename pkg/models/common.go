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
	"testing"

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
	if conn != nil {
		_, err = conn.ExecContext(ctx, string(file))
	} else {
		_, err = db.Exec(string(file))
	}
	if err != nil {
		return err
	}
	return nil
}

// LoadTestSQLData loads all the required test SQL files.
func LoadTestSQLData(db *sqlx.DB, ctx context.Context, conn *sqlx.Conn) error {
	files := []string{"../models/tests/mines.sql", "../models/tests/all_urls.sql", "../models/tests/projects.sql",
		"../models/tests/licenses.sql", "../models/tests/versions.sql", "../models/tests/npmjs_dependencies.sql",
		"../models/tests/golang_projects.sql",
	}
	return loadTestSQLDataFiles(db, ctx, conn, files)
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

// sqliteSetup sets up an in-memory SQL Lite DB for testing.
func sqliteSetup(t *testing.T) *sqlx.DB {
	db, err := sqlx.Connect("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	return db
}

// sqliteConn sets up a connection to a test DB.
func sqliteConn(t *testing.T, ctx context.Context, db *sqlx.DB) *sqlx.Conn {
	conn, err := db.Connx(ctx) // Get a connection from the pool
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	return conn
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
