// SPDX-License-Identifier: GPL-2.0-or-later
/*
 * Copyright (C) 2018-2023 SCANOSS.COM
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
	"context"
	"fmt"
	"testing"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
)

func TestProjectsSearch(t *testing.T) {
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer zlog.SyncZap()
	ctx := ctxzap.ToContext(context.Background(), zlog.L)
	s := ctxzap.Extract(ctx).Sugar()
	db := sqliteSetup(t) // Setup SQL Lite DB
	defer CloseDB(db)
	conn := sqliteConn(t, ctx, db) // Get a connection from the pool
	defer CloseConn(conn)
	err = loadTestSQLDataFiles(db, ctx, conn, []string{"../models/tests/projects.sql", "../models/tests/mines.sql", "../models/tests/licenses.sql"})
	if err != nil {
		t.Fatalf("failed to load SQL test data: %v", err)
	}
	projectsModel := NewProjectModel(ctx, s, conn)
	var purlName = "tablestyle"
	var purlType = "gem"
	fmt.Printf("Searching for project list: %v - %v\n", purlName, purlType)
	projects, err := projectsModel.GetProjectsByPurlName(purlName, purlType)
	if err != nil {
		t.Errorf("projects.GetProjectsByPurlName() error = %v", err)
	}
	if len(projects) < 1 {
		t.Errorf("projects.GetProjectsByPurlName() No projects returned from query")
	}
	fmt.Printf("Projects: %#v\n", projects)

	purlName = ""
	purlType = "npm"
	fmt.Printf("Searching for project list: %v - %v\n", purlName, purlType)
	_, err = projectsModel.GetProjectsByPurlName(purlName, purlType)
	if err == nil {
		t.Errorf("projects.GetProjectsByPurlName() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	purlName = "tablestyle"
	purlType = ""
	fmt.Printf("Searching for project list: %v - %v\n", purlName, purlType)
	_, err = projectsModel.GetProjectsByPurlName(purlName, purlType)
	if err == nil {
		t.Errorf("projects.GetProjectsByPurlName() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	purlName = "tablestyle"
	var mineId int32 = 1
	fmt.Printf("Searching for project: %v - %v\n", purlName, mineId)
	project, err := projectsModel.GetProjectByPurlName("tablestyle", mineId)
	if err != nil {
		t.Errorf("projects.GetProjectByPurlName() error = %+v", err)
	}
	if len(project.PurlName) == 0 {
		t.Errorf("projects.GetProjectByPurlName() No project returned from query")
	} else {
		fmt.Printf("Project: %v\n", project)
	}
	purlName = ""
	mineId = -1
	fmt.Printf("Searching for project list: %v - %v\n", purlName, purlType)
	_, err = projectsModel.GetProjectByPurlName(purlName, mineId)
	if err == nil {
		t.Errorf("projects.GetProjectByPurlName() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	purlName = "NONEXISTENT"
	mineId = -1
	fmt.Printf("Searching for project list: %v - %v\n", purlName, purlType)
	_, err = projectsModel.GetProjectByPurlName(purlName, mineId)
	if err == nil {
		t.Errorf("projects.GetProjectByPurlName() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
}

func TestProjectsSearchBadSql(t *testing.T) {
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer zlog.SyncZap()
	ctx := ctxzap.ToContext(context.Background(), zlog.L)
	s := ctxzap.Extract(ctx).Sugar()
	db := sqliteSetup(t) // Setup SQL Lite DB
	defer CloseDB(db)
	conn := sqliteConn(t, ctx, db) // Get a connection from the pool
	defer CloseConn(conn)
	projectsModel := NewProjectModel(ctx, s, conn)
	_, err = projectsModel.GetProjectsByPurlName("rubbish", "rubbish")
	if err == nil {
		t.Errorf("projects.GetProjectsByPurlName() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
	_, err = projectsModel.GetProjectByPurlName("rubbish", 2)
	if err == nil {
		t.Errorf("projects.GetProjectByPurlName() error = did not get an error")
	} else {
		fmt.Printf("Got expected error = %v\n", err)
	}
}
