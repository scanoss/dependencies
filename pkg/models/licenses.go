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

// Handle all interaction with the licenses table

package models

import (
	"context"
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
	"regexp"
	zlog "scanoss.com/dependencies/pkg/logger"
	"strings"
)

type licenseModel struct {
	ctx  context.Context
	conn *sqlx.Conn
}

type License struct {
	Id          int32  `db:"id"`
	LicenseName string `db:"license_name"`
	LicenseId   string `db:"spdx_id"`
	IsSpdx      bool   `db:"is_spdx"`
}

var bannedLicPrefixes = []string{"see ", "\"", "'", "-", "*", ".", "/", "?", "@", "\\", ";", ",", "`", "$"} // unwanted license prefixes
var bannedLicSuffixes = []string{".md", ".txt", ".html"}                                                    // unwanted license suffixes
var whiteSpaceRegex = regexp.MustCompile("\\s+")                                                            // generic whitespace regex

// TODO add cache for licenses already searched for?

func NewLicenseModel(ctx context.Context, conn *sqlx.Conn) *licenseModel {
	return &licenseModel{ctx: ctx, conn: conn}
}

// GetLicenseByName retrieves the license details for the given license name
func (m *licenseModel) GetLicenseByName(name string) (License, error) {
	if len(name) == 0 {
		zlog.S.Error("Please specify a valid License Name to query")
		return License{}, errors.New("please specify a valid License Name to query")
	}
	rows, err := m.conn.QueryxContext(m.ctx,
		"SELECT id, license_name, spdx_id, is_spdx FROM licenses"+
			" WHERE license_name = $1",
		name)
	defer CloseRows(rows)
	if err != nil {
		zlog.S.Errorf("Error: Failed to query license table for %v: %v", name, err)
		return License{}, fmt.Errorf("failed to query the license table: %v", err)
	}
	var license License
	for rows.Next() {
		err = rows.StructScan(&license)
		if err != nil {
			zlog.S.Errorf("Failed to parse license table results for %#v: %v", rows, err)
			zlog.S.Errorf("Query failed for license_name = %v", name)
			return License{}, fmt.Errorf("failed to query the license table: %v", err)
		}
		break // Only process the first row
	}
	return license, nil
}

func (m *licenseModel) SaveLicense(name string) error {
	if len(name) == 0 {
		zlog.S.Error("Please specify a valid License Name to save")
		return errors.New("please specify a valid License Name to save")
	}

	return errors.New("not implemented yet")
}

// CleanseLicenseName cleans up a license name to make it searchable in the licenses table
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
		//zlog.S.Debugf("in: %v clean: %v cleaner: %v cleanest: %v", name, clean, cleaner, cleanest)
		return strings.TrimSpace(cleanest), nil // return the cleansed license name
	}
	return "", nil // empty string, so just return it.
}
