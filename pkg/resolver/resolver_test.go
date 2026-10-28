// SPDX-License-Identifier: GPL-2.0-or-later
/*
 * Copyright (C) 2018-2025 SCANOSS.COM
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

package resolver

import (
	"context"
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	_ "modernc.org/sqlite"
)

// servingDDL is the slice of the serving schema the resolver reads.
const servingDDL = `
CREATE TABLE component (purl TEXT, vendor TEXT, component TEXT, source_purl_id TEXT, purl_id TEXT);
CREATE TABLE component_version (purl_id TEXT, version TEXT, version_name TEXT, version_semver TEXT);
`

// newTestResolver builds a resolver over an in-memory serving schema holding one
// npm package linked to the GitHub repository it was built from, plus a component
// with no source link at all.
func newTestResolver(t *testing.T) *Resolver {
	t.Helper()
	// One connection only: an in-memory SQLite database is per-connection, so a
	// second would be empty.
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open the test database: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	if _, err = db.Exec(servingDDL); err != nil {
		t.Fatalf("failed to create the serving schema: %v", err)
	}
	stmts := []string{
		`INSERT INTO component (purl, purl_id, source_purl_id) VALUES ('pkg:npm/log4js', 'uuid-npm', 'uuid-github')`,
		`INSERT INTO component (purl, purl_id, source_purl_id) VALUES ('pkg:github/log4js-node/log4js-node', 'uuid-github', NULL)`,
		`INSERT INTO component (purl, purl_id, source_purl_id) VALUES ('pkg:npm/orphan', 'uuid-orphan', NULL)`,
		`INSERT INTO component (purl, purl_id, source_purl_id) VALUES ('pkg:npm/dangling', 'uuid-dangling', 'uuid-missing')`,
		`INSERT INTO component_version (purl_id, version) VALUES ('uuid-npm', '1.0.0')`,
		`INSERT INTO component_version (purl_id, version) VALUES ('uuid-npm', '2.0.0')`,
		`INSERT INTO component_version (purl_id, version) VALUES ('uuid-npm', '3.1.0')`,
		`INSERT INTO component_version (purl_id, version) VALUES ('uuid-github', '3.1.0')`,
		// A component whose versions carry the "v" prefix, to pin the other direction.
		`INSERT INTO component (purl, purl_id, source_purl_id) VALUES ('pkg:npm/vtagged', 'uuid-vtagged', NULL)`,
		`INSERT INTO component_version (purl_id, version) VALUES ('uuid-vtagged', 'v1.0.0')`,
	}
	for _, stmt := range stmts {
		if _, err = db.Exec(stmt); err != nil {
			t.Fatalf("failed to seed %q: %v", stmt, err)
		}
	}
	return New(zap.NewNop().Sugar(), db)
}

func TestResolvePinsVersion(t *testing.T) {
	r := newTestResolver(t)
	ctx := context.Background()

	tests := []struct {
		name        string
		purl        string
		requirement string
		wantVersion string
		wantStatus  string
	}{
		{
			// A bare purl resolves to the highest published version, which is what the
			// mining resolution did and what callers of this endpoint depend on.
			name: "no requirement takes the highest version",
			purl: "pkg:npm/log4js", wantVersion: "3.1.0", wantStatus: StatusSuccess,
		},
		{
			name: "exact version that exists",
			purl: "pkg:npm/log4js", requirement: "2.0.0",
			wantVersion: "2.0.0", wantStatus: StatusSuccess,
		},
		{
			// Reported, not dropped: the caller decides whether to keep going, because
			// OSV may still carry data for a version the knowledge base does not.
			name: "exact version that does not exist",
			purl: "pkg:npm/log4js", requirement: "9.9.9",
			wantVersion: "9.9.9", wantStatus: StatusVersionNotFound,
		},
		{
			name: "semver range takes the highest match",
			purl: "pkg:npm/log4js", requirement: "<3.0.0",
			wantVersion: "2.0.0", wantStatus: StatusSuccess,
		},
		{
			name: "semver range matching nothing",
			purl: "pkg:npm/log4js", requirement: ">9.0.0",
			wantVersion: "", wantStatus: StatusVersionNotFound,
		},
		{
			name:        "version in the purl itself",
			purl:        "pkg:npm/log4js@1.0.0",
			wantVersion: "1.0.0", wantStatus: StatusSuccess,
		},
		{
			// A leading "v" is a labelling convention, not part of the version. The
			// mining lookup matched both spellings and answered with the stored one.
			name: "v-prefixed request matches the unprefixed version",
			purl: "pkg:npm/log4js", requirement: "v2.0.0",
			wantVersion: "2.0.0", wantStatus: StatusSuccess,
		},
		{
			name: "unprefixed request matches a v-prefixed version",
			purl: "pkg:npm/vtagged", requirement: "1.0.0",
			wantVersion: "v1.0.0", wantStatus: StatusSuccess,
		},
		{
			name:        "component with no versions",
			purl:        "pkg:npm/orphan",
			wantVersion: "", wantStatus: StatusVersionNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.Resolve(ctx, tt.purl, tt.requirement)
			if err != nil {
				t.Fatalf("Resolve(%q, %q) errored: %v", tt.purl, tt.requirement, err)
			}
			if got.Version != tt.wantVersion {
				t.Errorf("version = %q, want %q", got.Version, tt.wantVersion)
			}
			if got.StatusCode != tt.wantStatus {
				t.Errorf("status = %q, want %q", got.StatusCode, tt.wantStatus)
			}
		})
	}
}

func TestResolveErrors(t *testing.T) {
	r := newTestResolver(t)
	ctx := context.Background()

	tests := []struct {
		name string
		purl string
		want error
	}{
		{"empty purl", "", ErrEmptyPurl},
		{"unparseable purl", "not-a-purl", ErrInvalidPurl},
		{"unknown component", "pkg:npm/does-not-exist", ErrComponentNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := r.Resolve(ctx, tt.purl, ""); !errors.Is(err, tt.want) {
				t.Errorf("Resolve(%q) error = %v, want %v", tt.purl, err, tt.want)
			}
		})
	}

	t.Run("no database configured", func(t *testing.T) {
		if _, err := New(zap.NewNop().Sugar(), nil).Resolve(ctx, "pkg:npm/log4js", ""); !errors.Is(err, ErrNoDatabase) {
			t.Errorf("want ErrNoDatabase, got %v", err)
		}
	})
}

func TestSourcePurlFor(t *testing.T) {
	r := newTestResolver(t)
	ctx := context.Background()

	t.Run("linked source is resolved", func(t *testing.T) {
		got := r.SourcePurlFor(ctx, "pkg:npm/log4js")
		if got == nil {
			t.Fatal("want the linked source purl, got nil")
		}
		if got.Purl != "pkg:github/log4js-node/log4js-node" {
			t.Errorf("source purl = %q", got.Purl)
		}
		if got.PurlType != "github" {
			t.Errorf("source purl type = %q, want github", got.PurlType)
		}
	})

	// A component with no link, a link pointing at a row that is not there, and an
	// unknown component all mean the same thing to the caller: no fallback target.
	for name, purl := range map[string]string{
		"no source link":    "pkg:npm/orphan",
		"dangling link":     "pkg:npm/dangling",
		"unknown component": "pkg:npm/does-not-exist",
	} {
		t.Run(name, func(t *testing.T) {
			if got := r.SourcePurlFor(ctx, purl); got != nil {
				t.Errorf("want nil, got %+v", got)
			}
		})
	}
}
