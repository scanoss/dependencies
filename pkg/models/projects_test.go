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

func TestProjectsSearch(t *testing.T) {
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	err = loadSqlData(db, "./tests/projects.sql")
	if err != nil {
		t.Fatalf("failed to load SQL test data: %v", err)
	}
	projectsModel := NewProjectModel(db)

	projects, err := projectsModel.GetProjectsByPurlName("tablestyle", 1)
	if err != nil {
		t.Errorf("projects.GetProjectsByPurlName() error = %v", err)
	}
	if len(projects) < 1 {
		t.Errorf("projects.GetProjectsByPurlName() No projects returned from query")
	}
	fmt.Printf("Projects: %v\n", projects)

	_, err = projectsModel.GetProjectsByPurlName("", 0)
	if err == nil {
		t.Errorf("projects.GetProjectsByPurlName() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = projectsModel.GetProjectsByPurlName("", -1)
	if err == nil {
		t.Errorf("projects.GetProjectsByPurlName() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = projectsModel.GetProjectsByPurlName("rubbish", -99)
	if err == nil {
		t.Errorf("projects.GetProjectsByPurlName() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
}
