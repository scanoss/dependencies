// SPDX-License-Identifier: GPL-2.0-or-later
/*
 * Copyright (C) 2025 SCANOSS.COM
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
	"testing"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	_ "modernc.org/sqlite"
)

// TestResolveStoredSpellings pins each way the export spells a component key.
// Every row here is a real spelling from the serving export; before the candidate
// keys, all of them except the plain one answered COMPONENT_NOT_FOUND.
func TestResolveStoredSpellings(t *testing.T) {
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open the test database: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if _, err = db.Exec(servingDDL); err != nil {
		t.Fatalf("failed to create the serving schema: %v", err)
	}
	stored := map[string]string{
		"uuid-scoped":   "pkg:npm/%40angular/core",
		"uuid-qualfied": "pkg:generic/orange-fashion?repository_url=code.google.com",
		"uuid-plus":     "pkg:deb/debian/g++-7-multilib-mipsel-linux-gnu",
		"uuid-encplus":  "pkg:videolan/liba52-0-0.7.5%2bsvn613-16.1.x86_64.rpm",
		"uuid-plain":    "pkg:github/scanoss/engine",
	}
	for id, purl := range stored {
		if _, err = db.Exec(`INSERT INTO component (purl, purl_id) VALUES (?, ?)`, purl, id); err != nil {
			t.Fatalf("failed to seed %q: %v", purl, err)
		}
		if _, err = db.Exec(`INSERT INTO component_version (purl_id, version) VALUES (?, '1.0.0')`, id); err != nil {
			t.Fatalf("failed to seed a version for %q: %v", purl, err)
		}
	}
	r := New(zap.NewNop().Sugar(), db)

	tests := []struct {
		purl   string
		wantID string
	}{
		{"pkg:npm/%40angular/core", "uuid-scoped"},
		{"pkg:npm/@angular/core", "uuid-scoped"},
		{"pkg:npm/%40angular/core@1.0.0", "uuid-scoped"},
		{"pkg:npm/@angular/core@1.0.0", "uuid-scoped"},
		{"pkg:generic/orange-fashion?repository_url=code.google.com", "uuid-qualfied"},
		{"pkg:generic/orange-fashion@1.0.0?repository_url=code.google.com", "uuid-qualfied"},
		{"pkg:deb/debian/g++-7-multilib-mipsel-linux-gnu", "uuid-plus"},
		{"pkg:deb/debian/g%2B%2B-7-multilib-mipsel-linux-gnu", "uuid-plus"},
		{"pkg:videolan/liba52-0-0.7.5%2bsvn613-16.1.x86_64.rpm", "uuid-encplus"},
		{"pkg:github/scanoss/engine", "uuid-plain"},
		{"pkg:github/scanoss/engine#src/main.c", "uuid-plain"},
	}
	for _, tt := range tests {
		t.Run(tt.purl, func(t *testing.T) {
			got, resolveErr := r.Resolve(context.Background(), tt.purl, "")
			if resolveErr != nil {
				t.Fatalf("Resolve(%q) error = %v", tt.purl, resolveErr)
			}
			if got.PurlID != tt.wantID {
				t.Errorf("Resolve(%q) PurlID = %q, want %q", tt.purl, got.PurlID, tt.wantID)
			}
			if got.Purl != stored[tt.wantID] {
				t.Errorf("Resolve(%q) Purl = %q, want the stored %q", tt.purl, got.Purl, stored[tt.wantID])
			}
		})
	}
}

func TestRawKey(t *testing.T) {
	tests := []struct{ purl, key, qualifiers string }{
		{"pkg:npm/@angular/core@17.0.0", "pkg:npm/@angular/core", ""},
		{"pkg:npm/%40angular/core", "pkg:npm/%40angular/core", ""},
		{"pkg:generic/x@1.0?repository_url=code.google.com/p/x", "pkg:generic/x", "repository_url=code.google.com/p/x"},
		{"pkg:github/a/b@v1#sub/dir", "pkg:github/a/b", ""},
	}
	for _, tt := range tests {
		key, q := rawKey(tt.purl)
		if key != tt.key || q != tt.qualifiers {
			t.Errorf("rawKey(%q) = (%q, %q), want (%q, %q)", tt.purl, key, q, tt.key, tt.qualifiers)
		}
	}
}
