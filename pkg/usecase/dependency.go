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
	"errors"
	"strings"

	"github.com/scanoss/go-models-helper/pkg/models"
	"scanoss.com/dependencies/pkg/dtos"
)

type DependencyUseCase struct {
	ctx    context.Context
	models *models.ScanossModels
}

// NewDependencies creates a new instance of the Dependency Use Case.
func NewDependencies(ctx context.Context, models *models.ScanossModels) *DependencyUseCase {
	return &DependencyUseCase{
		ctx:    ctx,
		models: models,
	}
}

// GetDependencies takes the Dependency Input request, searches for component details and returns a Dependency Output struct.
func (d DependencyUseCase) GetDependencies(request dtos.DependencyInput) (dtos.DependencyOutput, bool, error) {
	var depFileOutputs []dtos.DependencyFileOutput
	var problems = false
	d.models.Logger().Infof("Processing %v dependency files...", len(request.Files))
	for _, file := range request.Files {
		var fileOutput dtos.DependencyFileOutput
		fileOutput.File = file.File
		fileOutput.ID = "dependency"
		fileOutput.Status = "pending"
		var depOutputs []dtos.DependenciesOutput
		d.models.Logger().Infof("Processing %v purls for %v...", len(file.Purls), file.File)
		for _, purl := range file.Purls {
			if len(purl.Purl) == 0 {
				d.models.Logger().Infof("Empty Purl string supplied for: %v. Skipping", file.File)
				continue
			}
			var depOutput dtos.DependenciesOutput
			depOutput.Purl = strings.Split(purl.Purl, "@")[0] // Remove any version specific info from the PURL
			url, err := d.models.AllUrls.GetURLsByPurlString(purl.Purl, purl.Requirement)
			if err != nil {
				d.models.Logger().Warnf("Problem encountered extracting URLs for: %v, %v - %v.", file.File, purl, err)
				problems = true // Record this as a warning
				continue
			}

			// Avoids empty version
			if len(url.Version) > 0 {
				depOutput.Version = url.Version
			} else {
				purlParts := strings.Split(purl.Purl, "@")
				if len(purlParts) > 1 {
					depOutput.Version = purlParts[1]
				} else if len(purl.Requirement) > 0 {
					depOutput.Version = purl.Requirement
				} else {
					depOutput.Version = "unknown"
				}
			}

			depOutput.Component = url.Component
			depOutput.URL = url.URL
			var licenses []dtos.DependencyLicense
			splitLicenses := strings.Split(url.LicenseID, "/") // Check to see if we have multiple licenses returned
			if len(splitLicenses) > 1 {
				for _, splitLicense := range splitLicenses {
					spl := strings.TrimSpace(splitLicense)
					d.models.Logger().Debugf("Searching for split license: %v", spl)
					lic, err := d.models.Licenses.GetLicenseByName(spl, false)
					if err != nil || len(lic.LicenseName) == 0 {
						if err != nil {
							d.models.Logger().Warnf("Problem encountered searching for license %v (%v): %v", spl, splitLicense, err)
						}
						var license dtos.DependencyLicense
						license.Name = spl
						license.SpdxID = spl
						license.IsSpdx = false
						licenses = append(licenses, license)
					} else {
						var license dtos.DependencyLicense
						license.Name = lic.LicenseName
						license.SpdxID = lic.LicenseID
						license.IsSpdx = lic.IsSpdx
						licenses = append(licenses, license)
					}
				}
			} else {
				var license dtos.DependencyLicense
				license.Name = url.License
				license.SpdxID = url.LicenseID
				license.IsSpdx = url.IsSpdx
				licenses = append(licenses, license)
			}
			depOutput.Licenses = licenses
			depOutputs = append(depOutputs, depOutput)
		}
		fileOutput.Dependencies = depOutputs
		depFileOutputs = append(depFileOutputs, fileOutput)
	}
	d.models.Logger().Debugf("Output dependencies: %v", depFileOutputs)
	if problems {
		d.models.Logger().Warnf("Encountered issues while processing dependencies: %v", request)
		return dtos.DependencyOutput{Files: depFileOutputs}, true, errors.New("encountered issues while processing dependencies")
	}

	return dtos.DependencyOutput{Files: depFileOutputs}, false, nil
}
