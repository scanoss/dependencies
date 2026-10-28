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
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	pkggodevclient "github.com/guseggert/pkggodev-client"
	"github.com/jmoiron/sqlx"
	componentHelper "github.com/scanoss/go-component-helper/componenthelper"
	"github.com/scanoss/go-grpc-helper/pkg/grpc/database"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	"scanoss.com/dependencies/pkg/resolver"
)

// newLicensesModel builds the licence lookup over the fixtures. describe stands in
// for pkg.go.dev; nil disables the live lookup.
func newLicensesModel(t *testing.T, db *sqlx.DB,
	describe func(pkggodevclient.DescribePackageRequest) (*pkggodevclient.Package, error),
) *ComponentLicensesModel {
	t.Helper()
	ctx := context.Background()
	var golang *GolangModel
	if describe != nil {
		golang = &GolangModel{ctx: ctx, s: zlog.S, lic: NewLicenseModel(ctx, zlog.S, db), describe: describe}
	}
	return NewComponentLicensesModel(ctx, zlog.S, db, database.NewDBSelectContext(zlog.S, db, nil, false),
		resolver.New(zlog.S, db), golang)
}

// resolve runs a purl through the resolver, as the use case does before the lookup.
func resolve(t *testing.T, db *sqlx.DB, purl, requirement string) (componentHelper.Component, resolver.ComponentKey) {
	t.Helper()
	components, ids := resolver.ResolveComponentsWithIDs(context.Background(), zlog.S, resolver.New(zlog.S, db),
		[]componentHelper.ComponentDTO{{Purl: purl, Requirement: requirement}}, 1)
	if len(components) != 1 {
		t.Fatalf("resolving %v gave %d components", purl, len(components))
	}
	return components[0], ids[0]
}

func TestGetURLsByComponent(t *testing.T) {
	db := sqliteSetup(t)
	defer zlog.SyncZap()
	model := newLicensesModel(t, db, nil)
	mit := License{LicenseName: "MIT", LicenseID: "MIT", IsSpdx: true}
	apache := License{LicenseName: "Apache 2.0", LicenseID: "Apache-2.0", IsSpdx: true}

	tests := []struct {
		name, purl, requirement string
		wantVersion             string
		wantURL                 string
		want                    []License
	}{
		{name: "exact version", purl: "pkg:npm/lodash", requirement: "4.17.0", wantVersion: "4.17.0",
			wantURL: "https://www.npmjs.com/package/lodash", want: []License{mit}},
		{name: "bare purl takes the latest", purl: "pkg:npm/lodash", wantVersion: "4.17.21",
			wantURL: "https://www.npmjs.com/package/lodash", want: []License{mit}},
		{name: "range", purl: "pkg:npm/electron-debug", requirement: "^3.1.0", wantVersion: "3.2.0",
			wantURL: "https://www.npmjs.com/package/electron-debug", want: []License{mit}},
		// The requested version is not published: the latest one answers.
		{name: "unknown version", purl: "pkg:npm/lodash", requirement: "9.9.9", wantVersion: "4.17.21",
			wantURL: "https://www.npmjs.com/package/lodash", want: []License{mit}},
		{name: "release without a licence", purl: "pkg:npm/sort-paths", requirement: "1.1.1", wantVersion: "1.1.1",
			wantURL: "https://www.npmjs.com/package/sort-paths"},
		// A release with two licences reports both, in licence id order.
		{name: "two licences", purl: "pkg:npm/dual", requirement: "1.0.0", wantVersion: "1.0.0",
			wantURL: "https://www.npmjs.com/package/dual", want: []License{mit, apache}},
		{name: "compound identifier", purl: "pkg:deb/debian/goffice", wantVersion: "0.10.52-4",
			want: []License{{LicenseName: "GPL-2 or GPL-3", LicenseID: "GPL-2.0-only/GPL-3.0-only"}}},
		// The export stores the scope percent-encoded; the resolver finds it either way.
		{name: "scoped npm, as the caller writes it", purl: "pkg:npm/@scope/pkg", wantVersion: "1.0.0", want: []License{mit}},
		{name: "scoped npm, encoded", purl: "pkg:npm/%40scope/pkg", wantVersion: "1.0.0",
			wantURL: "https://www.npmjs.com/package/%40scope/pkg", want: []License{mit}},
		{name: "unknown component", purl: "pkg:npm/does-not-exist", requirement: "1.0.0",
			wantURL: "https://www.npmjs.com/package/does-not-exist"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component, purlID := resolve(t, db, tt.purl, tt.requirement)
			got, err := model.GetURLsByComponent(component, purlID)
			if err != nil {
				t.Fatalf("GetURLsByComponent() error = %v", err)
			}
			if got.Version != tt.wantVersion {
				t.Errorf("version = %q, want %q", got.Version, tt.wantVersion)
			}
			if tt.wantURL != "" && got.URL != tt.wantURL {
				t.Errorf("url = %q, want %q", got.URL, tt.wantURL)
			}
			if !reflect.DeepEqual(got.Licenses, tt.want) {
				t.Errorf("licenses = %#v, want %#v", got.Licenses, tt.want)
			}
		})
	}
}

func TestGetURLsByComponentGolang(t *testing.T) {
	db := sqliteSetup(t)
	defer zlog.SyncZap()
	mit := License{LicenseName: "MIT", LicenseID: "MIT", IsSpdx: true}
	var asked []string
	live := func(req pkggodevclient.DescribePackageRequest) (*pkggodevclient.Package, error) {
		asked = append(asked, req.Package)
		if req.Package == "golang.org/x/net@v0.17.0" {
			return &pkggodevclient.Package{Version: "v0.17.0", License: "Apache 2.0", Repository: "go.googlesource.com/net"}, nil
		}
		return nil, errors.New("not found")
	}
	model := newLicensesModel(t, db, live)

	t.Run("in the knowledge base", func(t *testing.T) {
		asked = nil
		component, purlID := resolve(t, db, "pkg:golang/go.uber.org/zap", "v1.27.0")
		got, err := model.GetURLsByComponent(component, purlID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Version != "v1.27.0" || !reflect.DeepEqual(got.Licenses, []License{mit}) || len(asked) > 0 {
			t.Errorf("got %#v (asked pkg.go.dev %v)", got, asked)
		}
	})
	t.Run("live from pkg.go.dev", func(t *testing.T) {
		component, purlID := resolve(t, db, "pkg:golang/golang.org/x/net", "v0.17.0")
		got, err := model.GetURLsByComponent(component, purlID)
		if err != nil {
			t.Fatal(err)
		}
		want := []License{{LicenseName: "Apache 2.0", LicenseID: "Apache-2.0", IsSpdx: true}}
		if got.Version != "v0.17.0" || got.URL != "https://go.googlesource.com/net" || !reflect.DeepEqual(got.Licenses, want) {
			t.Errorf("got %#v", got)
		}
	})
	t.Run("github module from its repository", func(t *testing.T) {
		asked = nil
		component, purlID := resolve(t, db, "pkg:golang/github.com/gin-gonic/gin", "v1.9.1")
		got, err := model.GetURLsByComponent(component, purlID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Version != "v1.9.1" || !reflect.DeepEqual(got.Licenses, []License{mit}) {
			t.Errorf("got %#v", got)
		}
		if len(asked) > 0 {
			t.Errorf("a github.com module asked pkg.go.dev: %v", asked)
		}
	})
	t.Run("tag missing from the knowledge base", func(t *testing.T) {
		// The export carries golang modules at pseudo-versions only.
		for _, stmt := range []string{
			`INSERT INTO component (purl, purl_id) VALUES ('pkg:golang/github.com/gin-gonic/gin', 'uuid-golang-gin')`,
			`INSERT INTO component_version (purl_id, version) VALUES ('uuid-golang-gin', '1.12.1+0.20260626164816.34dac209ffb6')`,
			`INSERT INTO component_version_license (purl_id, version, license_id) VALUES ('uuid-golang-gin', '1.12.1+0.20260626164816.34dac209ffb6', 2)`,
		} {
			if _, err := db.Exec(stmt); err != nil {
				t.Fatal(err)
			}
		}
		component, purlID := resolve(t, db, "pkg:golang/github.com/gin-gonic/gin", "v1.9.1")
		got, err := model.GetURLsByComponent(component, purlID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Version != "v1.9.1" || !reflect.DeepEqual(got.Licenses, []License{mit}) {
			t.Errorf("got %#v, want the GitHub repository's v1.9.1", got)
		}
	})
	t.Run("nowhere", func(t *testing.T) {
		component, purlID := resolve(t, db, "pkg:golang/example.com/nothing", "v1.0.0")
		got, err := model.GetURLsByComponent(component, purlID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Licenses) > 0 {
			t.Errorf("got %#v", got)
		}
	})
}

// A pkg.go.dev that does not answer in time is given up on - both attempts share
// the one deadline, so the versionless retry is not made - and the lookup carries
// on to the knowledge base instead of holding the request.
func TestGetURLsByComponentGolangTimeout(t *testing.T) {
	db := sqliteSetup(t)
	defer zlog.SyncZap()
	var mu sync.Mutex
	var asked []string
	release := make(chan struct{})
	defer close(release)
	slow := func(req pkggodevclient.DescribePackageRequest) (*pkggodevclient.Package, error) {
		mu.Lock()
		asked = append(asked, req.Package)
		mu.Unlock()
		<-release
		return nil, errors.New("too late")
	}
	model := newLicensesModel(t, db, slow)
	model.golang.timeout = 50 * time.Millisecond

	component, key := resolve(t, db, "pkg:golang/golang.org/x/net", "v0.17.0")
	start := time.Now()
	got, err := model.GetURLsByComponent(component, key)
	if err != nil {
		t.Fatalf("GetURLsByComponent() error = %v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("the lookup took %v; it should give up after 50ms", elapsed)
	}
	if len(got.Licenses) > 0 {
		t.Errorf("licenses = %#v, want none: nothing else knows this module", got.Licenses)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(asked) != 1 {
		t.Errorf("asked pkg.go.dev %v; after a timeout the versionless retry must not run", asked)
	}
}
