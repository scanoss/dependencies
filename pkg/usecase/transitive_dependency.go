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
	transitiveDep "scanoss.com/dependencies/pkg/transdep"
)

type DependencyJobCollection struct {
	DependencyJobs []transitiveDep.DependencyJob
	ResponseLimit  int
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
func (d TransitiveDependencyUseCase) GetTransitiveDependencies(depJobCollection DependencyJobCollection) ([]transitiveDep.Dependency, error) {
	// creates new dependency graph struct
	depGraph := transitiveDep.NewDepGraph()
	entryDependenciesIndex := d.createEntryDependenciesIndex(depJobCollection.DependencyJobs)
	// Increase the max response size to account for entry dependencies that will be filtered out later
	responseSize := depJobCollection.ResponseLimit + len(entryDependenciesIndex)
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
	transitiveDependencyCollector.InitJobs(depJobCollection.DependencyJobs)
	transitiveDependencyCollector.Start()
	var transitiveDependencies []transitiveDep.Dependency
	for _, d := range depGraph.Flatten() {
		if _, ok := entryDependenciesIndex[d.Purl+"@"+d.Version]; !ok {
			transitiveDependencies = append(transitiveDependencies, d)
		}
	}
	return transitiveDependencies, nil
}
