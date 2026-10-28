// SPDX-License-Identifier: GPL-2.0-or-later
/*
 * Copyright (C) 2026 SCANOSS.COM
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
	"testing"
)

// Ported from scanoss.api's license.TestNearestVersions, so the two order the
// fallback candidates the same way.
func TestNearestVersions(t *testing.T) {
	versions := []string{"1.0.0", "1.1.0", "2.0.0", "2.1.0", "3.0.0", "bad-version"}
	got := NearestVersions("2.0.0", versions)
	// nearest to 2.0.0, excluding itself; ties prefer the higher side;
	// unparseable last.
	want := []string{"2.1.0", "1.1.0", "3.0.0", "1.0.0", "bad-version"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}

	// target not a valid semver → parseable asc, then unparseable.
	got2 := NearestVersions("not-semver", []string{"2.0.0", "1.0.0", "x"})
	if len(got2) != 3 || got2[0] != "1.0.0" || got2[1] != "2.0.0" || got2[2] != "x" {
		t.Errorf("non-semver target order = %v", got2)
	}

	// target excluded, dedupe.
	got3 := NearestVersions("1.0.0", []string{"1.0.0", "1.0.0", "1.1.0"})
	if len(got3) != 1 || got3[0] != "1.1.0" {
		t.Errorf("exclude/dedupe = %v", got3)
	}
}

// Versions that differ only in build metadata - golang pseudo-versions as the
// export stores them - are ordered by it, so the nearest in time comes first.
func TestNearestVersionsOrdersBuildMetadata(t *testing.T) {
	versions := []string{
		"0.0.0+20221111041011-e4397a8d7954",
		"0.0.0+20230818070408-28594793d352",
		"0.0.0+20230111025418-671eee99d473",
	}
	got := NearestVersions("0.0.0+20260728050559-d6f5572444cd", versions)
	want := []string{"0.0.0+20230818070408-28594793d352", "0.0.0+20230111025418-671eee99d473", "0.0.0+20221111041011-e4397a8d7954"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
