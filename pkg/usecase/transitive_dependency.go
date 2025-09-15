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

package usecase

import (
	"context"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	myconfig "scanoss.com/dependencies/pkg/config"
	"scanoss.com/dependencies/pkg/dtos"
	"scanoss.com/dependencies/pkg/errors"
	"scanoss.com/dependencies/pkg/models"
	transitiveDep "scanoss.com/dependencies/pkg/transdep"
)

type DependencyJobCollection struct {
	DependencyJobs []transitiveDep.DependencyJob
	ResponseLimit  int
}

// toJobCollection converts a TransitiveDependencyDTO to DependencyJobCollection.
func toJobCollection(s *zap.SugaredLogger, dto dtos.TransitiveDependencyDTO) (DependencyJobCollection, error) {
	dependencyJobs := make([]transitiveDep.DependencyJob, 0, len(dto.Components))
	for _, component := range dto.Components {
		purlName, err := transitiveDep.ExtractPackageIdentifierFromPurl(component.Purl)
		if err != nil {
			s.Errorf("failed to convert purl:%v, %v", component.Purl, err)
			continue
		}
		dependencyJobs = append(dependencyJobs, transitiveDep.DependencyJob{
			PurlName:    purlName,
			Version:     component.Requirement,
			Requirement: component.Requirement,
			Ecosystem:   dto.Ecosystem,
			Depth:       *dto.Depth,
		})
	}

	if len(dependencyJobs) == 0 {
		return DependencyJobCollection{}, errors.NewBadRequestError(
			"no valid dependency jobs could be created from input",
			nil,
		)
	}

	return DependencyJobCollection{
		DependencyJobs: dependencyJobs,
		ResponseLimit:  *dto.Limit,
	}, nil
}

type TransitiveDependencyUseCase struct {
	ctx             context.Context
	S               *zap.SugaredLogger
	db              *sqlx.DB
	dependencyModel *models.DependencyModel
	config          *myconfig.ServerConfig
}

// NewTransitiveDependencies creates a new instance of the Dependency Use Case.
func NewTransitiveDependencies(ctx context.Context, s *zap.SugaredLogger, db *sqlx.DB, config *myconfig.ServerConfig) *TransitiveDependencyUseCase {
	return &TransitiveDependencyUseCase{
		ctx:             ctx,
		S:               s,
		db:              db,
		dependencyModel: models.NewDependencyModel(ctx, s, db),
		config:          config,
	}
}

func (d TransitiveDependencyUseCase) createEntryDependenciesIndex(dependencyJobs []transitiveDep.DependencyJob) map[string]struct{} {
	inputSet := make(map[string]struct{})
	for _, dj := range dependencyJobs {
		dep, err := transitiveDep.ExtractDependencyFromJob(dj)
		if err != nil {
			d.S.Errorf("failed to convert dependency:%v, %v", dj, err)
			continue
		}
		inputSet[dep.Purl+"@"+dep.Version] = struct{}{}
	}
	return inputSet
}

// GetTransitiveDependencies takes the Dependency Input request, searches for component details and returns a Dependency Output struct.
func (d TransitiveDependencyUseCase) GetTransitiveDependencies(s *zap.SugaredLogger, transitiveDependencyDTO dtos.TransitiveDependencyDTO) ([]transitiveDep.Dependency, error) {
	jobCollection, err := toJobCollection(s, transitiveDependencyDTO)
	if err != nil {
		return nil, err
	}
	depGraph := transitiveDep.NewDepGraph()
	entryDependenciesIndex := d.createEntryDependenciesIndex(jobCollection.DependencyJobs)
	// Increase the max response size to account for entry dependencies that will be filtered out later
	responseSize := jobCollection.ResponseLimit + len(entryDependenciesIndex)
	dependencyCollectorCfg := transitiveDep.DependencyCollectorCfg{
		MaxWorkers:    d.config.TransitiveResources.MaxWorkers,
		MaxQueueLimit: responseSize,
		TimeOut:       d.config.TransitiveResources.TimeOut,
	}
	transitiveDependencyCollector := transitiveDep.NewDependencyCollector(
		d.ctx,
		transitiveDep.ProcessCollectorResult(d.S, depGraph, responseSize),
		dependencyCollectorCfg,
		models.NewDependencyModel(d.ctx, d.S, d.db),
		d.S)

	err = transitiveDependencyCollector.InitJobs(jobCollection.DependencyJobs)
	if err != nil {
		s.Errorf("Error initializing transitive dependencies jobs: %v", err)
		// Default to internal error for unknown errors
		return nil, errors.NewInternalError("failed to initialize dependency jobs", err)
	}

	transitiveDependencyCollector.Start()
	var transitiveDependencies []transitiveDep.Dependency
	for _, dep := range depGraph.Flatten() {
		if _, ok := entryDependenciesIndex[dep.Purl+"@"+dep.Version]; !ok {
			transitiveDependencies = append(transitiveDependencies, dep)
		}
	}

	// Check if we found any dependencies
	if len(transitiveDependencies) == 0 {
		return nil, errors.NewNotFoundError("transitive dependencies for the given components")
	}

	return transitiveDependencies, nil
}
