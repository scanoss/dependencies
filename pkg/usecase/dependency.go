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

// Package usecase implements the business logic for dependency operations.
package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	componentHelper "github.com/scanoss/go-component-helper/componenthelper"
	"github.com/scanoss/go-grpc-helper/pkg/grpc/database"
	"github.com/scanoss/go-grpc-helper/pkg/grpc/domain"
	purlutils "github.com/scanoss/go-purl-helper/pkg"
	"go.uber.org/zap"
	myconfig "scanoss.com/dependencies/pkg/config"
	"scanoss.com/dependencies/pkg/dtos"
	"scanoss.com/dependencies/pkg/models"
)

type DependencyUseCase struct {
	ctx     context.Context
	s       *zap.SugaredLogger
	db      *sqlx.DB
	allUrls *models.AllUrlsModel
	lic     *models.LicenseModel
}

// NewDependencies creates a new instance of the Dependency Use Case.
func NewDependencies(ctx context.Context, s *zap.SugaredLogger, db *sqlx.DB, config *myconfig.ServerConfig) *DependencyUseCase {
	return &DependencyUseCase{ctx: ctx, s: s,
		db: db,
		allUrls: models.NewAllURLModel(ctx, s, db, models.NewProjectModel(ctx, s, db),
			models.NewGolangProjectModel(ctx, s, db, config),
			models.NewMineModel(ctx, s, db),
			database.NewDBSelectContext(s, db, nil, config.Database.Trace),
		),
		lic: models.NewLicenseModel(ctx, s, db),
	}
}

// GetDependencies takes the Dependency Input request, searches for component details and returns a Dependency Output struct.
func (d DependencyUseCase) GetDependencies(request dtos.DependencyInput) (dtos.DependencyOutput, bool, error) {
	var depFileOutputs []dtos.DependencyFileOutput
	d.s.Infof("Processing %v dependency files...", len(request.Files))
	for _, file := range request.Files {
		var fileOutput dtos.DependencyFileOutput
		fileOutput.File = file.File
		fileOutput.ID = "dependency"
		fileOutput.Status = "pending"
		var depOutputs []dtos.DependenciesOutput
		d.s.Infof("Processing %v purls for %v...", len(file.Purls), file.File)

		var depOutput dtos.DependenciesOutput
		processedComponents := componentHelper.GetComponentsVersion(componentHelper.ComponentVersionCfg{
			MaxWorkers: 5,
			DB:         d.db,
			Ctx:        d.ctx,
			S:          d.s,
			Input:      file.Purls,
		})

		for _, processedComponent := range processedComponents {
			depOutput = dtos.DependenciesOutput{
				Purl:        processedComponent.Purl,
				Requirement: processedComponent.Requirement,
				Version:     processedComponent.Version,
				URL:         processedComponent.URL,
				Component:   processedComponent.Name,
				Status:      processedComponent.Status,
			}
			// avoid processing invalid components not found components
			if processedComponent.Status.StatusCode == domain.InvalidPurl {
				depOutput.Status = processedComponent.Status
				depOutput.Component = processedComponent.Name
				depOutputs = append(depOutputs, depOutput)
				continue
			}

			// Look up component details (URL, license, version) from the all_urls table
			url, err := d.allUrls.GetURLsByPurlString(processedComponent)
			if err != nil {
				d.s.Warnf("Problem encountered extracting URLs for: %v, %v - %v.", file.File, processedComponent.Purl, err)
				depOutput.Status = domain.ComponentStatus{
					Message:    "component not found",
					StatusCode: domain.NoInfo,
				}
				depOutputs = append(depOutputs, depOutput)
				continue
			}

			// Skip components with no license data available
			if url.License == "" {
				// Preserve the upstream status from go-component-helper
				if processedComponent.Status.StatusCode != domain.Success {
					depOutputs = append(depOutputs, depOutput)
					continue
				}
				depOutput.Licenses = []dtos.DependencyLicense{}
				depOutput.Status = domain.ComponentStatus{
					StatusCode: domain.NoInfo,
					Message:    "No license information found",
				}
				depOutputs = append(depOutputs, depOutput)
				continue
			}

			if len(url.Version) == 0 {
				depOutput.Status = processedComponent.Status
				depOutputs = append(depOutputs, depOutput)
				continue
			}

			depOutput.Version = url.Version
			if processedComponent.Requirement != "" {
				v := purlutils.GetVersionFromReqOperator(processedComponent.Requirement)
				// Compare versions ignoring the "v" prefix (e.g., "v1.2.3" == "1.2.3")
				// If the version does not satisfy the requirement, mark the status accordingly.
				if strings.TrimPrefix(url.Version, "v") != strings.TrimPrefix(v, "v") {
					depOutput.Status = domain.ComponentStatus{
						StatusCode: domain.RequirementNotMet,
						Message:    fmt.Sprintf("Requirement not met, showing information for version '%s'", url.Version),
					}
					depOutput.Version = v
				}
			}

			// Fall back to the component URL from the URL lookup if the output has no URL set
			if len(depOutput.URL) == 0 && len(url.URL) > 0 {
				depOutput.URL = url.URL
			}

			depOutput.Licenses = d.resolveLicenses(url)
			depOutputs = append(depOutputs, depOutput)
		}
		fileOutput.Dependencies = depOutputs
		depFileOutputs = append(depFileOutputs, fileOutput)
	}
	d.s.Debugf("Output dependencies: %v", depFileOutputs)
	return dtos.DependencyOutput{Files: depFileOutputs}, false, nil
}

// resolveLicenses resolves the license information for a component URL,
// handling compound license IDs separated by "/".
func (d DependencyUseCase) resolveLicenses(url models.AllURL) []dtos.DependencyLicense {
	var licenses []dtos.DependencyLicense
	splitLicenses := strings.Split(url.LicenseID, "/")
	if len(splitLicenses) <= 1 {
		return []dtos.DependencyLicense{{Name: url.License, SpdxID: url.LicenseID, IsSpdx: url.IsSpdx}}
	}
	for _, splitLicense := range splitLicenses {
		spl := strings.TrimSpace(splitLicense)
		d.s.Debugf("Searching for split license: %v", spl)
		lic, licErr := d.lic.GetLicenseByName(spl, false)
		if licErr != nil || len(lic.LicenseName) == 0 {
			if licErr != nil {
				d.s.Warnf("Problem encountered searching for license %v (%v): %v", spl, splitLicense, licErr)
			}
			licenses = append(licenses, dtos.DependencyLicense{Name: spl, SpdxID: spl, IsSpdx: false})
		} else {
			licenses = append(licenses, dtos.DependencyLicense{Name: lic.LicenseName, SpdxID: lic.LicenseID, IsSpdx: lic.IsSpdx})
		}
	}
	return licenses
}
