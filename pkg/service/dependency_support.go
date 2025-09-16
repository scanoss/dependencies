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
	"fmt"

	"github.com/package-url/packageurl-go"
	pb "github.com/scanoss/papi/api/dependenciesv2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
	"scanoss.com/dependencies/pkg/config"
	"scanoss.com/dependencies/pkg/dtos"
	"scanoss.com/dependencies/pkg/errors"
	"scanoss.com/dependencies/pkg/shared"
	trasitive_dependencies "scanoss.com/dependencies/pkg/transdep"
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
		return dtos.DependencyInput{}, errors.NewInternalError("problem marshalling dependency input", err)
	}
	dtoRequest, err := dtos.ParseDependencyInput(s, data)
	if err != nil {
		s.Errorf("Problem parsing dependency request input: %v", err)
		return dtos.DependencyInput{}, errors.NewBadRequestError("problem parsing dependency input", err)
	}
	return dtoRequest, nil
}

// convertDependencyOutput converts an internal Dependency Output structure into a Dependency Response struct.
func convertDependencyOutput(s *zap.SugaredLogger, output dtos.DependencyOutput) (*pb.DependencyResponse, error) {
	data, err := json.Marshal(output)
	if err != nil {
		s.Errorf("Problem marshalling dependency request output: %v", err)
		return &pb.DependencyResponse{}, errors.NewInternalError("problem marshalling dependency output", err)
	}
	s.Debugf("Parsed data: %v", string(data))
	var depResp pb.DependencyResponse
	err = json.Unmarshal(data, &depResp)
	if err != nil {
		s.Errorf("Problem unmarshalling dependency request output: %v", err)
		return &pb.DependencyResponse{}, errors.NewInternalError("problem unmarshalling dependency output", err)
	}
	return &depResp, nil
}

// determineEcosystem extracts and validates the ecosystem from transitive dependency components.
// It ensures all components belong to the same ecosystem and that the ecosystem is registered.
// Returns the ecosystem name or an error if validation fails.
func determineEcosystem(transitiveDependencyDTO dtos.TransitiveDependencyDTO) (string, error) {
	if len(transitiveDependencyDTO.Components) == 0 {
		return "", errors.NewBadRequestError("no components provided to determine ecosystem", nil)
	}

	ecosystems := make(map[string]bool)
	for _, component := range transitiveDependencyDTO.Components {
		if component.Purl == "" {
			return "", errors.NewBadRequestError("component purl cannot be empty", nil)
		}
		p, err := packageurl.FromString(component.Purl)
		if err != nil {
			fmt.Printf("Error parsing purl: %v\n", err)
			continue
		}
		ecosystems[p.Type] = true

		// Early exit if we detect multiple ecosystems
		if len(ecosystems) > 1 {
			var ecosystemTypes []string
			for ecosystem := range ecosystems {
				ecosystemTypes = append(ecosystemTypes, ecosystem)
			}
			return "", errors.NewBadRequestError(fmt.Sprintf("multiple ecosystems found, expected single ecosystem: %v", ecosystemTypes), nil)
		}
	}

	if len(ecosystems) == 0 {
		return "", errors.NewBadRequestError("no valid ecosystems found in components", nil)
	}

	// Get the single ecosystem
	var ecosystem string
	for key := range ecosystems {
		ecosystem = key
		break
	}
	// Validate that the ecosystem is registered
	if _, ok := shared.RegisteredEcosystems[ecosystem]; !ok {
		return "", errors.NewBadRequestError(fmt.Sprintf("invalid ecosystem: '%s'. Supported ecosystems: 'composer', 'crates', 'maven', 'npm' and 'gem'", ecosystem), nil)
	}
	return ecosystem, nil
}

func validateTransitiveDependencyRequest(request *pb.TransitiveDependencyRequest) error {
	if len(request.Components) == 0 {
		return errors.NewBadRequestError("'components' field is required and must contain at least one component", nil)
	}
	return nil
}

func convertProtobufToDTO(request *pb.TransitiveDependencyRequest) dtos.TransitiveDependencyDTO {
	components := make([]dtos.ComponentDTO, len(request.Components))
	for i, component := range request.Components {
		components[i] = dtos.ComponentDTO{
			Purl:        component.Purl,
			Requirement: component.Requirement,
		}
	}

	var depth *int
	if request.Depth != 0 {
		depthValue := int(request.Depth)
		depth = &depthValue
	}

	var limit *int
	if request.Limit != 0 {
		limitValue := int(request.Limit)
		limit = &limitValue
	}

	return dtos.TransitiveDependencyDTO{
		Depth:      depth,
		Ecosystem:  "", // Will be set later
		Components: components,
		Limit:      limit,
	}
}

func convertToTransitiveDependencyDTO(
	s *zap.SugaredLogger,
	config *config.ServerConfig,
	request *pb.TransitiveDependencyRequest) (dtos.TransitiveDependencyDTO, error) {
	if err := validateTransitiveDependencyRequest(request); err != nil {
		return dtos.TransitiveDependencyDTO{}, err
	}

	transitiveDepDTO := convertProtobufToDTO(request)
	s.Debugf("Converted transitive dependency request: %v", transitiveDepDTO)

	var invalidPurls []string
	var validComponents []dtos.ComponentDTO
	for _, component := range transitiveDepDTO.Components {
		_, err := trasitive_dependencies.ExtractPackageIdentifierFromPurl(component.Purl)
		if err != nil {
			invalidPurls = append(invalidPurls, component.Purl)
			continue
		}
		validComponents = append(validComponents, component)
	}

	fmt.Printf("Valid components: %v\n", validComponents)
	fmt.Printf("Invalid components: %v\n", invalidPurls)
	if len(validComponents) == 0 && len(invalidPurls) > 0 {
		return dtos.TransitiveDependencyDTO{}, errors.NewBadRequestError(fmt.Sprintf("invalid purls: %v", invalidPurls), nil)
	}

	transitiveDepDTO.Components = validComponents

	// Get max depth limit
	depthLimit := trasitive_dependencies.GetMaxLimit(config.TransitiveResources.MaxDepth,
		config.TransitiveResources.DefaultDepth, transitiveDepDTO.Depth)
	// Get max response limit
	responseLimit := trasitive_dependencies.GetMaxLimit(config.TransitiveResources.MaxResponseSize,
		config.TransitiveResources.DefaultResponseSize, transitiveDepDTO.Limit)

	ecosystem, err := determineEcosystem(transitiveDepDTO)
	if err != nil {
		return dtos.TransitiveDependencyDTO{}, err
	}
	transitiveDepDTO.Ecosystem = ecosystem
	transitiveDepDTO.Limit = &responseLimit
	transitiveDepDTO.Depth = &depthLimit

	return transitiveDepDTO, nil
}

func convertToTransitiveDependencyOutput(dependencies []trasitive_dependencies.Dependency) *pb.TransitiveDependencyResponse {
	var tdr pb.TransitiveDependencyResponse
	for _, d := range dependencies {
		tdr.Dependencies = append(tdr.Dependencies, &pb.TransitiveDependencyResponse_Component{
			Purl:        d.Purl,
			Version:     d.Version,
			Requirement: d.Version,
		})
	}
	return &tdr
}
