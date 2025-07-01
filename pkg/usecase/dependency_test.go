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
	"testing"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/jmoiron/sqlx"
	"github.com/scanoss/go-models-helper/pkg/models"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	_ "modernc.org/sqlite"
	"scanoss.com/dependencies/pkg/dtos"
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
	conn, err := db.Connx(ctx) // Get a connection from the pool
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer models.CloseConn(conn)
	err = models.LoadTestSQLData(db, ctx, conn)
	if err != nil {
		t.Fatalf("an error '%s' was not expected when loading test data", err)
	}
	// Create models for testing
	modelConfig := models.ModelConfig{
		CommitMissing: false,
		Trace:         false,
	}
	scanossModels := models.NewScanossModels(ctx, s, conn, modelConfig)
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
	depUc := NewDependencies(ctx, scanossModels)
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
	if err == nil {
		t.Fatalf("did not get an expected error: %v", dependencies)
	}
	fmt.Printf("Got expected error (warn: %v): %+v\n", warn, err)
}
