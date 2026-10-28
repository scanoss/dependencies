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
	"sort"
	"testing"

	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
)

func TestDependenciesSearch(t *testing.T) {
	db := sqliteSetup(t)
	defer zlog.SyncZap()
	model := NewDependencyModel(context.Background(), zlog.S, db)

	if _, err := model.GetDependencies("vue-phone", "1.0.8", "notExists"); err == nil {
		t.Error("expected an error for an unsupported ecosystem")
	}

	tests := []struct {
		name      string
		purlName  string
		version   string
		ecosystem string
		want      []UnresolvedDependency
	}{
		{
			// Every scope comes back (dev and peer too, as the mining arrays held
			// them), the namespace is kept, and an empty requirement is passed on.
			name: "every scope", purlName: "vue-phone", version: "1.0.9", ecosystem: "npm",
			want: []UnresolvedDependency{
				{Purl: "@types/node", Requirement: "~18.0.0"},
				{Purl: "isbinaryfile", Requirement: ""},
				{Purl: "lodash", Requirement: "^4.17.0"},
			},
		},
		{name: "release without dependencies", purlName: "vue-phone", version: "1.0.8", ecosystem: "npm", want: []UnresolvedDependency{}},
		{name: "unknown component", purlName: "does-not-exist", version: "1.0.0", ecosystem: "npm", want: []UnresolvedDependency{}},
		{
			// The request names a scoped package as @scope/pkg; the export stores it
			// percent-encoded.
			name: "scoped npm package", purlName: "@scope/pkg", version: "1.0.0", ecosystem: "npm",
			want: []UnresolvedDependency{{Purl: "@scope/other", Requirement: "2.0.0"}},
		},
		{
			name: "maven group and artifact", purlName: "org.example/lib", version: "1.0.0", ecosystem: "maven",
			want: []UnresolvedDependency{{Purl: "org.example/core", Requirement: "2.0.0"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := model.GetDependencies(tt.purlName, tt.version, tt.ecosystem)
			if err != nil {
				t.Fatalf("GetDependencies() error = %v", err)
			}
			sort.Slice(got, func(i, j int) bool { return got[i].Purl < got[j].Purl })
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDependencies() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

// A walk derives versions from requirements, which carry no "v"; the release may
// publish one. It also pins the crates ecosystem to the export's cargo purl type.
func TestDependenciesSearchTogglesVersionPrefix(t *testing.T) {
	db := sqliteSetup(t)
	defer zlog.SyncZap()
	if _, err := db.Exec(`INSERT INTO component (purl, purl_id) VALUES ('pkg:cargo/vtagged', 'uuid-vtagged')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO component_version_dependency (purl_id, version, dep_purl, dep_version)
		VALUES ('uuid-vtagged', 'v1.0.0', 'pkg:cargo/serde', '1.0')`); err != nil {
		t.Fatal(err)
	}
	got, err := NewDependencyModel(context.Background(), zlog.S, db).GetDependencies("vtagged", "1.0.0", "crates")
	if err != nil {
		t.Fatalf("GetDependencies() error = %v", err)
	}
	if want := []UnresolvedDependency{{Purl: "serde", Requirement: "1.0"}}; !reflect.DeepEqual(got, want) {
		t.Errorf("GetDependencies() = %#v, want %#v", got, want)
	}
}

func TestPackageIdentifier(t *testing.T) {
	tests := map[string]string{
		"pkg:npm/lodash":                   "lodash",
		"pkg:npm/%40types/node":            "@types/node",
		"pkg:maven/org.example/core@2.0.0": "org.example/core",
		"pkg:composer/laravel/framework":   "laravel/framework",
	}
	for purl, want := range tests {
		if got, err := packageIdentifier(purl); err != nil || got != want {
			t.Errorf("packageIdentifier(%q) = %q, %v; want %q", purl, got, err, want)
		}
	}
	if _, err := packageIdentifier("not a purl"); err == nil {
		t.Error("packageIdentifier() accepted something that is not a purl")
	}
}
