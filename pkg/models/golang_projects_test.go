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

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	pkggodevclient "github.com/guseggert/pkggodev-client"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	myconfig "scanoss.com/dependencies/pkg/config"
)

func TestGolangProjectUrlsSearch(t *testing.T) {
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
	golangProjModel := NewGolangProjectModel(ctx, s, db, conn, myConfig)

	url, err := golangProjModel.GetGolangUrlsByPurlNameType("google.golang.org/grpc", "golang", "")
	if err != nil {
		t.Errorf("FAILED: golang_projects.GetUrlsByPurlName() error = %v", err)
	}
	if len(url.PurlName) == 0 {
		t.Errorf("FAILED: golang_projects.GetGoLangURLByPurlString() No URLs returned from query")
	}
	fmt.Printf("Golang URL: %#v\n", url)

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
		t.Errorf("FAILED: golang_projects.GetURLsByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = golangProjModel.GetGoLangURLByPurlString("", "")
	if err == nil {
		t.Errorf("FAILED: golang_projects.GetURLsByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = golangProjModel.GetGoLangURLByPurlString("rubbish-purl", "")
	if err == nil {
		t.Errorf("FAILED: golang_projects.GetGoLangURLByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	url, err = golangProjModel.GetGoLangURLByPurlString("pkg:golang/google.golang.org/grpc", "")
	if err != nil {
		t.Errorf("FAILED: golang_projects.GetGoLangURLByPurlString() error = %v", err)
	}
	if len(url.PurlName) == 0 {
		t.Errorf("FAILED: golang_projects.GetGoLangURLByPurlString() No URLs returned from query")
	}
	fmt.Printf("Golang URL: %v\n", url)
}

func TestGolangProjectsSearchVersion(t *testing.T) {
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
		t.Fatalf("FAILED: failed to load SQL test data: %v", err)
	}
	myConfig, err := myconfig.NewServerConfig(nil)
	if err != nil {
		t.Fatalf("FAILED: failed to load Config: %v", err)
	}
	myConfig.Components.CommitMissing = true
	myConfig.Database.Trace = true
	golangProjModel := NewGolangProjectModel(ctx, s, db, conn, myConfig)

	url, err := golangProjModel.GetGolangUrlsByPurlNameTypeVersion("google.golang.org/grpc", "golang", "1.19.0")
	if err != nil {
		t.Errorf("FAILED: golang_projects.GetGolangUrlsByPurlNameTypeVersion() error = %v", err)
	}
	if len(url.PurlName) == 0 {
		t.Errorf("FAILED: golang_projects.GetGolangUrlsByPurlNameTypeVersion() No URLs returned from query")
	}
	fmt.Printf("Golang URL Version: %#v\n", url)

	url, err = golangProjModel.GetGoLangURLByPurlString("pkg:golang/google.golang.org/grpc@v1.19.0", "")
	if err != nil {
		t.Errorf("FAILED: golang_projects.GetGoLangURLByPurlString() error = failed to find purl by version string")
	}
	fmt.Printf("Golang URL Version: %#v\n", url)

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

	url, err = golangProjModel.GetGoLangURLByPurlString("pkg:golang/google.golang.org/grpc", "22.22.22") // Shouldn't exist
	if err != nil {
		t.Errorf("FAILED: golang_projects.GetGoLangURLByPurlString() error = failed to find purl by version string")
	}
	url, err = golangProjModel.GetGoLangURLByPurlString("pkg:golang/google.golang.org/grpc", "=v1.19.0")
	if err != nil {
		t.Errorf("FAILED: golang_projects.GetGoLangURLByPurlString() error = %v", err)
	}
	if len(url.PurlName) == 0 {
		t.Errorf("FAILED: golang_projects.GetGoLangURLByPurlString() No URLs returned from query")
	}
	fmt.Printf("Golang URL: %v\n", url)
	url, err = golangProjModel.GetGoLangURLByPurlString("pkg:golang/google.golang.org/grpc", "==v1.19.0")
	if err != nil {
		t.Errorf("FAILED: golang_projects.GetGoLangURLByPurlString() error = %v", err)
	}
	if len(url.PurlName) == 0 {
		t.Errorf("FAILED: golang_projects.GetGoLangURLByPurlString() No URLs returned from query")
	}
	fmt.Printf("Golang URL: %v\n", url)

	url, err = golangProjModel.GetGoLangURLByPurlString("pkg:golang/google.golang.org/grpc@1.7.0", "") // Should be missing license
	if err != nil {
		t.Errorf("FAILED: golang_projects.GetGoLangURLByPurlString() error = %v", err)
	}
	if len(url.License) == 0 {
		t.Errorf("FAILED: golang_projects.GetGoLangURLByPurlString() No URL License returned from query")
	}
	fmt.Printf("Golang URL: %v\n", url)
}

func TestGolangProjectsSearchVersionRequirement(t *testing.T) {
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
	golangProjModel := NewGolangProjectModel(ctx, s, db, conn, myConfig)

	url, err := golangProjModel.GetGoLangURLByPurlString("pkg:golang/google.golang.org/grpc", ">0.0.4")
	if err != nil {
		t.Errorf("FAILED: golang_projects.GetUrlsByPurlName() error = %v", err)
	}
	if len(url.PurlName) == 0 {
		t.Errorf("FAILED: golang_projects.GetUrlsByPurlName() No URLs returned from query")
	}
	fmt.Printf("Golang URL Version: %#v\n", url)

	url, err = golangProjModel.GetGoLangURLByPurlString("pkg:golang/google.golang.org/grpc", "v0.0.0-201910101010-s3333")
	if err != nil {
		t.Errorf("FAILED: golang_projects.GetUrlsByPurlName() error = %v", err)
	}
	if len(url.PurlName) == 0 {
		t.Errorf("FAILED: golang_projects.GetUrlsByPurlName() No URLs returned from query")
	}
	fmt.Printf("Golang URL Version: %#v\n", url)
}

func TestGolangPkgGoDev(t *testing.T) {
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
	golangProjModel := NewGolangProjectModel(ctx, s, db, conn, myConfig)

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
	fmt.Printf("Golang URL Version: %#v\n", url)

	url, err = golangProjModel.getLatestPkgGoDev("github.com/scanoss/papi", "golang", "v0.0.3")
	if err != nil {
		t.Errorf("FAILED: golang_projects.getLatestPkgGoDev() error = %v", err)
	}
	if len(url.PurlName) == 0 {
		t.Errorf("FAILED: golang_projects.getLatestPkgGoDev() No URLs returned from query")
	}
	fmt.Printf("Golang URL Version: %#v\n", url)

	var allUrl AllURL
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
	allUrl.MineID = 45
	err = golangProjModel.savePkg(allUrl, version, license, nil)
	if err == nil {
		t.Errorf("FAILED: golangProjModel.savePkg() error = did not get an error")
	}
	allUrl.Version = "v0.0.1"
	version.VersionName = "v0.0.1"
	version.ID = 5958021
	err = golangProjModel.savePkg(allUrl, version, license, nil)
	if err == nil {
		t.Errorf("FAILED: golangProjModel.savePkg() error = did not get an error")
	}
	license.LicenseName = "MIT"
	license.ID = 5614
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
	golangProjModel := NewGolangProjectModel(ctx, s, db, conn, myConfig)

	_, err = golangProjModel.GetGoLangURLByPurlString("pkg:golang/google.golang.org/grpc", "")
	if err == nil {
		t.Errorf("FAILED: golang_projects.GetGoLangURLByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = golangProjModel.GetGoLangURLByPurlString("pkg:golang/google.golang.org/grpc@1.19.0", "")
	if err == nil {
		t.Errorf("FAILED: golang_projects.GetGoLangURLByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = golangProjModel.getLatestPkgGoDev("github.com/scanoss/does-not-exist", "golang", "v0.0.99")
	if err == nil {
		t.Errorf("FAILED: golang_projects.getLatestPkgGoDev() error = did not get an error: %v", err)
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
}
