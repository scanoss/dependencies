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
	zlog "scanoss.com/dependencies/pkg/logger"
	"testing"
)

func TestVersionsSearch(t *testing.T) {
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
	err = loadTestSqlDataFiles(db, ctx, conn, []string{"../models/tests/versions.sql"})
	if err != nil {
		t.Fatalf("failed to load SQL test data: %v", err)
	}
	versionModel := NewVersionModel(ctx, conn)
	var name = "1.0.0"
	fmt.Printf("Searching for version: %v\n", name)
	version, err := versionModel.GetVersionByName(name, false)
	if err != nil {
		t.Errorf("versions.GetVersionByName() error = %v", err)
	}
	if len(version.VersionName) == 0 {
		t.Errorf("versions.GetVersionByName() No version returned from query")
	}
	fmt.Printf("Version: %#v\n", version)

	name = ""
	fmt.Printf("Searching for license: %v\n", name)
	_, err = versionModel.GetVersionByName(name, false)
	if err == nil {
		t.Errorf("versions.GetVersionByName() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	name = ""
	fmt.Printf("Saving for license: %v\n", name)
	_, err = versionModel.saveVersion(name)
	if err == nil {
		t.Errorf("versions.saveVersion() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}

	name = "22.22.22"
	fmt.Printf("Searching for version: %v\n", name)
	version, err = versionModel.GetVersionByName(name, true)
	if err != nil {
		t.Errorf("versions.GetVersionByName() error = %v", err)
	}
	if len(version.VersionName) == 0 {
		t.Errorf("versions.GetVersionByName() No version returned from query")
	}
	fmt.Printf("Version: %#v\n", version)

	name = "22.22.22"
	fmt.Printf("Searching for version: %v\n", name)
	version, err = versionModel.saveVersion(name)
	if err != nil {
		t.Errorf("versions.GetVersionByName() error = %v", err)
	}
	if len(version.VersionName) == 0 {
		t.Errorf("versions.GetVersionByName() No version returned from query")
	}
	fmt.Printf("Version: %#v\n", version)

}

func TestVersionsSearchBadSql(t *testing.T) {
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
	versionModel := NewVersionModel(ctx, conn)
	_, err = versionModel.GetVersionByName("rubbish", false)
	if err == nil {
		t.Errorf("versions.GetVersionByName() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = versionModel.saveVersion("rubbish")
	if err == nil {
		t.Errorf("versions.saveVersion() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
}
