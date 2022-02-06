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
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"testing"
)

func TestDbLoad(t *testing.T) {
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer CloseDB(db)
	err = loadSqlData(db, nil, nil, "./tests/mines.sql")
	if err != nil {
		t.Errorf("failed to load SQL test data: %v", err)
	}
	err = LoadTestSqlData(db, nil, nil)
	if err != nil {
		t.Errorf("failed to load SQL test data: %v", err)
	}
	err = loadSqlData(db, nil, nil, "./tests/does-not-exist.sql")
	if err == nil {
		t.Errorf("did not fail to load SQL test data")
	}
	err = loadTestSqlDataFiles(db, nil, nil, []string{"./tests/does-not-exist.sql"})
	if err == nil {
		t.Errorf("did not fail to load SQL test data")
	}
	err = loadSqlData(db, nil, nil, "./tests/bad_sql.sql")
	if err == nil {
		t.Errorf("did not fail to load SQL test data")
	}
}
