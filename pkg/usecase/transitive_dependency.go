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
	"fmt"
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
}

// NewDependencies creates a new instance of the Dependency Use Case.
func NewTransitiveDependencies(ctx context.Context, logger *zap.SugaredLogger, db *sqlx.DB, config *myconfig.ServerConfig) *TransitiveDependencyUseCase {
	return &TransitiveDependencyUseCase{
		ctx:             ctx,
		logger:          logger,
		db:              db,
		dependencyModel: models.NewDependencyModel(ctx, logger, db),
	}
}

// GetDependencies takes the Dependency Input request, searches for component details and returns a Dependency Output struct.
func (d TransitiveDependencyUseCase) GetTransitiveDependencies(input transitiveDep.TransitiveDependencyInput) ([]transitiveDep.Dependency, error) {
	depGraph := transitiveDep.NewDepGraph()

	// callback
	adaptDependencyToGraph := func(result transitiveDep.Result) {
		for _, purl := range result.Purls {
			dep := transitiveDep.GetPurlFromPackageName(result.Parent, input.Ecosystem)
			transitive := transitiveDep.GetPurlFromPackageName(purl, input.Ecosystem)
			depGraph.Insert(
				transitiveDep.Dependency{Purl: transitiveDep.GetPurlWithoutVersion(dep), Version: dep.Version},
				transitiveDep.Dependency{Purl: transitiveDep.GetPurlWithoutVersion(transitive), Version: transitive.Version})
		}
	}

	transitiveDependencyCollector := transitiveDep.NewDependencyCollector(adaptDependencyToGraph,
		transitiveDep.DependencyCollectorCfg{
			MaxWorkers:    20,    // this should be taken from config
			MaxQueueLimit: 20000, // this should be taken from config
		}, models.NewDependencyModel(d.ctx, d.logger, d.db))

	transitiveDependencyCollector.InitJobs(input)
	transitiveDependencyCollector.Start()

	fmt.Print(depGraph.String())

	return depGraph.Flatten(), nil
}
