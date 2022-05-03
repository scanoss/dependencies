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

func TestAllUrlsSearch(t *testing.T) {
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
	err = LoadTestSqlData(db, ctx, conn)
	if err != nil {
		t.Fatalf("failed to load SQL test data: %v", err)
	}
	allUrlsModel := NewAllUrlModel(ctx, conn, NewProjectModel(ctx, conn))

	allUrls, err := allUrlsModel.GetUrlsByPurlNameType("tablestyle", "gem", "")
	if err != nil {
		t.Errorf("all_urls.GetUrlsByPurlName() error = %v", err)
	}
	if len(allUrls.PurlName) == 0 {
		t.Errorf("all_urls.GetUrlsByPurlName() No URLs returned from query")
	}
	fmt.Printf("All Urls: %#v\n", allUrls)

	allUrls, err = allUrlsModel.GetUrlsByPurlNameType("NONEXISTENT", "none", "")
	if err != nil {
		t.Errorf("all_urls.GetUrlsByPurlName() error = %v", err)
	}
	if len(allUrls.PurlName) > 0 {
		t.Errorf("all_urls.GetUrlsByPurlNameType() URLs found when none should be: %v", allUrlsModel)
	}
	fmt.Printf("No Urls: %v\n", allUrls)

	_, err = allUrlsModel.GetUrlsByPurlNameType("NONEXISTENT", "", "")
	if err == nil {
		t.Errorf("all_urls.GetUrlsByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = allUrlsModel.GetUrlsByPurlNameType("", "", "")
	if err == nil {
		t.Errorf("all_urls.GetUrlsByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = allUrlsModel.GetUrlsByPurlString("", "")
	if err == nil {
		t.Errorf("all_urls.GetUrlsByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = allUrlsModel.GetUrlsByPurlString("rubbish-purl", "")
	if err == nil {
		t.Errorf("all_urls.GetUrlsByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	allUrls, err = allUrlsModel.GetUrlsByPurlString("pkg:gem/tablestyle", "")
	if err != nil {
		t.Errorf("all_urls.GetUrlsByPurlString() error = %v", err)
	}
	if len(allUrls.PurlName) == 0 {
		t.Errorf("all_urls.GetUrlsByPurlString() No URLs returned from query")
	}
	fmt.Printf("All Urls: %v\n", allUrls)
}

func TestAllUrlsSearchVersion(t *testing.T) {
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
	err = LoadTestSqlData(db, ctx, conn)
	if err != nil {
		t.Fatalf("failed to load SQL test data: %v", err)
	}
	allUrlsModel := NewAllUrlModel(ctx, conn, NewProjectModel(ctx, conn))

	allUrls, err := allUrlsModel.GetUrlsByPurlNameTypeVersion("tablestyle", "gem", "0.0.12")
	if err != nil {
		t.Errorf("all_urls.GetUrlsByPurlName() error = %v", err)
	}
	if len(allUrls.PurlName) == 0 {
		t.Errorf("all_urls.GetUrlsByPurlNameTypeVersion() No URLs returned from query")
	}
	fmt.Printf("All Urls Version: %#v\n", allUrls)

	allUrls, err = allUrlsModel.GetUrlsByPurlString("pkg:gem/tablestyle@0.0.7", "")
	if err != nil {
		t.Errorf("all_urls.GetUrlsByPurlString() error = failed to find purl by version string")
	}
	fmt.Printf("All Urls Version String: %#v\n", allUrls)

	_, err = allUrlsModel.GetUrlsByPurlNameTypeVersion("", "", "")
	if err == nil {
		t.Errorf("all_urls.GetUrlsByPurlNameTypeVersion() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = allUrlsModel.GetUrlsByPurlNameTypeVersion("NONEXISTENT", "", "")
	if err == nil {
		t.Errorf("all_urls.GetUrlsByPurlNameTypeVersion() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = allUrlsModel.GetUrlsByPurlNameTypeVersion("NONEXISTENT", "NONEXISTENT", "")
	if err == nil {
		t.Errorf("all_urls.GetUrlsByPurlNameTypeVersion() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}

	allUrls, err = allUrlsModel.GetUrlsByPurlString("pkg:gem/tablestyle", "22.22.22") // Shouldn't exist
	if err != nil {
		t.Errorf("all_urls.GetUrlsByPurlString() error = failed to find purl by version string")
	}
	if len(allUrls.PurlName) > 0 {
		t.Errorf("all_urls.GetUrlsByPurlString() error = Found match, when we shouldn't: %v", allUrls)
	}
}

func TestAllUrlsSearchVersionRequirement(t *testing.T) {
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
	err = LoadTestSqlData(db, ctx, conn)
	if err != nil {
		t.Fatalf("failed to load SQL test data: %v", err)
	}
	allUrlsModel := NewAllUrlModel(ctx, conn, NewProjectModel(ctx, conn))

	allUrls, err := allUrlsModel.GetUrlsByPurlString("pkg:gem/tablestyle", ">0.0.4")
	if err != nil {
		t.Errorf("all_urls.GetUrlsByPurlName() error = %v", err)
	}
	if len(allUrls.PurlName) == 0 {
		t.Errorf("all_urls.GetUrlsByPurlString() No URLs returned from query")
	}
	fmt.Printf("All Urls Version: %#v\n", allUrls)

	allUrls, err = allUrlsModel.GetUrlsByPurlString("pkg:gem/tablestyle", "<0.0.4>")
	if err != nil {
		t.Errorf("all_urls.GetUrlsByPurlName() error = %v", err)
	}
	if len(allUrls.PurlName) == 0 {
		t.Errorf("all_urls.GetUrlsByPurlString() No URLs returned from query")
	}

}

func TestAllUrlsSearchNoProject(t *testing.T) {
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
	err = LoadTestSqlData(db, ctx, conn)
	if err != nil {
		t.Fatalf("failed to load SQL test data: %v", err)
	}
	allUrlsModel := NewAllUrlModel(ctx, conn, nil)

	allUrls, err := allUrlsModel.GetUrlsByPurlNameType("tablestyle", "gem", "0.0.8")
	if err != nil {
		t.Errorf("all_urls.GetUrlsByPurlName() error = %v", err)
	}
	if len(allUrls.PurlName) == 0 {
		t.Errorf("all_urls.GetUrlsByPurlNameType() No URLs returned from query")
	}
	fmt.Printf("All Urls: %#v\n", allUrls)
}

func TestAllUrlsSearchNoLicense(t *testing.T) {
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
	err = LoadTestSqlData(db, ctx, conn)
	if err != nil {
		t.Fatalf("failed to load SQL test data: %v", err)
	}
	allUrlsModel := NewAllUrlModel(ctx, conn, NewProjectModel(ctx, conn))

	allUrls, err := allUrlsModel.GetUrlsByPurlString("pkg:gem/tablestyle@0.0.8", "")
	if err != nil {
		t.Errorf("all_urls.GetUrlsByPurlString() error = %v", err)
	}
	if len(allUrls.PurlName) == 0 {
		t.Errorf("all_urls.GetUrlsByPurlString() No URLs returned from query")
	}
	fmt.Printf("All (with project) Urls: %#v\n", allUrls)
}

func TestAllUrlsSearchBadSql(t *testing.T) {
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
	allUrlsModel := NewAllUrlModel(ctx, conn, NewProjectModel(ctx, conn))
	_, err = allUrlsModel.GetUrlsByPurlString("pkg:gem/tablestyle", "")
	if err == nil {
		t.Errorf("all_urls.GetUrlsByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = allUrlsModel.GetUrlsByPurlString("pkg:gem/tablestyle@0.0.8", "")
	if err == nil {
		t.Errorf("all_urls.GetUrlsByPurlString() error = did not get an error: %v", err)
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	// Load some tables (leaving out projects)
	err = loadTestSqlDataFiles(db, ctx, conn, []string{"./tests/mines.sql", "./tests/all_urls.sql", "./tests/licenses.sql", "./tests/versions.sql"})
	if err != nil {
		t.Fatalf("failed to load SQL test data: %v", err)
	}
	//allUrls, err := allUrlsModel.GetUrlsByPurlNameType("tablestyle", "gem", "")
	allUrls, err := allUrlsModel.GetUrlsByPurlString("pkg:gem/tablestyle@0.0.8", "")
	if err != nil {
		t.Errorf("all_urls.GetUrlsByPurlName() error = %v", err)
	}
	if len(allUrls.PurlName) == 0 {
		t.Errorf("all_urls.GetUrlsByPurlNameType() No URLs returned from query")
	}
	fmt.Printf("All Urls: %v\n", allUrls)
}
