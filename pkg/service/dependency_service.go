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
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"scanoss.com/dependencies/pkg/dtos"
	"strings"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/jmoiron/sqlx"
	gd "github.com/scanoss/go-grpc-helper/pkg/grpc/database"
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

// For a basic print of a single node and its children
func PrintFromNode(tree map[string]*dtos.Dependency, startNode string) {
	if dep, exists := tree[startNode]; exists {
		fmt.Printf("Tree from node %s:\n", startNode)
		printNode(dep, 0)
	} else {
		fmt.Printf("Node %s not found in tree\n", startNode)
	}
}

func printNode(dep *dtos.Dependency, level int) {
	if dep == nil {
		return
	}

	indent := strings.Repeat("  ", level)
	fmt.Printf("%s%s@%s\n", indent, dep.Purl, dep.Version)

	for _, child := range dep.Children {
		printNode(child, level+1)
	}
}

func SaveDependencyTreeToFile(dep *dtos.Dependency, filename string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Open file for writing
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	// Create a buffered writer for better performance
	writer := bufio.NewWriter(file)

	// Write the tree structure recursively
	writeTreeNode(dep, 0, writer)

	// Flush the buffer to ensure all data is written
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush writer: %v", err)
	}

	return nil
}

func writeTreeNode(dep *dtos.Dependency, level int, writer *bufio.Writer) {
	if dep == nil {
		return
	}

	// Write current node with indentation
	indent := strings.Repeat("  ", level)
	_, err := writer.WriteString(fmt.Sprintf("%s%s@%s\n", indent, dep.Purl, dep.Version))
	if err != nil {
		fmt.Printf("Error writing to file: %v\n", err)
		return
	}

	// Recursively write all children
	for _, child := range dep.Children {
		writeTreeNode(child, level+1, writer)
	}
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
	var purls []string
	var versions []string
	for _, file := range request.Files {

		for _, purl := range file.Purls {
			purls = append(purls, purl.Purl)
			versions = append(versions, purl.Requirement)
			continue
		}
	}

	fmt.Println("DTO", dtoRequest)
	visitedNodes := make(map[string]bool)
	tree := make(map[string]*dtos.Dependency)
	dtoDependencies, warn, err := depUc.GetTransientDependencies(purls, versions, visitedNodes, tree, 4)
	purlVersion := fmt.Sprintf("%s@%s", purls[0], versions[0])
	//printNode(tree[purlVersion], 0)
	SaveDependencyTreeToFile(tree[purlVersion], "/Users/agus/Public/tree.json")
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
