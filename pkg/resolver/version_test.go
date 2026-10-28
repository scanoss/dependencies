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

import "testing"

func TestSelectVersion(t *testing.T) {
	versions := []string{"1.0.0", "1.10.0", "1.2.0", "2.0.0-beta", "2.0.0"}

	tests := []struct {
		name       string
		versions   []string
		constraint string
		want       string
		wantOK     bool
	}{
		{
			name:     "no constraint takes the highest",
			versions: versions, want: "2.0.0", wantOK: true,
		},
		{
			// Ordering is semantic, not lexical: 1.10.0 beats 1.2.0.
			name:     "highest is compared by precedence, not as text",
			versions: versions, constraint: "<2.0.0", want: "1.10.0", wantOK: true,
		},
		{
			name:     "constraint excluding everything",
			versions: versions, constraint: ">9.0.0", want: "", wantOK: false,
		},
		{
			name:     "empty version list",
			versions: nil, want: "", wantOK: false,
		},
		{
			// A version that will not parse is scored as v0.0.0 rather than dropped, so
			// a component whose versions are all non-semver still resolves to one.
			name:     "non-semver versions still resolve",
			versions: []string{"trunk", "release-candidate"}, want: "trunk", wantOK: true,
		},
		{
			name:     "semver wins over non-semver",
			versions: []string{"trunk", "1.0.0"}, want: "1.0.0", wantOK: true,
		},
		{
			// An unparseable constraint degrades to "highest version", which is what
			// the mining resolution did rather than failing the lookup.
			name:     "unparseable constraint is ignored",
			versions: versions, constraint: "not a constraint", want: "2.0.0", wantOK: true,
		},
		{
			name:     "empty strings are skipped",
			versions: []string{"", "1.0.0", ""}, want: "1.0.0", wantOK: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := SelectVersion(tt.versions, tt.constraint)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("SelectVersion(%v, %q) = (%q, %v), want (%q, %v)",
					tt.versions, tt.constraint, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestParseIdentity(t *testing.T) {
	tests := []struct {
		purl     string
		wantName string
		wantType string
		wantVer  string
	}{
		{"pkg:npm/log4js", "log4js", "npm", ""},
		{"pkg:npm/log4js@1.0.0", "log4js", "npm", "1.0.0"},
		// A namespace becomes part of the lookup name, which is how the serving
		// component.purl key is built.
		{"pkg:github/log4js-node/log4js-node", "log4js-node/log4js-node", "github", ""},
		{"pkg:maven/org.apache.logging.log4j/log4j-core@2.14.1", "org.apache.logging.log4j/log4j-core", "maven", "2.14.1"},
	}
	for _, tt := range tests {
		t.Run(tt.purl, func(t *testing.T) {
			got, err := ParseIdentity(tt.purl)
			if err != nil {
				t.Fatalf("ParseIdentity(%q) errored: %v", tt.purl, err)
			}
			if got.PurlName != tt.wantName || got.PurlType != tt.wantType || got.Version != tt.wantVer {
				t.Errorf("= %+v, want name=%q type=%q version=%q", got, tt.wantName, tt.wantType, tt.wantVer)
			}
			if key := servingPurlKey(got); key != "pkg:"+tt.wantType+"/"+tt.wantName {
				t.Errorf("servingPurlKey = %q", key)
			}
		})
	}

	if _, err := ParseIdentity("not-a-purl"); err == nil {
		t.Error("want an error for an unparseable purl")
	}
}
