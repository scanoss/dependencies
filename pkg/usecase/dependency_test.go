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
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"scanoss.com/dependencies/pkg/dtos"
	zlog "scanoss.com/dependencies/pkg/logger"
	"scanoss.com/dependencies/pkg/models"
	"testing"
)

func TestDependencyUseCase(t *testing.T) {
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
	defer models.CloseDB(db)
	conn, err := db.Connx(ctx) // Get a connection from the pool
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer models.CloseConn(conn)
	err = models.LoadTestSqlData(db, ctx, conn)
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
        }
      ]
    }
  ]
}
`
	depUc := NewDependencies(ctx, conn)
	requestDto, err := dtos.ParseDependencyInput([]byte(depRequestData))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when parsing input json", err)
	}
	dependencies, err := depUc.GetDependencies(requestDto)
	if err != nil {
		t.Fatalf("an error '%s' was not expected when getting dependencies", err)
	}
	fmt.Printf("Dependency response: %+v\n", dependencies)

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
	requestDto, err = dtos.ParseDependencyInput([]byte(depBadRequestData))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when parsing input json", err)
	}
	dependencies, err = depUc.GetDependencies(requestDto)
	if err == nil {
		t.Fatalf("did not get an expected error: %v", dependencies)
	}
	fmt.Printf("Got expected error: %+v\n", err)
}
