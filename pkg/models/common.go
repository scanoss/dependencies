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

// This file common tasks for the models package

package models

import (
	"github.com/jmoiron/sqlx"
	"io/ioutil"
)

// loadSqlData Load the specified SQL files into the supplied DB
func loadSqlData(db *sqlx.DB, filename string) error {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	_, err = db.Exec(string(file))
	if err != nil {
		return err
	}
	return nil
}

// LoadTestSqlData loads all the required test SQL files
func LoadTestSqlData(db *sqlx.DB) error {
	files := []string{"../models/tests/mines.sql", "../models/tests/all_urls.sql", "../models/tests/projects.sql"}
	return loadTestSqlDataFiles(db, files)
}

// loadTestSqlDataFiles loads a list of test SQL files
func loadTestSqlDataFiles(db *sqlx.DB, files []string) error {
	for _, file := range files {
		err := loadSqlData(db, file)
		if err != nil {
			return err
		}
	}
	return nil
}
