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
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	zlog "scanoss.com/dependencies/pkg/logger"
	"testing"
)

func TestMines(t *testing.T) {
	ctx := context.Background()
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer zlog.SyncZap()
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer CloseDB(db)
	conn, err := db.Connx(ctx) // Get a connection from the pool (with context)
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer CloseConn(conn)
	err = loadSqlData(db, ctx, conn, "./tests/mines.sql")
	if err != nil {
		t.Fatalf("failed to load SQL test data: %v", err)
	}
	mine := NewMineModel(ctx, conn)
	var purlType = "maven"
	mineIds, err := mine.GetMineIdsByPurlType(purlType)
	if err != nil {
		t.Errorf("mines.GetMineIdByPurlType() error = %v", err)
	}
	fmt.Printf("Mine ID for %v: %v\n", purlType, mineIds)

	purlType = "gem"
	mineIds, err = mine.GetMineIdsByPurlType(purlType)
	if err != nil {
		t.Errorf("mines.GetMineIdByPurlType() error = %v", err)
	}
	fmt.Printf("Mine ID for %v: %v\n", purlType, mineIds)

	purlType = ""
	mineIds, err = mine.GetMineIdsByPurlType(purlType)
	if err != nil {
		fmt.Printf("Mine ID not found: %v\n", err)
	} else {
		t.Errorf("mines.GetMineIdByPurlType() found for %v = %v", purlType, mineIds)
	}

	purlType = "NONEXISTENT"
	mineIds, err = mine.GetMineIdsByPurlType(purlType)
	if err != nil {
		fmt.Printf("Mine ID not found: %v\n", err)
	} else {
		t.Errorf("mines.GetMineIdByPurlType() found for %v = %v", purlType, mineIds)
	}
	purlType = "npm"
	mineIds, err = mine.GetMineIdsByPurlType(purlType)
	if err != nil {
		t.Errorf("mines.GetMineIdsByPurlType() error = %v", err)
	}
	fmt.Printf("Mine IDs for %v: %v\n", purlType, mineIds)
}

// TestMinesBadSql test bad queries without creating/loading the mines table
func TestMinesBadSql(t *testing.T) {
	ctx := context.Background()
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer zlog.SyncZap()
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer CloseDB(db)
	conn, err := db.Connx(ctx) // Get a connection from the pool
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer CloseConn(conn)
	mine := NewMineModel(ctx, conn)
	purlType := "NONEXISTENT"
	mineIds, err := mine.GetMineIdsByPurlType(purlType)
	if err != nil {
		fmt.Printf("Mine ID not found: %v\n", err)
	} else {
		t.Errorf("mines.GetMineIdByPurlType() found for %v = %v", purlType, mineIds)
	}
}
