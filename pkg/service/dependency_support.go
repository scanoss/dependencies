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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"regexp"
	trasitive_dependencies "scanoss.com/dependencies/pkg/transitive_dependencies"

	pb "github.com/scanoss/papi/api/dependenciesv2"
	"go.uber.org/zap"
	"scanoss.com/dependencies/pkg/dtos"
)

// Structure for storing OTEL metrics.
type metricsCounters struct {
	depFileCounter metric.Int64Counter
	depsCounter    metric.Int64Counter
	depHistogram   metric.Int64Histogram // milliseconds
}

var oltpMetrics = metricsCounters{}

// setupMetrics configures all the metrics recorders for the platform.
func setupMetrics() {
	meter := otel.Meter("scanoss.com/dependencies")
	oltpMetrics.depFileCounter, _ = meter.Int64Counter("deps.file_count", metric.WithDescription("The number of dependency request files received"))
	oltpMetrics.depsCounter, _ = meter.Int64Counter("deps.dep_count", metric.WithDescription("The number of dependency request components received"))
	oltpMetrics.depHistogram, _ = meter.Int64Histogram("deps.req_time", metric.WithDescription("The time taken to run a dependency request (ms)"))
}

// convertDependencyInput converts a Dependency Request structure into an internal Dependency Input struct.
func convertDependencyInput(s *zap.SugaredLogger, request *pb.DependencyRequest) (dtos.DependencyInput, error) {
	data, err := json.Marshal(request)
	if err != nil {
		s.Errorf("Problem marshalling dependency request input: %v", err)
		return dtos.DependencyInput{}, errors.New("problem marshalling dependency input")
	}
	dtoRequest, err := dtos.ParseDependencyInput(s, data)
	if err != nil {
		s.Errorf("Problem parsing dependency request input: %v", err)
		return dtos.DependencyInput{}, errors.New("problem parsing dependency input")
	}
	return dtoRequest, nil
}

// convertDependencyOutput converts an internal Dependency Output structure into a Dependency Response struct.
func convertDependencyOutput(s *zap.SugaredLogger, output dtos.DependencyOutput) (*pb.DependencyResponse, error) {
	data, err := json.Marshal(output)
	if err != nil {
		s.Errorf("Problem marshalling dependency request output: %v", err)
		return &pb.DependencyResponse{}, errors.New("problem marshalling dependency output")
	}
	s.Debugf("Parsed data: %v", string(data))
	var depResp pb.DependencyResponse
	err = json.Unmarshal(data, &depResp)
	if err != nil {
		s.Errorf("Problem unmarshalling dependency request output: %v", err)
		return &pb.DependencyResponse{}, errors.New("problem unmarshalling dependency output")
	}
	return &depResp, nil
}

func convertToTransitiveDependencyInput(s *zap.SugaredLogger, request *pb.TransitiveDependencyRequest) (trasitive_dependencies.TransitiveDependencyInput, error) {
	data, err := json.Marshal(request.Purls)
	if err != nil {
		s.Errorf("Problem marshalling dependency request input: %v", err)
		return trasitive_dependencies.TransitiveDependencyInput{}, errors.New("problem marshalling dependency input")
	}
	s.Debugf("Parsed data: %v", data)
	components, err := dtos.ParseComponentsInput(s, data)
	if err != nil {
		s.Errorf("Problem parsing dependency request input: %v", err)
		return trasitive_dependencies.TransitiveDependencyInput{}, errors.New("problem parsing dependency input")
	}
	return trasitive_dependencies.TransitiveDependencyInput{
		Components: components,
		Ecosystem:  request.Ecosystem,
		Depth:      int(request.Depth),
	}, nil
}

func convertToTransitiveDependencyOutput(s *zap.SugaredLogger, purls []trasitive_dependencies.Purl) (*pb.TransitiveDependencyResponse, error) {

	splitPurlVersionRegEx, _ := regexp.Compile(`(?P<purl>.*?)@(?P<version>.*)`)
	var tdr pb.TransitiveDependencyResponse

	for _, purl := range purls {
		matches := splitPurlVersionRegEx.FindStringSubmatch(string(purl))
		if len(matches) == 0 {
			continue
		}

		// Get the indices of the named groups
		purlIndex := splitPurlVersionRegEx.SubexpIndex("purl")
		versionIndex := splitPurlVersionRegEx.SubexpIndex("version")

		// Check if there's a match and enough capture groups
		if len(matches) < 2 {
			continue
		}

		tdr.Dependencies = append(tdr.Dependencies, &pb.TransitiveDependencyResponse_Dependencies{
			Purl:    matches[purlIndex],
			Version: matches[versionIndex],
		})
	}

	return &tdr, nil
}

func convertToTransitiveDependencyOutput2(s *zap.SugaredLogger, dependencies []trasitive_dependencies.Dependency) (*pb.TransitiveDependencyResponse, error) {
	var tdr pb.TransitiveDependencyResponse
	for _, d := range dependencies {
		tdr.Dependencies = append(tdr.Dependencies, &pb.TransitiveDependencyResponse_Dependencies{
			Purl:    d.Purl,
			Version: d.Version,
		})
	}
	return &tdr, nil
}
