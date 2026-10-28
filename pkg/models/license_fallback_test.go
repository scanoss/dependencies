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

package models

import (
	"reflect"
	"testing"

	"github.com/jmoiron/sqlx"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
)

// fallbackFixture adds, on top of the shared fixtures, components whose requested
// release carries only license 523 - the export's empty "no licence" row, which
// 43% of component_version_license points at - so each fallback step can be
// reached on its own.
const fallbackFixture = `
INSERT INTO license (id, long_name, SPDX, content) VALUES (523, '', NULL, NULL);
INSERT INTO license (id, long_name, SPDX, content) VALUES (601, 'BSD 3-Clause', 'BSD-3-Clause', NULL);
INSERT INTO license (id, long_name, SPDX, content) VALUES (602, 'ISC', 'ISC', NULL);

-- 1. The source component has a licence at the same version.
INSERT INTO component (purl, vendor, component, source_purl_id, purl_id) VALUES ('pkg:npm/fb-source', 'acme', 'fb-source', 'uuid-fb-source-src', 'uuid-fb-source');
INSERT INTO component (purl, vendor, component, source_purl_id, purl_id) VALUES ('pkg:github/acme/fb-source', 'acme', 'fb-source', 'uuid-fb-source-src', 'uuid-fb-source-src');
INSERT INTO component_version (purl_id, version, version_name, version_semver) VALUES ('uuid-fb-source', '1.0.0', '1.0.0', '1.0.0');
INSERT INTO component_version (purl_id, version, version_name, version_semver) VALUES ('uuid-fb-source-src', '1.0.0', '1.0.0', '1.0.0');
INSERT INTO component_version_license (purl_id, version, license_id, source) VALUES ('uuid-fb-source', '1.0.0', 523, 0);
INSERT INTO component_version_license (purl_id, version, license_id, source) VALUES ('uuid-fb-source-src', '1.0.0', 601, 0);

-- 2. No source; two licensed neighbours equally near (a tie goes to the higher).
INSERT INTO component (purl, vendor, component, source_purl_id, purl_id) VALUES ('pkg:npm/fb-nearest', 'acme', 'fb-nearest', NULL, 'uuid-fb-nearest');
INSERT INTO component_version (purl_id, version, version_name, version_semver) VALUES ('uuid-fb-nearest', '1.0.0', '1.0.0', '1.0.0');
INSERT INTO component_version (purl_id, version, version_name, version_semver) VALUES ('uuid-fb-nearest', '2.0.0', '2.0.0', '2.0.0');
INSERT INTO component_version (purl_id, version, version_name, version_semver) VALUES ('uuid-fb-nearest', '2.1.0', '2.1.0', '2.1.0');
INSERT INTO component_version (purl_id, version, version_name, version_semver) VALUES ('uuid-fb-nearest', '4.0.0', '4.0.0', '4.0.0');
INSERT INTO component_version_license (purl_id, version, license_id, source) VALUES ('uuid-fb-nearest', '1.0.0', 1, 0);
INSERT INTO component_version_license (purl_id, version, license_id, source) VALUES ('uuid-fb-nearest', '2.0.0', 523, 0);
INSERT INTO component_version_license (purl_id, version, license_id, source) VALUES ('uuid-fb-nearest', '2.1.0', 2, 0);
INSERT INTO component_version_license (purl_id, version, license_id, source) VALUES ('uuid-fb-nearest', '4.0.0', 523, 0);

-- 3. Nothing licensed on the component; its source has a licence at another version.
INSERT INTO component (purl, vendor, component, source_purl_id, purl_id) VALUES ('pkg:npm/fb-source-nearest', 'acme', 'fb-source-nearest', 'uuid-fb-sn-src', 'uuid-fb-sn');
INSERT INTO component (purl, vendor, component, source_purl_id, purl_id) VALUES ('pkg:github/acme/fb-source-nearest', 'acme', 'fb-source-nearest', NULL, 'uuid-fb-sn-src');
INSERT INTO component_version (purl_id, version, version_name, version_semver) VALUES ('uuid-fb-sn', '1.0.0', '1.0.0', '1.0.0');
INSERT INTO component_version (purl_id, version, version_name, version_semver) VALUES ('uuid-fb-sn-src', '0.9.0', '0.9.0', '0.9.0');
INSERT INTO component_version_license (purl_id, version, license_id, source) VALUES ('uuid-fb-sn', '1.0.0', 523, 0);
INSERT INTO component_version_license (purl_id, version, license_id, source) VALUES ('uuid-fb-sn-src', '0.9.0', 602, 0);

-- 5. The release holds only a placeholder; the nearest version with one is skipped
-- for the nearest version with an actual licence.
INSERT INTO license (id, long_name, SPDX, content) VALUES (4810, 'None detected', NULL, NULL);
INSERT INTO component (purl, vendor, component, source_purl_id, purl_id) VALUES ('pkg:npm/fb-placeholder', 'acme', 'fb-placeholder', NULL, 'uuid-fb-ph');
INSERT INTO component_version (purl_id, version, version_name, version_semver) VALUES ('uuid-fb-ph', '1.0.0', '1.0.0', '1.0.0');
INSERT INTO component_version (purl_id, version, version_name, version_semver) VALUES ('uuid-fb-ph', '2.0.0', '2.0.0', '2.0.0');
INSERT INTO component_version (purl_id, version, version_name, version_semver) VALUES ('uuid-fb-ph', '3.0.0', '3.0.0', '3.0.0');
INSERT INTO component_version_license (purl_id, version, license_id, source) VALUES ('uuid-fb-ph', '1.0.0', 1, 0);
INSERT INTO component_version_license (purl_id, version, license_id, source) VALUES ('uuid-fb-ph', '2.0.0', 4810, 0);
INSERT INTO component_version_license (purl_id, version, license_id, source) VALUES ('uuid-fb-ph', '3.0.0', 4810, 0);

-- 4. Nothing licensed anywhere.
INSERT INTO component (purl, vendor, component, source_purl_id, purl_id) VALUES ('pkg:npm/fb-none', 'acme', 'fb-none', NULL, 'uuid-fb-none');
INSERT INTO component_version (purl_id, version, version_name, version_semver) VALUES ('uuid-fb-none', '1.0.0', '1.0.0', '1.0.0');
INSERT INTO component_version (purl_id, version, version_name, version_semver) VALUES ('uuid-fb-none', '2.0.0', '2.0.0', '2.0.0');
INSERT INTO component_version_license (purl_id, version, license_id, source) VALUES ('uuid-fb-none', '1.0.0', 523, 0);
INSERT INTO component_version_license (purl_id, version, license_id, source) VALUES ('uuid-fb-none', '2.0.0', 523, 0);
`

func loadFallbackFixture(t *testing.T, db *sqlx.DB) {
	t.Helper()
	if _, err := db.Exec(fallbackFixture); err != nil {
		t.Fatalf("failed to load the fallback fixture: %v", err)
	}
}

func TestGetURLsByComponentLicenseFallback(t *testing.T) {
	db := sqliteSetup(t)
	defer zlog.SyncZap()
	loadFallbackFixture(t, db)
	model := newLicensesModel(t, db, nil)
	bsd := License{LicenseName: "BSD 3-Clause", LicenseID: "BSD-3-Clause"}
	apache := License{LicenseName: "Apache 2.0", LicenseID: "Apache-2.0", IsSpdx: true}
	isc := License{LicenseName: "ISC", LicenseID: "ISC"}

	tests := []struct {
		name, purl, requirement string
		want                    []License
		wantMessage             string
	}{
		{name: "source at the same version", purl: "pkg:npm/fb-source", requirement: "1.0.0", want: []License{bsd},
			wantMessage: "Showing results from the source code purl (pkg:github/acme/fb-source)"},
		{name: "nearest version, tie to the higher", purl: "pkg:npm/fb-nearest", requirement: "2.0.0", want: []License{apache},
			wantMessage: "Showing results from version 2.1.0"},
		{name: "nearest version below", purl: "pkg:npm/fb-nearest", requirement: "4.0.0", want: []License{apache},
			wantMessage: "Showing results from version 2.1.0"},
		{name: "nearest version of the source", purl: "pkg:npm/fb-source-nearest", requirement: "1.0.0", want: []License{isc},
			wantMessage: "Showing results from the source code purl (pkg:github/acme/fb-source-nearest) version 0.9.0"},
		{name: "nothing to borrow", purl: "pkg:npm/fb-none", requirement: "2.0.0"},
		{name: "placeholder borrows past placeholders", purl: "pkg:npm/fb-placeholder", requirement: "3.0.0",
			want: []License{{LicenseName: "MIT", LicenseID: "MIT", IsSpdx: true}}, wantMessage: "Showing results from version 1.0.0"},
		// A release with a licence of its own never borrows one.
		{name: "own licence", purl: "pkg:npm/fb-nearest", requirement: "1.0.0",
			want: []License{{LicenseName: "MIT", LicenseID: "MIT", IsSpdx: true}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component, key := resolve(t, db, tt.purl, tt.requirement)
			got, err := model.GetURLsByComponent(component, key)
			if err != nil {
				t.Fatalf("GetURLsByComponent() error = %v", err)
			}
			if (len(got.Licenses) > 0 || len(tt.want) > 0) && !reflect.DeepEqual(got.Licenses, tt.want) {
				t.Errorf("licenses = %#v, want %#v", got.Licenses, tt.want)
			}
			if got.FallbackMessage != tt.wantMessage {
				t.Errorf("fallback message = %q, want %q", got.FallbackMessage, tt.wantMessage)
			}
			// The licence is reported against the requested release, not the one it
			// was borrowed from.
			if got.Version != tt.requirement {
				t.Errorf("version = %q, want %q", got.Version, tt.requirement)
			}
		})
	}
}
