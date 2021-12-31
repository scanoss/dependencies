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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"scanoss.com/dependencies/pkg/dtos"
	"scanoss.com/dependencies/pkg/usecase"
)

const (
	// apiVersion is version of API is provided by server
	apiVersion = "v2" // TODO keep or remove?
)

type dependencyServer struct {
	pb.DependenciesServer
	db    *sqlx.DB
	depUc *usecase.DependencyUseCase
}

var (
	depResp = "{\n  \"audit-workbench-master/package.json\": {\n    \"id\": \"dependency\",\n    \"status\": \"pending\",\n    \"dependencies\": [\n      {\n        \"purl\": \"abort-controller\",\n        \"component\": \"abort-controller\",\n        \"vendor\": \"Toru Nagashima\",\n        \"version\": \"\",\n        \"license\": [\n          {\n            \"name\": \"MIT\"\n          }\n        ]\n      },\n      {\n        \"purl\": \"chart.js\",\n        \"component\": \"chart.js\",\n        \"vendor\": \"npmjs\",\n        \"version\": \"\",\n        \"license\": [\n          {\n            \"name\": \"MIT\"\n          }\n        ]\n      }\n    ]\n  }\n}"
)

func NewDependencyServer(db *sqlx.DB) pb.DependenciesServer {
	return &dependencyServer{db: db, depUc: usecase.NewDependencies(db)}
}

// checkAPI checks if the API version requested by client is supported by server
func (d *dependencyServer) checkAPI(api string) error { // TODO is this even required?
	// API version is "" means use current version of the service
	if len(api) > 0 {
		if apiVersion != api {
			return status.Errorf(codes.Unimplemented,
				"unsupported API version: service implements API version '%s', but asked for '%s'", apiVersion, api)
		}
	}
	return nil
}

func (d dependencyServer) Echo(ctx context.Context, request *common.EchoRequest) (*common.EchoResponse, error) {
	log.Printf("Received: %v", request.GetMessage())
	return &common.EchoResponse{Message: request.GetMessage()}, nil
}

func (d dependencyServer) GetDependencies(ctx context.Context, request *pb.DependencyRequest) (*pb.DependencyResponse, error) {
	log.Printf("Processing dependency request: %v", request)
	dependencies := request.GetDependencies()
	if len(dependencies) == 0 {
		statusResp := common.StatusResponse{Status: common.StatusCode_FAILED, Message: "No dependency request data supplied"}
		return &pb.DependencyResponse{Dependencies: "", Status: &statusResp}, errors.New("no request data supplied")
	}
	dtoRequest, err := convertDependencyInput(request)
	if err != nil {
		statusResp := common.StatusResponse{Status: common.StatusCode_FAILED, Message: "Problem parsing dependency input data"}
		return &pb.DependencyResponse{Dependencies: "", Status: &statusResp}, errors.New("problem parsing dependency input data")
	}
	d.depUc.GetDependencies(dtoRequest)

	// TODO add the actual dependency lookup here
	statusResp := common.StatusResponse{Status: common.StatusCode_SUCCESS, Message: "it worked"}
	return &pb.DependencyResponse{Dependencies: depResp, Status: &statusResp}, nil
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

func convertDependencyOutput() {

}
