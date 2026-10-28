// SPDX-License-Identifier: GPL-2.0-or-later
/*
 * Copyright (C) 2018-2026 SCANOSS.COM
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
	"reflect"
	"testing"

	"github.com/jmoiron/sqlx"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
)

func TestLicensesSearch(t *testing.T) {
	db := sqliteSetup(t)
	defer zlog.SyncZap()
	licenseModel := NewLicenseModel(context.Background(), zlog.S, db)

	tests := []struct {
		name string
		want License
	}{
		{name: "MIT", want: License{LicenseName: "MIT", LicenseID: "MIT", IsSpdx: true}},
		// The name is the catalogue's long_name, not the SPDX identifier.
		{name: "Apache 2.0", want: License{LicenseName: "Apache 2.0", LicenseID: "Apache-2.0", IsSpdx: true}},
		// SPDX status is registry membership, not a stored flag.
		{name: "Custom Licence", want: License{LicenseName: "Custom Licence", LicenseID: "LicenseRef-custom", IsSpdx: false}},
		{name: "Unknown License", want: License{}},
		{name: "", want: License{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := licenseModel.GetLicenseByName(tt.name)
			if err != nil {
				t.Fatalf("GetLicenseByName(%q) error = %v", tt.name, err)
			}
			if got != tt.want {
				t.Errorf("GetLicenseByName(%q) = %#v, want %#v", tt.name, got, tt.want)
			}
		})
	}
}

// A missing table is a query failure, not "no licence".
func TestLicensesSearchBadSql(t *testing.T) {
	if err := zlog.NewSugaredDevLogger(); err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer zlog.SyncZap()
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer CloseDB(db)
	if _, err = NewLicenseModel(context.Background(), zlog.S, db).GetLicenseByName("MIT"); err == nil {
		t.Error("GetLicenseByName() without a license table did not fail")
	}
}

func TestCleanseLicenseName(t *testing.T) {
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer zlog.SyncZap()
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "MIT",
			input: "MIT",
			want:  "MIT",
		},
		{
			name:  "Apache 2",
			input: " Apache 2.0 ",
			want:  "Apache 2.0",
		},
		{
			name:  "Apache/MIT",
			input: " Apache 2.0, 	MIT		",
			want:  "Apache 2.0; MIT",
		},
		{
			name:  "Empty String",
			input: "",
			want:  "",
		},
		{
			name:    "Banned prefixes",
			input:   "see something else",
			want:    "",
			wantErr: true,
		},
		{
			name:    "Banned suffixes",
			input:   "license name.html",
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, cErr := CleanseLicenseName(tt.input)
			if (cErr != nil) != tt.wantErr {
				t.Errorf("licenses.CleanseLicenseName() error = %v, wantErr %v", cErr, tt.wantErr)
				return
			}
			if cErr == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("licenses.CleanseLicenseName() = %v, want %v", got, tt.want)
			}
		})
	}
}
