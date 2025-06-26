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

package models

import (
	"context"
	"fmt"
	"testing"

	"github.com/scanoss/go-grpc-helper/pkg/grpc/database"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	myconfig "scanoss.com/dependencies/pkg/config"
)

func TestAllUrlsSearch(t *testing.T) {
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer zlog.SyncZap()
	ctx := ctxzap.ToContext(context.Background(), zlog.L)
	s := ctxzap.Extract(ctx).Sugar()
	db := sqliteSetup(t) // Setup SQL Lite DB
	defer CloseDB(db)
	conn := sqliteConn(t, ctx, db) // Get a connection from the pool
	defer CloseConn(conn)
	err = LoadTestSQLData(db, ctx, conn)
	if err != nil {
		t.Fatalf("failed to load SQL test data: %v", err)
	}
	myConfig, err := myconfig.NewServerConfig(nil)
	if err != nil {
		t.Fatalf("failed to load Config: %v", err)
	}
	myConfig.Components.CommitMissing = true
	myConfig.Database.Trace = true
	allUrlsModel := NewAllURLModel(ctx, s, conn, NewProjectModel(ctx, s, conn),
		NewGolangProjectModel(ctx, s, db, conn, myConfig), database.NewDBSelectContext(s, db, conn, myConfig.Database.Trace))

	allUrls, err := allUrlsModel.GetURLsByPurlNameType("tablestyle", "gem", "")
	if err != nil {
		t.Errorf("all_urls.GetUrlsByPurlName() error = %v", err)
	}
	if len(allUrls.PurlName) == 0 {
		t.Errorf("all_urls.GetUrlsByPurlName() No URLs returned from query")
	}
	fmt.Printf("All Urls: %#v\n", allUrls)

	allUrls, err = allUrlsModel.GetURLsByPurlNameType("NONEXISTENT", "none", "")
	if err != nil {
		t.Errorf("all_urls.GetUrlsByPurlName() error = %v", err)
	}
	if len(allUrls.PurlName) > 0 {
		t.Errorf("all_urls.GetURLsByPurlNameType() URLs found when none should be: %v", allUrlsModel)
	}
	fmt.Printf("No Urls: %v\n", allUrls)

	_, err = allUrlsModel.GetURLsByPurlNameType("NONEXISTENT", "", "")
	if err == nil {
		t.Errorf("all_urls.GetURLsByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = allUrlsModel.GetURLsByPurlNameType("", "", "")
	if err == nil {
		t.Errorf("all_urls.GetURLsByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = allUrlsModel.GetURLsByPurlString("", "")
	if err == nil {
		t.Errorf("all_urls.GetURLsByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = allUrlsModel.GetURLsByPurlString("rubbish-purl", "")
	if err == nil {
		t.Errorf("all_urls.GetURLsByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	allUrls, err = allUrlsModel.GetURLsByPurlString("pkg:gem/tablestyle", "")
	if err != nil {
		t.Errorf("all_urls.GetURLsByPurlString() error = %v", err)
	}
	if len(allUrls.PurlName) == 0 {
		t.Errorf("all_urls.GetURLsByPurlString() No URLs returned from query")
	}
	fmt.Printf("All Urls: %v\n", allUrls)

	allUrls, err = allUrlsModel.GetURLsByPurlString("pkg:golang/google.golang.org/grpc", "")
	if err != nil {
		t.Errorf("all_urls.GetURLsByPurlString() error = %v", err)
	}
	if len(allUrls.PurlName) == 0 {
		t.Errorf("all_urls.GetURLsByPurlString() No URLs returned from query")
	}
	fmt.Printf("Golang URL: %v\n", allUrls)

	fmt.Printf("Searching for pkg:golang/github.com/scanoss/dependencies")
	allUrls, err = allUrlsModel.GetURLsByPurlString("pkg:golang/github.com/scanoss/dependencies", "")
	if err != nil {
		t.Errorf("all_urls.GetURLsByPurlString() error = %v", err)
	}
	if len(allUrls.PurlName) == 0 {
		t.Errorf("all_urls.GetURLsByPurlString() No URLs returned from query")
	}
	fmt.Printf("Golang URL: %v\n", allUrls)
}

func TestAllUrlsSearchVersion(t *testing.T) {
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer zlog.SyncZap()
	ctx := ctxzap.ToContext(context.Background(), zlog.L)
	s := ctxzap.Extract(ctx).Sugar()
	db := sqliteSetup(t) // Setup SQL Lite DB
	defer CloseDB(db)
	conn := sqliteConn(t, ctx, db) // Get a connection from the pool
	defer CloseConn(conn)
	err = LoadTestSQLData(db, ctx, conn)
	if err != nil {
		t.Fatalf("failed to load SQL test data: %v", err)
	}
	myConfig, err := myconfig.NewServerConfig(nil)
	if err != nil {
		t.Fatalf("failed to load Config: %v", err)
	}
	myConfig.Components.CommitMissing = true
	myConfig.Database.Trace = true
	allUrlsModel := NewAllURLModel(ctx, s, conn, NewProjectModel(ctx, s, conn),
		NewGolangProjectModel(ctx, s, db, conn, myConfig), database.NewDBSelectContext(s, db, conn, myConfig.Database.Trace))

	allUrls, err := allUrlsModel.GetURLsByPurlNameTypeVersion("tablestyle", "gem", "0.0.12")
	if err != nil {
		t.Errorf("all_urls.GetUrlsByPurlName() error = %v", err)
	}
	if len(allUrls.PurlName) == 0 {
		t.Errorf("all_urls.GetURLsByPurlNameTypeVersion() No URLs returned from query")
	}
	fmt.Printf("All Urls Version: %#v\n", allUrls)

	allUrls, err = allUrlsModel.GetURLsByPurlString("pkg:gem/tablestyle@0.0.7", "")
	if err != nil {
		t.Errorf("all_urls.GetURLsByPurlString() error = failed to find purl by version string")
	}
	fmt.Printf("All Urls Version String: %#v\n", allUrls)

	_, err = allUrlsModel.GetURLsByPurlNameTypeVersion("", "", "")
	if err == nil {
		t.Errorf("all_urls.GetURLsByPurlNameTypeVersion() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = allUrlsModel.GetURLsByPurlNameTypeVersion("NONEXISTENT", "", "")
	if err == nil {
		t.Errorf("all_urls.GetURLsByPurlNameTypeVersion() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = allUrlsModel.GetURLsByPurlNameTypeVersion("NONEXISTENT", "NONEXISTENT", "")
	if err == nil {
		t.Errorf("all_urls.GetURLsByPurlNameTypeVersion() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}

	allUrls, err = allUrlsModel.GetURLsByPurlString("pkg:gem/tablestyle", "22.22.22") // Shouldn't exist
	if err != nil {
		t.Errorf("all_urls.GetURLsByPurlString() error = failed to find purl by version string")
	}
	if len(allUrls.PurlName) > 0 {
		t.Errorf("all_urls.GetURLsByPurlString() error = Found match, when we shouldn't: %v", allUrls)
	}
}

func TestAllUrlsSearchVersionRequirement(t *testing.T) {
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer zlog.SyncZap()
	ctx := ctxzap.ToContext(context.Background(), zlog.L)
	s := ctxzap.Extract(ctx).Sugar()
	db := sqliteSetup(t) // Setup SQL Lite DB
	defer CloseDB(db)
	conn := sqliteConn(t, ctx, db) // Get a connection from the pool
	defer CloseConn(conn)
	err = LoadTestSQLData(db, ctx, conn)
	if err != nil {
		t.Fatalf("failed to load SQL test data: %v", err)
	}
	myConfig, err := myconfig.NewServerConfig(nil)
	if err != nil {
		t.Fatalf("failed to load Config: %v", err)
	}
	myConfig.Components.CommitMissing = true
	myConfig.Database.Trace = true
	allUrlsModel := NewAllURLModel(ctx, s, conn, NewProjectModel(ctx, s, conn),
		NewGolangProjectModel(ctx, s, db, conn, myConfig), database.NewDBSelectContext(s, db, conn, myConfig.Database.Trace))

	allUrls, err := allUrlsModel.GetURLsByPurlString("pkg:gem/tablestyle", ">0.0.4")
	if err != nil {
		t.Errorf("all_urls.GetUrlsByPurlName() error = %v", err)
	}
	if len(allUrls.PurlName) == 0 {
		t.Errorf("all_urls.GetURLsByPurlString() No URLs returned from query")
	}
	fmt.Printf("All Urls Version: %#v\n", allUrls)

	allUrls, err = allUrlsModel.GetURLsByPurlString("pkg:gem/tablestyle", "<0.0.4>")
	if err != nil {
		t.Errorf("all_urls.GetUrlsByPurlName() error = %v", err)
	}
	if len(allUrls.PurlName) == 0 {
		t.Errorf("all_urls.GetURLsByPurlString() No URLs returned from query")
	}
}

func TestAllUrlsSearchNoProject(t *testing.T) {
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer zlog.SyncZap()
	ctx := ctxzap.ToContext(context.Background(), zlog.L)
	s := ctxzap.Extract(ctx).Sugar()
	db := sqliteSetup(t) // Setup SQL Lite DB
	defer CloseDB(db)
	conn := sqliteConn(t, ctx, db) // Get a connection from the pool
	defer CloseConn(conn)
	err = LoadTestSQLData(db, ctx, conn)
	if err != nil {
		t.Fatalf("failed to load SQL test data: %v", err)
	}
	myConfig, err := myconfig.NewServerConfig(nil)
	if err != nil {
		t.Fatalf("failed to load Config: %v", err)
	}
	myConfig.Components.CommitMissing = true
	myConfig.App.Trace = true
	allUrlsModel := NewAllURLModel(ctx, s, conn, nil, NewGolangProjectModel(ctx, s, db, conn, myConfig), database.NewDBSelectContext(s, db, conn, myConfig.Database.Trace))

	allUrls, err := allUrlsModel.GetURLsByPurlNameType("tablestyle", "gem", "0.0.8")
	if err != nil {
		t.Errorf("all_urls.GetUrlsByPurlName() error = %v", err)
	}
	if len(allUrls.PurlName) == 0 {
		t.Errorf("all_urls.GetURLsByPurlNameType() No URLs returned from query")
	}
	fmt.Printf("All Urls: %#v\n", allUrls)
}

func TestAllUrlsSearchNoLicense(t *testing.T) {
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer zlog.SyncZap()
	ctx := ctxzap.ToContext(context.Background(), zlog.L)
	s := ctxzap.Extract(ctx).Sugar()
	db := sqliteSetup(t) // Setup SQL Lite DB
	defer CloseDB(db)
	conn := sqliteConn(t, ctx, db) // Get a connection from the pool
	defer CloseConn(conn)
	err = LoadTestSQLData(db, ctx, conn)
	if err != nil {
		t.Fatalf("failed to load SQL test data: %v", err)
	}
	myConfig, err := myconfig.NewServerConfig(nil)
	if err != nil {
		t.Fatalf("failed to load Config: %v", err)
	}
	myConfig.Components.CommitMissing = true
	myConfig.App.Trace = true
	allUrlsModel := NewAllURLModel(ctx, s, conn, NewProjectModel(ctx, s, conn),
		NewGolangProjectModel(ctx, s, db, conn, myConfig), database.NewDBSelectContext(s, db, conn, myConfig.Database.Trace))

	allUrls, err := allUrlsModel.GetURLsByPurlString("pkg:gem/tablestyle@0.0.8", "")
	if err != nil {
		t.Errorf("all_urls.GetURLsByPurlString() error = %v", err)
	}
	if len(allUrls.PurlName) == 0 {
		t.Errorf("all_urls.GetURLsByPurlString() No URLs returned from query")
	}
	fmt.Printf("All (with project) Urls: %#v\n", allUrls)
}

func TestAllUrlsSearchBadSql(t *testing.T) {
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer zlog.SyncZap()
	ctx := ctxzap.ToContext(context.Background(), zlog.L)
	s := ctxzap.Extract(ctx).Sugar()
	db := sqliteSetup(t) // Setup SQL Lite DB
	defer CloseDB(db)
	conn := sqliteConn(t, ctx, db) // Get a connection from the pool
	defer CloseConn(conn)
	myConfig, err := myconfig.NewServerConfig(nil)
	if err != nil {
		t.Fatalf("failed to load Config: %v", err)
	}
	myConfig.Components.CommitMissing = true
	myConfig.App.Trace = true
	allUrlsModel := NewAllURLModel(ctx, s, conn, NewProjectModel(ctx, s, conn),
		NewGolangProjectModel(ctx, s, db, conn, myConfig), database.NewDBSelectContext(s, db, conn, myConfig.Database.Trace))
	_, err = allUrlsModel.GetURLsByPurlString("pkg:gem/tablestyle", "")
	if err == nil {
		t.Errorf("all_urls.GetURLsByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = allUrlsModel.GetURLsByPurlString("pkg:gem/tablestyle@0.0.8", "")
	if err == nil {
		t.Errorf("all_urls.GetURLsByPurlString() error = did not get an error: %v", err)
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	// Load some tables (leaving out projects)
	err = loadTestSQLDataFiles(db, ctx, conn, []string{"./tests/mines.sql", "./tests/all_urls.sql", "./tests/licenses.sql", "./tests/versions.sql"})
	if err != nil {
		t.Fatalf("failed to load SQL test data: %v", err)
	}
	// allUrls, err := allUrlsModel.GetURLsByPurlNameType("tablestyle", "gem", "")
	allUrls, err := allUrlsModel.GetURLsByPurlString("pkg:gem/tablestyle@0.0.8", "")
	if err != nil {
		t.Errorf("all_urls.GetUrlsByPurlName() error = %v", err)
	}
	if len(allUrls.PurlName) == 0 {
		t.Errorf("all_urls.GetURLsByPurlNameType() No URLs returned from query")
	}
	fmt.Printf("All Urls: %v\n", allUrls)
}
