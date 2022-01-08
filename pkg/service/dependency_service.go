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
	"encoding/json"
	"errors"
	"github.com/jmoiron/sqlx"
	common "github.com/scanoss/papi/api/commonv2"
	pb "github.com/scanoss/papi/api/dependenciesv2"
	"log"
	"scanoss.com/dependencies/pkg/dtos"
	"scanoss.com/dependencies/pkg/usecase"
)

type dependencyServer struct {
	pb.DependenciesServer
	db *sqlx.DB
}

func NewDependencyServer(db *sqlx.DB) pb.DependenciesServer {
	return &dependencyServer{db: db}
}

func (d dependencyServer) Echo(ctx context.Context, request *common.EchoRequest) (*common.EchoResponse, error) {
	log.Printf("Received: %v", request.GetMessage())
	return &common.EchoResponse{Message: request.GetMessage()}, nil
}

func (d dependencyServer) GetDependencies(ctx context.Context, request *pb.DependencyRequest) (*pb.DependencyResponse, error) {
	log.Printf("Processing dependency request: %v", request)
	depRequest := request.GetFiles()
	if depRequest == nil || len(depRequest) == 0 {
		statusResp := common.StatusResponse{Status: common.StatusCode_FAILED, Message: "No dependency request data supplied"}
		return &pb.DependencyResponse{Status: &statusResp}, errors.New("no request data supplied")
	}
	dtoRequest, err := convertDependencyInput(request)
	if err != nil {
		statusResp := common.StatusResponse{Status: common.StatusCode_FAILED, Message: "Problem parsing dependency input data"}
		return &pb.DependencyResponse{Status: &statusResp}, errors.New("problem parsing dependency input data")
	}
	conn, err := d.db.Connx(ctx) // Get a connection from the pool
	if err != nil {
		statusResp := common.StatusResponse{Status: common.StatusCode_FAILED, Message: "Failed to get database pool connection"}
		return &pb.DependencyResponse{Status: &statusResp}, errors.New("problem getting database pool connection")
	}
	defer conn.Close()
	depUc := usecase.NewDependencies(ctx, conn)
	dtoDependencies, err := depUc.GetDependencies(dtoRequest)
	if err != nil {
		statusResp := common.StatusResponse{Status: common.StatusCode_FAILED, Message: "Problems encountered extracting dependency data"}
		return &pb.DependencyResponse{Status: &statusResp}, errors.New("problems encountered extracting dependency data")
	}
	log.Printf("Parsed Dependencies: %+v", dtoDependencies)
	depResponse, err := convertDependencyOutput(dtoDependencies)
	if err != nil {
		statusResp := common.StatusResponse{Status: common.StatusCode_FAILED, Message: "Problems encountered extracting dependency data"}
		return &pb.DependencyResponse{Status: &statusResp}, errors.New("problems encountered extracting dependency data")
	}
	statusResp := common.StatusResponse{Status: common.StatusCode_SUCCESS, Message: "Success"}
	return &pb.DependencyResponse{Files: depResponse.Files, Status: &statusResp}, nil
}

func convertDependencyInput(request *pb.DependencyRequest) (dtos.DependencyInput, error) {
	data, err := json.Marshal(request)
	if err != nil {
		log.Printf("Error: Problem marshalling dependency request input: %v", err)
		return dtos.DependencyInput{}, errors.New("problem marshalling dependency input")
	}
	dtoRequest, err := dtos.ParseDependencyInput(data)
	if err != nil {
		log.Printf("Error: Problem parsing dependency request input: %v", err)
		return dtos.DependencyInput{}, errors.New("problem parsing dependency input")
	}
	return dtoRequest, nil
}

func convertDependencyOutput(output dtos.DependencyOutput) (pb.DependencyResponse, error) {
	data, err := json.Marshal(output)
	if err != nil {
		log.Printf("Error: Problem marshalling dependency request output: %v", err)
		return pb.DependencyResponse{}, errors.New("problem marshalling dependency output")
	}
	log.Printf("Parsed data: %v", string(data))
	var depResp pb.DependencyResponse
	err = json.Unmarshal(data, &depResp)
	if err != nil {
		log.Printf("Error: Problem unmarshalling dependency request output: %v", err)
		return pb.DependencyResponse{}, errors.New("problem unmarshalling dependency output")
	}
	return depResp, nil
}
