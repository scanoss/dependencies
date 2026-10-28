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
)

// TestSelfSourceIsNoSource pins that a component naming itself as its source has
// no fallback target. Most GitHub components do, and following the link meant an
// unindexed scan of component for nothing.
func TestSelfSourceIsNoSource(t *testing.T) {
	r := newTestResolver(t)
	if _, err := r.db.Exec(`INSERT INTO component (purl, purl_id, source_purl_id) VALUES ('pkg:github/self/ref', 'uuid-self', 'uuid-self')`); err != nil {
		t.Fatalf("failed to seed: %v", err)
	}
	if _, err := r.db.Exec(`INSERT INTO component_version (purl_id, version) VALUES ('uuid-self', '1.0.0')`); err != nil {
		t.Fatalf("failed to seed: %v", err)
	}
	got, err := r.Resolve(context.Background(), "pkg:github/self/ref", "")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got.SourcePurl != nil {
		t.Errorf("SourcePurl = %+v, want nil for a self-referencing source", got.SourcePurl)
	}
}
