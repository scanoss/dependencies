// SPDX-License-Identifier: GPL-2.0-or-later
/*
 * Copyright (C) 2018-2022 SCANOSS.COM
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

// Handle all interaction with the license catalogue

package models

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type LicenseModel struct {
	ctx context.Context
	s   *zap.SugaredLogger
	db  *sqlx.DB
}

// License is one licence as the response reports it: its name, SPDX identifier
// and whether that identifier is in the SPDX registry.
type License struct {
	LicenseName string `db:"license_name"`
	LicenseID   string `db:"spdx_id"`
	IsSpdx      bool   `db:"is_spdx"`
}

var bannedLicPrefixes = []string{"see ", "\"", "'", "-", "*", ".", "/", "?", "@", "\\", ";", ",", "`", "$"} // unwanted license prefixes
var bannedLicSuffixes = []string{".md", ".txt", ".html"}                                                    // unwanted license suffixes
var whiteSpaceRegex = regexp.MustCompile(`\s+`)                                                             // generic whitespace regex

// NewLicenseModel create a new instance of the License Model.
func NewLicenseModel(ctx context.Context, s *zap.SugaredLogger, db *sqlx.DB) *LicenseModel {
	return &LicenseModel{ctx: ctx, s: s, db: db}
}

// licenseByNameQuery reads a licence from the serving catalogue by its name.
//
// Mining's `licenses` table carried license_name, spdx_id and an is_spdx flag;
// serving's `license` carries the name as long_name and the identifier as SPDX,
// and whether it is SPDX is membership of the identifier in "SPDX_license_data".
// Names are not unique in the catalogue, so the lowest id wins, for a stable answer.
const licenseByNameQuery = `SELECT l.long_name AS license_name,
	COALESCE(l."SPDX", '') AS spdx_id,
	CASE WHEN sld."SPDX_id" IS NOT NULL THEN 1 ELSE 0 END AS is_spdx
FROM license l
LEFT JOIN "SPDX_license_data" sld ON sld."SPDX_id" = l."SPDX"
WHERE l.long_name = ?
ORDER BY l.id
LIMIT 1`

// GetLicenseByName retrieves the license details for the given license name. An
// unknown name is not an error: it returns an empty License.
//
// The mining version could also insert a missing name (COMP_COMMIT_MISSING). The
// serving knowledge base is read-only, so it only ever reads.
func (m *LicenseModel) GetLicenseByName(name string) (License, error) {
	if len(name) == 0 {
		m.s.Warn("No License Name specified to query")
		return License{}, nil
	}
	var licenses []License
	err := m.db.SelectContext(m.ctx, &licenses, m.db.Rebind(licenseByNameQuery), name)
	if err != nil {
		m.s.Errorf("Failed to query license table for %v: %v", name, err)
		return License{}, fmt.Errorf("failed to query the license table: %v", err)
	}
	if len(licenses) == 0 {
		return License{}, nil
	}
	return licenses[0], nil
}

// CleanseLicenseName cleans up a license name to make it searchable in the licenses table.
func CleanseLicenseName(name string) (string, error) {
	if len(name) > 0 {
		name = strings.TrimSpace(name)     // remove leading/trailing spaces before even starting
		nameLower := strings.ToLower(name) // check banned strings against lowercase
		for _, prefix := range bannedLicPrefixes {
			if strings.HasPrefix(nameLower, prefix) {
				return "", fmt.Errorf("license name has banned prefix: %v", prefix)
			}
		}
		for _, suffix := range bannedLicSuffixes {
			if strings.HasSuffix(nameLower, suffix) {
				return "", fmt.Errorf("license name has banned suffix: %v", suffix)
			}
		}
		clean := whiteSpaceRegex.ReplaceAllString(name, " ")    // gets rid of new lines, tabs, etc.
		cleaner := whiteSpaceRegex.ReplaceAllString(clean, " ") // reduces it down to a single space
		cleanest := strings.ReplaceAll(cleaner, ",", ";")       // swap commas with semicolons
		return strings.TrimSpace(cleanest), nil                 // return the cleansed license name
	}
	return "", nil // empty string, so just return it.
}
