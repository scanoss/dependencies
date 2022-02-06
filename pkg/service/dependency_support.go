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

package service

import (
	"encoding/json"
	"errors"
	pb "github.com/scanoss/papi/api/dependenciesv2"
	"scanoss.com/dependencies/pkg/dtos"
	zlog "scanoss.com/dependencies/pkg/logger"
)

func convertDependencyInput(request *pb.DependencyRequest) (dtos.DependencyInput, error) {
	data, err := json.Marshal(request)
	if err != nil {
		zlog.S.Errorf("Problem marshalling dependency request input: %v", err)
		return dtos.DependencyInput{}, errors.New("problem marshalling dependency input")
	}
	dtoRequest, err := dtos.ParseDependencyInput(data)
	if err != nil {
		zlog.S.Errorf("Problem parsing dependency request input: %v", err)
		return dtos.DependencyInput{}, errors.New("problem parsing dependency input")
	}
	return dtoRequest, nil
}

func convertDependencyOutput(output dtos.DependencyOutput) (*pb.DependencyResponse, error) {
	data, err := json.Marshal(output)
	if err != nil {
		zlog.S.Errorf("Problem marshalling dependency request output: %v", err)
		return &pb.DependencyResponse{}, errors.New("problem marshalling dependency output")
	}
	zlog.S.Debugf("Parsed data: %v", string(data))
	var depResp pb.DependencyResponse
	err = json.Unmarshal(data, &depResp)
	if err != nil {
		zlog.S.Errorf("Problem unmarshalling dependency request output: %v", err)
		return &pb.DependencyResponse{}, errors.New("problem unmarshalling dependency output")
	}
	return &depResp, nil
}
