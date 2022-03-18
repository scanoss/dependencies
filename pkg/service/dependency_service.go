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

// Package service implements the gRPC service endpoints
package service

import (
	"context"
	"errors"
	"github.com/jmoiron/sqlx"
	common "github.com/scanoss/papi/api/commonv2"
	pb "github.com/scanoss/papi/api/dependenciesv2"
	zlog "scanoss.com/dependencies/pkg/logger"
	"scanoss.com/dependencies/pkg/usecase"
)

type dependencyServer struct {
	pb.DependenciesServer
	db *sqlx.DB
}

func NewDependencyServer(db *sqlx.DB) pb.DependenciesServer {
	return &dependencyServer{db: db}
}

// Echo sends back the same message received
func (d dependencyServer) Echo(ctx context.Context, request *common.EchoRequest) (*common.EchoResponse, error) {
	zlog.S.Infof("Received (%v): %v", ctx, request.GetMessage())
	return &common.EchoResponse{Message: request.GetMessage()}, nil
}

// GetDependencies searches for information about the supplied dependencies
func (d dependencyServer) GetDependencies(ctx context.Context, request *pb.DependencyRequest) (*pb.DependencyResponse, error) {
	zlog.S.Infof("Processing dependency request: %v", request)
	// Make sure we have dependency data to query
	depRequest := request.GetFiles()
	if depRequest == nil || len(depRequest) == 0 {
		statusResp := common.StatusResponse{Status: common.StatusCode_FAILED, Message: "No dependency request data supplied"}
		return &pb.DependencyResponse{Status: &statusResp}, errors.New("no request data supplied")
	}
	dtoRequest, err := convertDependencyInput(request) // Convert to internal DTO for processing
	if err != nil {
		statusResp := common.StatusResponse{Status: common.StatusCode_FAILED, Message: "Problem parsing dependency input data"}
		return &pb.DependencyResponse{Status: &statusResp}, errors.New("problem parsing dependency input data")
	}
	conn, err := d.db.Connx(ctx) // Get a connection from the pool
	if err != nil {
		zlog.S.Errorf("Failed to get a database connection from the pool: %v", err)
		statusResp := common.StatusResponse{Status: common.StatusCode_FAILED, Message: "Failed to get database pool connection"}
		return &pb.DependencyResponse{Status: &statusResp}, errors.New("problem getting database pool connection")
	}
	defer closeDbConnection(conn)
	// Search the KB for information about each dependency
	depUc := usecase.NewDependencies(ctx, conn)
	dtoDependencies, err := depUc.GetDependencies(dtoRequest)
	if err != nil {
		zlog.S.Errorf("Failed to get dependencies: %v", err)
		statusResp := common.StatusResponse{Status: common.StatusCode_FAILED, Message: "Problems encountered extracting dependency data"}
		return &pb.DependencyResponse{Status: &statusResp}, nil
	}
	zlog.S.Debugf("Parsed Dependencies: %+v", dtoDependencies)
	depResponse, err := convertDependencyOutput(dtoDependencies) // Convert the internal data into a response object
	if err != nil {
		zlog.S.Errorf("Failed to covnert parsed dependencies: %v", err)
		statusResp := common.StatusResponse{Status: common.StatusCode_FAILED, Message: "Problems encountered extracting dependency data"}
		return &pb.DependencyResponse{Status: &statusResp}, nil
	}
	// Set the status and respond with the data
	statusResp := common.StatusResponse{Status: common.StatusCode_SUCCESS, Message: "Success"}
	return &pb.DependencyResponse{Files: depResponse.Files, Status: &statusResp}, nil
}

// closeDbConnection closes the specified database connection
func closeDbConnection(conn *sqlx.Conn) {
	zlog.S.Debugf("Closing DB Connection: %v", conn)
	err := conn.Close()
	if err != nil {
		zlog.S.Warnf("Warning: Problem closing database connection: %v", err)
	}
}
