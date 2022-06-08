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
	pkggodevclient "github.com/guseggert/pkggodev-client"
	"github.com/jmoiron/sqlx"
	myconfig "scanoss.com/dependencies/pkg/config"
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
	myConfig, err := myconfig.NewServerConfig(nil)
	if err != nil {
		t.Fatalf("failed to load Config: %v", err)
	}
	myConfig.Components.CommitMissing = true
	golangProjModel := NewGolangProjectModel(ctx, conn, myConfig)

	url, err := golangProjModel.GetGolangUrlsByPurlNameType("google.golang.org/grpc", "golang", "")
	if err != nil {
		t.Errorf("FAILED: golang_projects.GetUrlsByPurlName() error = %v", err)
	}
	if len(url.PurlName) == 0 {
		t.Errorf("FAILED: golang_projects.GetGoLangUrlByPurlString() No URLs returned from query")
	}
	fmt.Printf("Golang Url: %#v\n", url)

	url, err = golangProjModel.GetGolangUrlsByPurlNameType("NONEXISTENT", "none", "")
	if err != nil {
		t.Errorf("FAILED: golang_projects.GetGolangUrlsByPurlNameType() error = %v", err)
	}
	if len(url.PurlName) > 0 {
		t.Errorf("FAILED: golang_projects.GetGolangUrlsByPurlNameType() URLs found when none should be: %v", golangProjModel)
	}
	fmt.Printf("No Urls: %v\n", url)

	_, err = golangProjModel.GetGolangUrlsByPurlNameType("NONEXISTENT", "", "")
	if err == nil {
		t.Errorf("FAILED: golang_projects.GetGolangUrlsByPurlNameType() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = golangProjModel.GetGolangUrlsByPurlNameType("", "", "")
	if err == nil {
		t.Errorf("FAILED: golang_projects.GetUrlsByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = golangProjModel.GetGoLangUrlByPurlString("", "")
	if err == nil {
		t.Errorf("FAILED: golang_projects.GetUrlsByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = golangProjModel.GetGoLangUrlByPurlString("rubbish-purl", "")
	if err == nil {
		t.Errorf("FAILED: golang_projects.GetGoLangUrlByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	url, err = golangProjModel.GetGoLangUrlByPurlString("pkg:golang/google.golang.org/grpc", "")
	if err != nil {
		t.Errorf("FAILED: golang_projects.GetGoLangUrlByPurlString() error = %v", err)
	}
	if len(url.PurlName) == 0 {
		t.Errorf("FAILED: golang_projects.GetGoLangUrlByPurlString() No URLs returned from query")
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
	myConfig, err := myconfig.NewServerConfig(nil)
	if err != nil {
		t.Fatalf("failed to load Config: %v", err)
	}
	myConfig.Components.CommitMissing = true
	golangProjModel := NewGolangProjectModel(ctx, conn, myConfig)

	url, err := golangProjModel.GetGolangUrlsByPurlNameTypeVersion("google.golang.org/grpc", "golang", "1.19.0")
	if err != nil {
		t.Errorf("FAILED: golang_projects.GetGolangUrlsByPurlNameTypeVersion() error = %v", err)
	}
	if len(url.PurlName) == 0 {
		t.Errorf("FAILED: golang_projects.GetGolangUrlsByPurlNameTypeVersion() No URLs returned from query")
	}
	fmt.Printf("Golang Url Version: %#v\n", url)

	url, err = golangProjModel.GetGoLangUrlByPurlString("pkg:golang/google.golang.org/grpc@v1.19.0", "")
	if err != nil {
		t.Errorf("FAILED: golang_projects.GetGoLangUrlByPurlString() error = failed to find purl by version string")
	}
	fmt.Printf("Golang Url Version: %#v\n", url)

	_, err = golangProjModel.GetGolangUrlsByPurlNameTypeVersion("", "", "")
	if err == nil {
		t.Errorf("FAILED: golang_projects.GetGolangUrlsByPurlNameTypeVersion() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = golangProjModel.GetGolangUrlsByPurlNameTypeVersion("NONEXISTENT", "", "")
	if err == nil {
		t.Errorf("FAILED: golang_projects.GetGolangUrlsByPurlNameTypeVersion() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = golangProjModel.GetGolangUrlsByPurlNameTypeVersion("NONEXISTENT", "NONEXISTENT", "")
	if err == nil {
		t.Errorf("FAILED: golang_projects.GetGolangUrlsByPurlNameTypeVersion() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}

	url, err = golangProjModel.GetGoLangUrlByPurlString("pkg:golang/google.golang.org/grpc", "22.22.22") // Shouldn't exist
	if err != nil {
		t.Errorf("FAILED: golang_projects.GetGoLangUrlByPurlString() error = failed to find purl by version string")
	}
	if len(url.PurlName) > 0 {
		t.Errorf("golang_projects.GetGoLangUrlByPurlString() error = Found match, when we shouldn't: %v", url)
	}
	url, err = golangProjModel.GetGoLangUrlByPurlString("pkg:golang/google.golang.org/grpc", "=v1.19.0")
	if err != nil {
		t.Errorf("FAILED: golang_projects.GetGoLangUrlByPurlString() error = %v", err)
	}
	if len(url.PurlName) == 0 {
		t.Errorf("FAILED: golang_projects.GetGoLangUrlByPurlString() No URLs returned from query")
	}
	fmt.Printf("Golang Url: %v\n", url)
	url, err = golangProjModel.GetGoLangUrlByPurlString("pkg:golang/google.golang.org/grpc", "==v1.19.0")
	if err != nil {
		t.Errorf("FAILED: golang_projects.GetGoLangUrlByPurlString() error = %v", err)
	}
	if len(url.PurlName) == 0 {
		t.Errorf("FAILED: golang_projects.GetGoLangUrlByPurlString() No URLs returned from query")
	}
	fmt.Printf("Golang Url: %v\n", url)
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
	myConfig, err := myconfig.NewServerConfig(nil)
	if err != nil {
		t.Fatalf("failed to load Config: %v", err)
	}
	myConfig.Components.CommitMissing = true
	golangProjModel := NewGolangProjectModel(ctx, conn, myConfig)

	url, err := golangProjModel.GetGoLangUrlByPurlString("pkg:golang/google.golang.org/grpc", ">0.0.4")
	if err != nil {
		t.Errorf("FAILED: golang_projects.GetUrlsByPurlName() error = %v", err)
	}
	if len(url.PurlName) == 0 {
		t.Errorf("FAILED: golang_projects.GetUrlsByPurlName() No URLs returned from query")
	}
	fmt.Printf("Golang Url Version: %#v\n", url)

	url, err = golangProjModel.GetGoLangUrlByPurlString("pkg:golang/google.golang.org/grpc", "v0.0.0-201910101010-s3333")
	if err != nil {
		t.Errorf("FAILED: golang_projects.GetUrlsByPurlName() error = %v", err)
	}
	if len(url.PurlName) == 0 {
		t.Errorf("FAILED: golang_projects.GetUrlsByPurlName() No URLs returned from query")
	}
	fmt.Printf("Golang Url Version: %#v\n", url)
}

func TestGolangPkgGoDev(t *testing.T) {
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
	myConfig, err := myconfig.NewServerConfig(nil)
	if err != nil {
		t.Fatalf("failed to load Config: %v", err)
	}
	myConfig.Components.CommitMissing = true
	golangProjModel := NewGolangProjectModel(ctx, conn, myConfig)

	_, _, _, err = golangProjModel.queryPkgGoDev("", "")
	if err == nil {
		t.Errorf("FAILED: golang_projects.queryPkgGoDev() error = did not get an error")
	}

	url, err := golangProjModel.getLatestPkgGoDev("google.golang.org/grpc", "golang", "v0.0.0-201910101010-s3333")
	if err != nil {
		t.Errorf("FAILED: golang_projects.getLatestPkgGoDev() error = %v", err)
	}
	if len(url.PurlName) == 0 {
		t.Errorf("FAILED: golang_projects.getLatestPkgGoDev() No URLs returned from query")
	}
	fmt.Printf("Golang Url Version: %#v\n", url)

	url, err = golangProjModel.getLatestPkgGoDev("github.com/scanoss/papi", "golang", "v0.0.3")
	if err != nil {
		t.Errorf("FAILED: golang_projects.getLatestPkgGoDev() error = %v", err)
	}
	if len(url.PurlName) == 0 {
		t.Errorf("FAILED: golang_projects.getLatestPkgGoDev() No URLs returned from query")
	}
	fmt.Printf("Golang Url Version: %#v\n", url)

	var allUrl AllUrl
	var license License
	var version Version
	fmt.Printf("SavePkg: %#v - %#v - %#v", allUrl, license, version)
	err = golangProjModel.savePkg(allUrl, version, license, nil)
	if err == nil {
		t.Errorf("FAILED: golangProjModel.savePkg() error = did not get an error")
	}
	allUrl.PurlName = "github.com/scanoss/papi"
	err = golangProjModel.savePkg(allUrl, version, license, nil)
	if err == nil {
		t.Errorf("FAILED: golangProjModel.savePkg() error = did not get an error")
	}
	allUrl.MineId = 45
	err = golangProjModel.savePkg(allUrl, version, license, nil)
	if err == nil {
		t.Errorf("FAILED: golangProjModel.savePkg() error = did not get an error")
	}
	allUrl.Version = "v0.0.1"
	version.VersionName = "v0.0.1"
	version.Id = 5958021
	err = golangProjModel.savePkg(allUrl, version, license, nil)
	if err == nil {
		t.Errorf("FAILED: golangProjModel.savePkg() error = did not get an error")
	}
	license.LicenseName = "MIT"
	license.Id = 5614
	err = golangProjModel.savePkg(allUrl, version, license, nil)
	if err == nil {
		t.Errorf("FAILED: golangProjModel.savePkg() error = did not get an error")
	}
	var comp pkggodevclient.Package
	comp.Package = "github.com/scanoss/papi"
	comp.IsPackage = true
	comp.IsModule = true
	comp.Version = "v0.0.1"
	comp.License = "MIT"
	comp.HasRedistributableLicense = true
	comp.HasStableVersion = true
	comp.HasTaggedVersion = true
	comp.HasValidGoModFile = true
	comp.Repository = "github.com/scanoss/papi"
	err = golangProjModel.savePkg(allUrl, version, license, &comp)
	if err != nil {
		t.Errorf("FAILED: golangProjModel.savePkg() error = %v", err)
	}
	allUrl.Version = "v0.0.2"
	version.VersionName = "v0.0.2"
	comp.Version = "v0.0.2"
	err = golangProjModel.savePkg(allUrl, version, license, &comp)
	if err != nil {
		t.Errorf("FAILED: golangProjModel.savePkg() error = %v", err)
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
	myConfig, err := myconfig.NewServerConfig(nil)
	if err != nil {
		t.Fatalf("failed to load Config: %v", err)
	}
	myConfig.Components.CommitMissing = true
	golangProjModel := NewGolangProjectModel(ctx, conn, myConfig)

	_, err = golangProjModel.GetGoLangUrlByPurlString("pkg:golang/google.golang.org/grpc", "")
	if err == nil {
		t.Errorf("FAILED: golang_projects.GetGoLangUrlByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = golangProjModel.GetGoLangUrlByPurlString("pkg:golang/google.golang.org/grpc@1.19.0", "")
	if err == nil {
		t.Errorf("FAILED: golang_projects.GetGoLangUrlByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = golangProjModel.getLatestPkgGoDev("github.com/scanoss/papi", "golang", "v0.0.99")
	if err == nil {
		t.Errorf("FAILED: golang_projects.getLatestPkgGoDev() error = did not get an error: %v", err)
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}

}
