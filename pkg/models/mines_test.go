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
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"testing"
)

func TestMines(t *testing.T) {
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	err = loadSqlData(db, "./tests/mines.sql")
	if err != nil {
		t.Fatalf("failed to load SQL test data: %v", err)
	}
	mine := NewMineModel(db)
	var purlType = "maven"
	mineId, err := mine.GetMineIdByPurlType(purlType)
	if err != nil {
		t.Errorf("mines.GetMineIdByPurlType() error = %v", err)
	}
	fmt.Printf("Mine ID for %v: %v\n", purlType, mineId)

	purlType = "gem"
	mineId, err = mine.GetMineIdByPurlType(purlType)
	if err != nil {
		t.Errorf("mines.GetMineIdByPurlType() error = %v", err)
	}
	fmt.Printf("Mine ID for %v: %v\n", purlType, mineId)

	purlType = ""
	mineId, err = mine.GetMineIdByPurlType(purlType)
	if err != nil {
		fmt.Printf("Mine ID not found: %v\n", err)
	} else {
		t.Errorf("mines.GetMineIdByPurlType() found for %v = %v", purlType, mineId)
	}

	purlType = "NONEXISTENT"
	mineId, err = mine.GetMineIdByPurlType(purlType)
	if err != nil {
		fmt.Printf("Mine ID not found: %v\n", err)
	} else {
		t.Errorf("mines.GetMineIdByPurlType() found for %v = %v", purlType, mineId)
	}
}

func TestMinesBadSql(t *testing.T) {
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	mine := NewMineModel(db)
	mine.ResetMineCache()
	purlType := "NONEXISTENT"
	mineId, err := mine.GetMineIdByPurlType(purlType)
	if err != nil {
		fmt.Printf("Mine ID not found: %v\n", err)
	} else {
		t.Errorf("mines.GetMineIdByPurlType() found for %v = %v", purlType, mineId)
	}
}
