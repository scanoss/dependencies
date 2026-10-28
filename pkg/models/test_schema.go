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

// The serving schema, used to build the database the tests run against.
//
// Fixtures under tests/ carry data only; the tables come from here. This is a
// transcription of the serving schema, taken from the production DuckDB file with
// `SELECT sql FROM duckdb_tables()`, restricted to the tables this service reads.
// The types are the SQLite equivalents, since the tests run on SQLite: DuckDB UUID
// becomes TEXT, which the queries tolerate because they CAST the UUID columns to
// VARCHAR. The only index the export carries is component(purl); the others exist
// so the fixtures stay quick, and change nothing a query relies on.
//
// Keep this in step with the real database. It is a transcription, so it can drift;
// when a query fails in production but passes here, suspect this file first.

package models

// testSchemaDDL is the schema every DB-backed test is built from.
const testSchemaDDL = `
-- Component identity. purl is the versionless purl string; purl_id and
-- source_purl_id are UUID in DuckDB, TEXT here.
CREATE TABLE component (purl TEXT, vendor TEXT, component TEXT, source_purl_id TEXT, purl_id TEXT);
CREATE INDEX idx_component_purl ON component (purl);

CREATE TABLE component_version (purl_id TEXT, version TEXT, version_name TEXT, version_semver TEXT);
CREATE INDEX idx_component_version_purl_id ON component_version (purl_id);

-- Declared licences per release, by purl_id. Mining kept one licence per URL row on
-- all_urls; a serving release can carry more than one.
CREATE TABLE component_version_license (purl_id TEXT, version TEXT, license_id INTEGER, source INTEGER);
CREATE INDEX idx_component_version_license ON component_version_license (purl_id, version);

CREATE TABLE license (id INTEGER, long_name TEXT, SPDX TEXT, content TEXT);
CREATE INDEX idx_license_id ON license (id);

CREATE TABLE SPDX_license_data (
    SPDX_id TEXT, type TEXT, reference TEXT, is_deprecated BOOLEAN, details_url TEXT,
    reference_number INTEGER, name TEXT, is_osi_approved BOOLEAN, see_also TEXT
);
CREATE INDEX idx_spdx_license_data_id ON SPDX_license_data (SPDX_id);

-- Declared dependencies per release. Replaces mining's per-ecosystem
-- <ecosystem>_dependencies tables, which kept them as a JSON array per row.
CREATE TABLE component_version_dependency (
    purl_id TEXT, version TEXT, dep_purl TEXT, dep_version TEXT, dep_scope TEXT, dep_resolved_version TEXT
);
CREATE INDEX idx_component_version_dependency ON component_version_dependency (purl_id, version);
`
