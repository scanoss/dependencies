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

package models

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"testing"
)

func TestAllUrlsSearch(t *testing.T) {
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	err = LoadTestSqlData(db)
	if err != nil {
		t.Fatalf("failed to load SQL test data: %v", err)
	}
	allUrlsModel := NewAllUrlModel(db, NewMineModel(db), NewProjectModel(db))

	allUrls, err := allUrlsModel.GetUrlsByPurlName("tablestyle", 1)
	if err != nil {
		t.Errorf("all_urls.GetUrlsByPurlName() error = %v", err)
	}
	if len(allUrls) < 1 {
		t.Errorf("all_urls.GetUrlsByPurlName() No URLs returned from query")
	}
	fmt.Printf("All Urls: %v\n", allUrls)

	allUrls, err = allUrlsModel.GetUrlsByPurlName("NONEXISTENT", 0)
	if err != nil {
		t.Errorf("all_urls.GetUrlsByPurlName() error = %v", err)
	}
	if len(allUrls) > 0 {
		t.Errorf("all_urls.GetUrlsByPurlName() URLs found when none should be: %v", allUrlsModel)
	}
	fmt.Printf("No Urls: %v\n", allUrls)

	_, err = allUrlsModel.GetUrlsByPurlName("", -1)
	if err == nil {
		t.Errorf("all_urls.GetUrlsByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = allUrlsModel.GetUrlsByPurlName("", 0)
	if err == nil {
		t.Errorf("all_urls.GetUrlsByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = allUrlsModel.GetUrlsByPurlString("")
	if err == nil {
		t.Errorf("all_urls.GetUrlsByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = allUrlsModel.GetUrlsByPurlString("rubbish-purl")
	if err == nil {
		t.Errorf("all_urls.GetUrlsByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	allUrls, err = allUrlsModel.GetUrlsByPurlString("pkg:gem/taballa.hp-PD/tablestyle")
	if err != nil {
		t.Errorf("all_urls.GetUrlsByPurlString() error = %v", err)
	}
	if len(allUrls) < 1 {
		t.Errorf("all_urls.GetUrlsByPurlString() No URLs returned from query")
	}
	fmt.Printf("All Urls: %v\n", allUrls)
}

func TestAllUrlsSearchBadSql(t *testing.T) {
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	allUrlsModel := NewAllUrlModel(db, NewMineModel(db), NewProjectModel(db))
	allUrlsModel.mine.ResetMineCache()
	_, err = allUrlsModel.GetUrlsByPurlString("pkg:gem/taballa.hp-PD/tablestyle")
	if err == nil {
		t.Errorf("all_urls.GetUrlsByPurlString() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	// Load some tables (leaving out projects)
	loadTestSqlDataFiles(db, []string{"./tests/mines.sql", "./tests/all_urls.sql"})

	allUrls, err := allUrlsModel.GetUrlsByPurlName("tablestyle", 1)
	if err != nil {
		t.Errorf("all_urls.GetUrlsByPurlName() error = %v", err)
	}
	if len(allUrls) < 1 {
		t.Errorf("all_urls.GetUrlsByPurlName() No URLs returned from query")
	}
	fmt.Printf("All Urls: %v\n", allUrls)
}
