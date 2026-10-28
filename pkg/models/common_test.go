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

package models

import (
	"testing"

	"github.com/jmoiron/sqlx"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	_ "modernc.org/sqlite"
)

// sqliteSetup opens an in-memory SQLite database with the serving schema and the
// test fixtures loaded. An in-memory database is per connection, so the pool is
// capped at one and the fixtures go through the pool rather than a held connection.
func sqliteSetup(t *testing.T) *sqlx.DB {
	t.Helper()
	if err := zlog.NewSugaredDevLogger(); err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	db.SetMaxOpenConns(1)
	if err = LoadTestSQLData(db, nil, nil); err != nil {
		t.Fatalf("failed to load SQL test data: %v", err)
	}
	t.Cleanup(func() { CloseDB(db) })
	return db
}

func TestDbLoad(t *testing.T) {
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	db.SetMaxOpenConns(1)
	defer CloseDB(db)
	// The fixtures carry data only, so loading one before the schema fails.
	if err = loadSQLData(db, nil, nil, "./tests/component.sql"); err == nil {
		t.Errorf("loading a fixture before the schema did not fail")
	}
	if err = LoadTestSQLData(db, nil, nil); err != nil {
		t.Errorf("failed to load SQL test data: %v", err)
	}
	// The schema is idempotent, so a second load of it is fine.
	if err = LoadTestSchema(db, nil, nil); err != nil {
		t.Errorf("failed to reload the test schema: %v", err)
	}
	if err = loadSQLData(db, nil, nil, "./tests/does-not-exist.sql"); err == nil {
		t.Errorf("did not fail to load SQL test data")
	}
	if err = loadTestSQLDataFiles(db, nil, nil, []string{"./tests/does-not-exist.sql"}); err == nil {
		t.Errorf("did not fail to load SQL test data")
	}
	if err = loadSQLData(db, nil, nil, "./tests/bad_sql.sql"); err == nil {
		t.Errorf("did not fail to load SQL test data")
	}
}
