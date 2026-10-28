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

package usecase

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/jmoiron/sqlx"
	"github.com/scanoss/go-component-helper/componenthelper"
	"github.com/scanoss/go-grpc-helper/pkg/grpc/domain"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	_ "modernc.org/sqlite"
	myconfig "scanoss.com/dependencies/pkg/config"
	"scanoss.com/dependencies/pkg/dtos"
	"scanoss.com/dependencies/pkg/models"
)

func TestDependencyUseCase(t *testing.T) {
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared S", err)
	}
	defer zlog.SyncZap()
	ctx := context.Background()
	ctx = ctxzap.ToContext(ctx, zlog.L)
	s := ctxzap.Extract(ctx).Sugar()
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer models.CloseDB(db)
	// An in-memory SQLite database is per connection, so the pool is capped at one
	// and the fixtures go through it: a held connection would starve the models.
	db.SetMaxOpenConns(1)
	err = models.LoadTestSQLData(db, ctx, nil)
	if err != nil {
		t.Fatalf("an error '%s' was not expected when loading test data", err)
	}
	var depRequestData = `{
  "depth": 1,
  "files": [
    {
      "file": "vue-dev/packages/weex-template-compiler/package.json",
      "purls": [
        {
          "purl": "pkg:npm/electron-debug",
          "requirement": "^3.1.0"
        },
        {
          "purl": "pkg:npm/isbinaryfile",
          "requirement": "^4.0.8"
        },
        {
          "purl": "pkg:npm/sort-paths",
          "requirement": "^1.1.1"
        },
        {
          "purl": "pkg:deb/debian/goffice",
          "requirement": ""
        }
      ]
    }
  ]
}
`
	myConfig, err := myconfig.NewServerConfig(nil)
	if err != nil {
		t.Fatalf("failed to load Config: %v", err)
	}
	depUc := NewDependencies(ctx, s, db, myConfig)
	requestDto, err := dtos.ParseDependencyInput(s, []byte(depRequestData))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when parsing input json", err)
	}
	dependencies, warn, err := depUc.GetDependencies(requestDto)
	if err != nil {
		t.Fatalf("an error '%s' was not expected when getting dependencies", err)
	}
	fmt.Printf("Dependency response (warn: %v): %+v\n", warn, dependencies)

	var depBadRequestData = `{
  "depth": 1,
  "files": [
    {
      "file": "vue-dev/packages/weex-template-compiler/package.json",
      "purls": [
        {
          "purl": "pkg:npm/",
          "requirement": "^3.1.0"
        },
        {
          "purl": "pkg:npm/isbinaryfile",
          "requirement": "^4.0.8"
        },
        {
          "purl": "",
          "requirement": "^1.1.1"
        }
      ]
    }
  ]
}
`
	requestDto, err = dtos.ParseDependencyInput(s, []byte(depBadRequestData))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when parsing input json", err)
	}
	dependencies, warn, err = depUc.GetDependencies(requestDto)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Mixed batch: invalid purls should have error statuses, valid ones may succeed or fail
	errorCount := 0
	for _, file := range dependencies.Files {
		for _, dep := range file.Dependencies {
			if dep.Status.StatusCode != "" && dep.Status.StatusCode != domain.Success {
				errorCount++
			}
		}
	}
	if errorCount == 0 {
		t.Errorf("expected at least one component-level error in mixed batch, got none")
	}
	fmt.Printf("Got %d component-level errors (warn: %v): %+v\n", errorCount, warn, dependencies)
}

func TestDependencyUseCaseOutput(t *testing.T) {
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared S", err)
	}
	defer zlog.SyncZap()
	ctx := context.Background()
	ctx = ctxzap.ToContext(ctx, zlog.L)
	s := ctxzap.Extract(ctx).Sugar()
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer models.CloseDB(db)
	// An in-memory SQLite database is per connection, so the pool is capped at one
	// and the fixtures go through it: a held connection would starve the models.
	db.SetMaxOpenConns(1)
	err = models.LoadTestSQLData(db, ctx, nil)
	if err != nil {
		t.Fatalf("an error '%s' was not expected when loading test data", err)
	}
	myConfig, err := myconfig.NewServerConfig(nil)
	if err != nil {
		t.Fatalf("failed to load Config: %v", err)
	}
	depUc := NewDependencies(ctx, s, db, myConfig)

	tests := []struct {
		name           string
		req            dtos.DependencyInput
		expectedOutput dtos.DependenciesOutput
	}{
		{
			name: "single npm dependency",
			req: dtos.DependencyInput{
				Depth: 1,
				Files: []dtos.DependencyFileInput{
					{
						File: "package.json",
						Purls: []componenthelper.ComponentDTO{
							{Purl: "pkg:npm/lodash", Requirement: "^4.17.0"},
						},
					},
				},
			},
			expectedOutput: dtos.DependenciesOutput{
				Component:   "lodash",
				Purl:        "pkg:npm/lodash",
				Version:     "4.17.0",
				Requirement: "^4.17.0",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dependencies, _, depErr := depUc.GetDependencies(tt.req)
			if depErr != nil {
				t.Fatalf("unexpected error: %v", depErr)
			}
			if len(dependencies.Files) == 0 || len(dependencies.Files[0].Dependencies) == 0 {
				t.Fatal("expected at least one dependency in response")
			}
			got := dependencies.Files[0].Dependencies[0]
			if got.Purl != tt.expectedOutput.Purl {
				t.Errorf("expected purl %v, got %v", tt.expectedOutput.Purl, got.Purl)
			}
			if got.Requirement != tt.expectedOutput.Requirement {
				t.Errorf("expected requirement %v, got %v", tt.expectedOutput.Requirement, got.Requirement)
			}
		})
	}
}

// TestDependencyUseCaseServing pins what the serving lookups answer, case by case.
func TestDependencyUseCaseServing(t *testing.T) {
	if err := zlog.NewSugaredDevLogger(); err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared S", err)
	}
	defer zlog.SyncZap()
	ctx := ctxzap.ToContext(context.Background(), zlog.L)
	s := ctxzap.Extract(ctx).Sugar()
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer models.CloseDB(db)
	db.SetMaxOpenConns(1)
	if err = models.LoadTestSQLData(db, ctx, nil); err != nil {
		t.Fatalf("an error '%s' was not expected when loading test data", err)
	}
	myConfig, err := myconfig.NewServerConfig(nil)
	if err != nil {
		t.Fatalf("failed to load Config: %v", err)
	}
	depUc := NewDependencies(ctx, s, db, myConfig)
	mit := dtos.DependencyLicense{Name: "MIT", SpdxID: "MIT", IsSpdx: true}

	tests := []struct {
		name, purl, requirement string
		wantVersion             string
		wantLicenses            []dtos.DependencyLicense
		wantStatus              domain.StatusCode
	}{
		{name: "exact version", purl: "pkg:npm/lodash", requirement: "4.17.0", wantVersion: "4.17.0",
			wantLicenses: []dtos.DependencyLicense{mit}, wantStatus: domain.Success},
		// The range resolves to 4.17.21, which the requirement's own version does not
		// match: reported as REQUIREMENT_NOT_MET with that version, as before.
		{name: "range", purl: "pkg:npm/lodash", requirement: "^4.17.0", wantVersion: "4.17.0",
			wantLicenses: []dtos.DependencyLicense{mit}, wantStatus: domain.RequirementNotMet},
		// Same version, other spelling of the leading "v": the caller's spelling is kept.
		{name: "v-prefixed request", purl: "pkg:npm/lodash", requirement: "v4.17.0", wantVersion: "v4.17.0",
			wantLicenses: []dtos.DependencyLicense{mit}, wantStatus: domain.Success},
		{name: "two licences", purl: "pkg:npm/dual", requirement: "1.0.0", wantVersion: "1.0.0",
			wantLicenses: []dtos.DependencyLicense{mit, {Name: "Apache 2.0", SpdxID: "Apache-2.0", IsSpdx: true}},
			wantStatus:   domain.Success},
		{name: "compound identifier split", purl: "pkg:deb/debian/goffice", wantVersion: "0.10.52-4",
			wantLicenses: []dtos.DependencyLicense{
				{Name: "GPL-2.0-only", SpdxID: "GPL-2.0-only", IsSpdx: true},
				{Name: "GPL-3.0-only", SpdxID: "GPL-3.0-only", IsSpdx: true},
			}, wantStatus: domain.Success},
		{name: "no licence", purl: "pkg:npm/sort-paths", requirement: "1.1.1", wantVersion: "1.1.1", wantLicenses: []dtos.DependencyLicense{},
			wantStatus: domain.NoInfo},
		{name: "unknown component", purl: "pkg:npm/does-not-exist", requirement: "1.0.0", wantStatus: domain.ComponentNotFound},
		{name: "invalid purl", purl: "notapurl", wantStatus: domain.InvalidPurl},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, _, depErr := depUc.GetDependencies(dtos.DependencyInput{Files: []dtos.DependencyFileInput{{
				File: "f", Purls: []componenthelper.ComponentDTO{{Purl: tt.purl, Requirement: tt.requirement}},
			}}})
			if depErr != nil {
				t.Fatalf("unexpected error: %v", depErr)
			}
			got := out.Files[0].Dependencies[0]
			if got.Version != tt.wantVersion {
				t.Errorf("version = %q, want %q", got.Version, tt.wantVersion)
			}
			if got.Status.StatusCode != tt.wantStatus {
				t.Errorf("status = %v (%v), want %v", got.Status.StatusCode, got.Status.Message, tt.wantStatus)
			}
			if !reflect.DeepEqual(got.Licenses, tt.wantLicenses) {
				t.Errorf("licenses = %#v, want %#v", got.Licenses, tt.wantLicenses)
			}
		})
	}
}
