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
	"scanoss.com/dependencies/pkg/models"
	transitiveDep "scanoss.com/dependencies/pkg/transitive_dependencies"
)

type TransitiveDependencyUseCase struct {
	ctx             context.Context
	logger          *zap.SugaredLogger
	db              *sqlx.DB
	dependencyModel *models.DependencyModel
	config          *myconfig.ServerConfig
}

// NewTransitiveDependencies creates a new instance of the Dependency Use Case.
func NewTransitiveDependencies(ctx context.Context, logger *zap.SugaredLogger, db *sqlx.DB, config *myconfig.ServerConfig) *TransitiveDependencyUseCase {
	return &TransitiveDependencyUseCase{
		ctx:             ctx,
		logger:          logger,
		db:              db,
		dependencyModel: models.NewDependencyModel(ctx, logger, db),
		config:          config,
	}
}

func (d TransitiveDependencyUseCase) createEntryDependenciesIndex(dependencyJobs []transitiveDep.DependencyJob) map[string]struct{} {
	inputSet := make(map[string]struct{})
	for _, dj := range dependencyJobs {
		dep, err := transitiveDep.ExtractDependencyFromJob(dj)
		if err != nil {
			d.logger.Errorf("failed to convert dependency:%v, %v", dj, err)
			continue
		}
		inputSet[dep.Purl+"@"+dep.Version] = struct{}{}
	}
	return inputSet
}

// GetTransitiveDependencies takes the Dependency Input request, searches for component details and returns a Dependency Output struct.
func (d TransitiveDependencyUseCase) GetTransitiveDependencies(dependencyJobs []transitiveDep.DependencyJob) ([]transitiveDep.Dependency, error) {
	// creates new dependency graph struct
	depGraph := transitiveDep.NewDepGraph()
	entryDependenciesIndex := d.createEntryDependenciesIndex(dependencyJobs)
	// Increase the max response size to account for entry dependencies that will be filtered out later
	maxDependencyResponseSize := d.config.TransitiveResources.MaxResponseSize + len(entryDependenciesIndex)

	// callback to handle dependency collector results
	adaptDependencyToGraph := func(result transitiveDep.Result) bool {
		parentDep, err := transitiveDep.ExtractDependencyFromJob(result.Parent)
		if err != nil {
			d.logger.Errorf("failed to convert dependency:%v, %v", result.Parent, err)
			return false
		}
		for _, td := range result.TransitiveDependencies {
			var tDep = transitiveDep.Dependency{}
			// Get purls for parent and transitive dependency
			tDep, err = transitiveDep.ExtractDependencyFromJob(td)
			if err != nil {
				d.logger.Errorf("failed to convert transitive dependency:%v, %v", td.PurlName, err)
				continue
			}
			// Stop if max limit response is reached
			if depGraph.GetDependenciesCount() == maxDependencyResponseSize {
				return true
			}
			// Insert relationship into dependency graph
			depGraph.Insert(parentDep, tDep)
		}
		return false
	}

	dependencyCollectorCfg := transitiveDep.DependencyCollectorCfg{
		MaxWorkers:    d.config.TransitiveResources.MaxWorkers,
		MaxQueueLimit: d.config.TransitiveResources.MaxQueueSize,
	}
	transitiveDependencyCollector := transitiveDep.NewDependencyCollector(
		d.ctx,
		adaptDependencyToGraph,
		dependencyCollectorCfg,
		models.NewDependencyModel(d.ctx, d.logger, d.db),
		d.logger)
	transitiveDependencyCollector.InitJobs(dependencyJobs)
	transitiveDependencyCollector.Start()

	var transitiveDependencies []transitiveDep.Dependency
	for _, d := range depGraph.Flatten() {
		if _, ok := entryDependenciesIndex[d.Purl+"@"+d.Version]; !ok {
			transitiveDependencies = append(transitiveDependencies, d)
		}
	}
	return transitiveDependencies, nil
}
