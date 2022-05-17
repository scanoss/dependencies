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

func TestGolangProjectUrlsSearch(t *testing.T) {
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
	golangProjModel := NewGolangProjectModel(ctx, conn)

	url, err := golangProjModel.GetGolangUrlsByPurlNameType("google.golang.org/grpc", "golang", "")
	if err != nil {
		t.Errorf("golang_projects.GetUrlsByPurlName() error = %v", err)
	}
	if len(url.PurlName) == 0 {
		t.Errorf("golang_projects.GetGoLangUrlByPurlString() No URLs returned from query")
	}
	fmt.Printf("Golang Url: %#v\n", url)

	url, err = golangProjModel.GetGolangUrlsByPurlNameType("NONEXISTENT", "none", "")
	if err != nil {
		t.Errorf("golang_projects.GetGolangUrlsByPurlNameType() error = %v", err)
	}
	if len(url.PurlName) > 0 {
		t.Errorf("golang_projects.GetGolangUrlsByPurlNameType() URLs found when none should be: %v", golangProjModel)
	}
	fmt.Printf("No Urls: %v\n", url)

	_, err = golangProjModel.GetGolangUrlsByPurlNameType("NONEXISTENT", "", "")
	if err == nil {
		t.Errorf("golang_projects.GetGolangUrlsByPurlNameType() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = golangProjModel.GetGolangUrlsByPurlNameType("", "", "")
	if err == nil {
		t.Errorf("golang_projects.GetUrlsByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = golangProjModel.GetGoLangUrlByPurlString("", "")
	if err == nil {
		t.Errorf("golang_projects.GetUrlsByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = golangProjModel.GetGoLangUrlByPurlString("rubbish-purl", "")
	if err == nil {
		t.Errorf("golang_projects.GetGoLangUrlByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	url, err = golangProjModel.GetGoLangUrlByPurlString("pkg:golang/google.golang.org/grpc", "")
	if err != nil {
		t.Errorf("golang_projects.GetGoLangUrlByPurlString() error = %v", err)
	}
	if len(url.PurlName) == 0 {
		t.Errorf("golang_projects.GetGoLangUrlByPurlString() No URLs returned from query")
	}
	fmt.Printf("Golang Url: %v\n", url)
}

func TestGolangProjectsSearchVersion(t *testing.T) {
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
	golangProjModel := NewGolangProjectModel(ctx, conn)

	url, err := golangProjModel.GetGolangUrlsByPurlNameTypeVersion("google.golang.org/grpc", "golang", "1.19.0")
	if err != nil {
		t.Errorf("golang_projects.GetGolangUrlsByPurlNameTypeVersion() error = %v", err)
	}
	if len(url.PurlName) == 0 {
		t.Errorf("golang_projects.GetGolangUrlsByPurlNameTypeVersion() No URLs returned from query")
	}
	fmt.Printf("Golang Url Version: %#v\n", url)

	url, err = golangProjModel.GetGoLangUrlByPurlString("pkg:golang/google.golang.org/grpc@v1.19.0", "")
	if err != nil {
		t.Errorf("golang_projects.GetGoLangUrlByPurlString() error = failed to find purl by version string")
	}
	fmt.Printf("Golang Url Version: %#v\n", url)

	_, err = golangProjModel.GetGolangUrlsByPurlNameTypeVersion("", "", "")
	if err == nil {
		t.Errorf("golang_projects.GetGolangUrlsByPurlNameTypeVersion() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = golangProjModel.GetGolangUrlsByPurlNameTypeVersion("NONEXISTENT", "", "")
	if err == nil {
		t.Errorf("golang_projects.GetGolangUrlsByPurlNameTypeVersion() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = golangProjModel.GetGolangUrlsByPurlNameTypeVersion("NONEXISTENT", "NONEXISTENT", "")
	if err == nil {
		t.Errorf("golang_projects.GetGolangUrlsByPurlNameTypeVersion() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}

	url, err = golangProjModel.GetGoLangUrlByPurlString("pkg:golang/google.golang.org/grpc", "22.22.22") // Shouldn't exist
	if err != nil {
		t.Errorf("golang_projects.GetGoLangUrlByPurlString() error = failed to find purl by version string")
	}
	if len(url.PurlName) > 0 {
		t.Errorf("golang_projects.GetGoLangUrlByPurlString() error = Found match, when we shouldn't: %v", url)
	}
}

func TestGolangProjectsSearchVersionRequirement(t *testing.T) {
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
	golangProjModel := NewGolangProjectModel(ctx, conn)

	url, err := golangProjModel.GetGoLangUrlByPurlString("pkg:golang/google.golang.org/grpc", ">0.0.4")
	if err != nil {
		t.Errorf("golang_projects.GetUrlsByPurlName() error = %v", err)
	}
	if len(url.PurlName) == 0 {
		t.Errorf("golang_projects.GetUrlsByPurlName() No URLs returned from query")
	}
	fmt.Printf("Golang Url Version: %#v\n", url)

	url, err = golangProjModel.GetGoLangUrlByPurlString("pkg:golang/google.golang.org/grpc", "<0.0.4>")
	if err != nil {
		t.Errorf("golang_projects.GetUrlsByPurlName() error = %v", err)
	}
	if len(url.PurlName) == 0 {
		t.Errorf("golang_projects.GetUrlsByPurlName() No URLs returned from query")
	}
}

func TestGolangProjectsSearchBadSql(t *testing.T) {
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
	golangProjModel := NewGolangProjectModel(ctx, conn)

	_, err = golangProjModel.GetGoLangUrlByPurlString("pkg:golang/google.golang.org/grpc", "")
	if err == nil {
		t.Errorf("golang_projects.GetGoLangUrlByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = golangProjModel.GetGoLangUrlByPurlString("pkg:golang/google.golang.org/grpc@1.19.0", "")
	if err == nil {
		t.Errorf("golang_projects.GetGoLangUrlByPurlString() error = did not get an error: %v", err)
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
}
