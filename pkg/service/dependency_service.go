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
	_ "errors"
	"fmt"
	_ "fmt"
	_ "github.com/scanoss/go-grpc-helper/pkg/grpc/database"
	gd "github.com/scanoss/go-grpc-helper/pkg/grpc/database"
	_ "google.golang.org/protobuf/runtime/protoimpl"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/jmoiron/sqlx"
	common "github.com/scanoss/papi/api/commonv2"
	pb "github.com/scanoss/papi/api/dependenciesv2"
	myconfig "scanoss.com/dependencies/pkg/config"
	"scanoss.com/dependencies/pkg/usecase"
)

type dependencyServer struct {
	pb.DependenciesServer
	db     *sqlx.DB
	config *myconfig.ServerConfig
}

// NewDependencyServer creates a new instance of Dependency Server.
func NewDependencyServer(db *sqlx.DB, config *myconfig.ServerConfig) pb.DependenciesServer {
	setupMetrics()
	return &dependencyServer{db: db, config: config}
}

// Echo sends back the same message received.
func (d dependencyServer) Echo(ctx context.Context, request *common.EchoRequest) (*common.EchoResponse, error) {
	s := ctxzap.Extract(ctx).Sugar()
	s.Infof("Received %v", request.GetMessage())
	return &common.EchoResponse{Message: request.GetMessage()}, nil
}

// GetDependencies searches for information about the supplied dependencies.
func (d dependencyServer) GetDependencies(ctx context.Context, request *pb.DependencyRequest) (*pb.DependencyResponse, error) {
	requestStartTime := time.Now() // Capture the scan start time
	s := ctxzap.Extract(ctx).Sugar()
	s.Info("Processing dependency request...")
	// Make sure we have dependency data to query
	depRequest := request.GetFiles()
	if len(depRequest) == 0 {
		s.Warn("No dependency request data supplied to decorate. Ignoring request.")
		statusResp := common.StatusResponse{Status: common.StatusCode_FAILED, Message: "No dependency request data supplied"}
		return &pb.DependencyResponse{Status: &statusResp}, errors.New("no request data supplied")
	}
	dtoRequest, err := convertDependencyInput(s, request) // Convert to internal DTO for processing
	if err != nil {
		statusResp := common.StatusResponse{Status: common.StatusCode_FAILED, Message: "Problem parsing dependency input data"}
		return &pb.DependencyResponse{Status: &statusResp}, errors.New("problem parsing dependency input data")
	}
	telemetryReqCounters(ctx, d.config, depRequest) // Update request counters
	conn, err := d.db.Connx(ctx)                    // Get a connection from the pool
	if err != nil {
		s.Errorf("Failed to get a database connection from the pool: %v", err)
		statusResp := common.StatusResponse{Status: common.StatusCode_FAILED, Message: "Failed to get database pool connection"}
		return &pb.DependencyResponse{Status: &statusResp}, errors.New("problem getting database pool connection")
	}
	defer gd.CloseSQLConnection(conn)
	// Search the KB for information about each dependency
	depUc := usecase.NewDependencies(ctx, s, conn, d.config)
	dtoDependencies, warn, err := depUc.GetDependencies(dtoRequest)
	statusResp := common.StatusResponse{Status: common.StatusCode_SUCCESS, Message: "Success"} // Assume success :-)
	if err != nil {
		if !warn { // Definitely an error, and not a warning
			s.Errorf("Failed to get dependencies: %v", err)
			statusResp = common.StatusResponse{Status: common.StatusCode_FAILED, Message: "Problems encountered extracting dependency data"}
			return &pb.DependencyResponse{Status: &statusResp}, nil
		}
		statusResp = common.StatusResponse{Status: common.StatusCode_SUCCEEDED_WITH_WARNINGS, Message: "Problems decorating some purls"}
	}
	depResponse, err := convertDependencyOutput(s, dtoDependencies) // Convert the internal data into a response object
	if err != nil {
		s.Errorf("Failed to covnert parsed dependencies: %v", err)
		statusResp = common.StatusResponse{Status: common.StatusCode_FAILED, Message: "Problems encountered extracting dependency data"}
		return &pb.DependencyResponse{Status: &statusResp}, errors.New("problem converting dependency DTO")
	}
	telemetryRequestTime(ctx, d.config, requestStartTime) // Record the request processing time
	// Set the status and respond with the data
	return &pb.DependencyResponse{Files: depResponse.Files, Status: &statusResp}, nil
}

func (d dependencyServer) GetTransitiveDependencies(ctx context.Context, request *pb.TransitiveDependencyRequest) (*pb.TransitiveDependencyResponse, error) {
	requestStartTime := time.Now() // Capture the scan start time
	logger := ctxzap.Extract(ctx).Sugar()
	logger.Info("Processing dependency request...")
	conn, err := d.db.Connx(ctx) // Get a connection from the pool
	if err != nil {
		logger.Errorf("Failed to get a database connection from the pool: %v", err)
		// Return error response...
		return nil, errors.New("problem getting database pool connection")
	}
	defer conn.Close()                                                                    // Move this here, after error check
	transitiveDependencyInput, err := convertToTransitiveDependencyInput(logger, request) // Convert to internal DTO for processing
	if err != nil {
		logger.Errorf("%v", err)
		statusResp := common.StatusResponse{Status: common.StatusCode_FAILED,
			Message: "Problems encountered extracting dependency data"}
		msg := fmt.Sprintf("problems parsing request: %v", err)
		return &pb.TransitiveDependencyResponse{Status: &statusResp}, errors.New(msg)
	}
	logger.Infof("Transitive dependencies input: %v", transitiveDependencyInput)
	transitiveDependenciesUc := usecase.NewTransitiveDependencies(ctx, logger, d.db, d.config)
	transitiveDependencies, err := transitiveDependenciesUc.GetTransitiveDependencies(transitiveDependencyInput)

	logger.Infof("Transitive dependencies %v", transitiveDependencies)

	output, err := convertToTransitiveDependencyOutput(logger, transitiveDependencies)

	telemetryRequestTime(ctx, d.config, requestStartTime) // Record the request processing time

	return output, nil
}

// telemetryRequestTime records the request time to telemetry.
func telemetryRequestTime(ctx context.Context, config *myconfig.ServerConfig, requestStartTime time.Time) {
	if config.Telemetry.Enabled {
		elapsedTime := time.Since(requestStartTime).Milliseconds() // Time taken to run the dependency request
		oltpMetrics.depHistogram.Record(ctx, elapsedTime)          // Record dep request time
	}
}

// telemetryReqCounters counts the number of requests for telemetry.
func telemetryReqCounters(ctx context.Context, config *myconfig.ServerConfig, depRequest []*pb.DependencyRequest_Files) {
	if config.Telemetry.Enabled {
		oltpMetrics.depFileCounter.Add(ctx, int64(len(depRequest))) // count the number of dep files requested (usually one)
		depCount := 0
		for _, depFile := range depRequest {
			depCount += len(depFile.GetPurls())
		}
		oltpMetrics.depsCounter.Add(ctx, int64(depCount)) // count the number of dependencies in the request
	}
}
