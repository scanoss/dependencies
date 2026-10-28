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
	"scanoss.com/dependencies/pkg/resolver"
)

type DependencyUseCase struct {
	ctx  context.Context
	s    *zap.SugaredLogger
	db   *sqlx.DB
	res  *resolver.Resolver
	urls *models.ComponentLicensesModel
	lic  *models.LicenseModel
}

// NewDependencies creates a new instance of the Dependency Use Case.
func NewDependencies(ctx context.Context, s *zap.SugaredLogger, db *sqlx.DB, config *myconfig.ServerConfig) *DependencyUseCase {
	res := resolver.New(s, db)
	lic := models.NewLicenseModel(ctx, s, db)
	return &DependencyUseCase{ctx: ctx, s: s,
		db:  db,
		res: res,
		urls: models.NewComponentLicensesModel(ctx, s, db,
			database.NewDBSelectContext(s, db, nil, config.Database.Trace),
			res, models.NewGolangModel(ctx, s, lic),
		),
		lic: lic,
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

		processedComponents, keys := resolver.ResolveComponentsWithIDs(d.ctx, d.s, d.res, file.Purls, 5)

		for i, processedComponent := range processedComponents {
			depOutputs = append(depOutputs, d.dependencyOutput(file.File, processedComponent, keys[i]))
		}
		fileOutput.Dependencies = depOutputs
		depFileOutputs = append(depFileOutputs, fileOutput)
	}
	d.s.Debugf("Output dependencies: %v", depFileOutputs)
	return dtos.DependencyOutput{Files: depFileOutputs}, false, nil
}

// dependencyOutput decorates one resolved component of a dependency file with its
// version, URL and licences. fileName is only used to label log messages.
func (d DependencyUseCase) dependencyOutput(fileName string, processedComponent componentHelper.Component, key resolver.ComponentKey) dtos.DependenciesOutput {
	depOutput := dtos.DependenciesOutput{
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
		return depOutput
	}

	// Look up the component's licences for the version it resolved to
	url, err := d.urls.GetURLsByComponent(processedComponent, key)
	if err != nil {
		d.s.Warnf("Problem encountered extracting URLs for: %v, %v - %v.", fileName, processedComponent.Purl, err)
		depOutput.Status = domain.ComponentStatus{
			Message:    "component not found",
			StatusCode: domain.NoInfo,
		}
		return depOutput
	}

	// Skip components with no license data available
	if len(url.Licenses) == 0 {
		// Preserve the upstream status from the resolver
		if processedComponent.Status.StatusCode != domain.Success {
			return depOutput
		}
		depOutput.Licenses = []dtos.DependencyLicense{}
		depOutput.Status = domain.ComponentStatus{
			StatusCode: domain.NoInfo,
			Message:    "No license information found",
		}
		return depOutput
	}

	if len(url.Version) == 0 {
		depOutput.Status = processedComponent.Status
		// A component with no release to answer for can still borrow a
		// licence from its source or another version; report it flagged.
		if len(url.FallbackMessage) > 0 {
			depOutput.Licenses = d.resolveLicenses(url)
			depOutput.Status = withFallback(depOutput.Status, url.FallbackMessage)
		}
		return depOutput
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
		} else if exact := strings.TrimSpace(processedComponent.Requirement); strings.TrimPrefix(exact, "v") == strings.TrimPrefix(url.Version, "v") {
			// The version the caller named exactly, spelled as they named it. The
			// export drops the leading "v" mining stored (v5.4.5 is now 5.4.5),
			// and a golang version without it is not the version asked for.
			depOutput.Version = exact
			// A version the knowledge base lacks but a fallback answered for (a
			// golang tag, from pkg.go.dev or the GitHub repository) was found.
			if depOutput.Status.StatusCode == domain.VersionNotFound {
				depOutput.Status = domain.ComponentStatus{StatusCode: domain.Success}
			}
		}
	}

	// Fall back to the component URL from the URL lookup if the output has no URL set
	if len(depOutput.URL) == 0 && len(url.URL) > 0 {
		depOutput.URL = url.URL
	}

	depOutput.Licenses = d.resolveLicenses(url)
	if len(url.FallbackMessage) > 0 {
		depOutput.Status = withFallback(depOutput.Status, url.FallbackMessage)
	}
	return depOutput
}

// fallbackStatus is the info code of a licence borrowed from the source component
// or another version. domain has no such constant; the service forwards any
// non-SUCCESS code verbatim, so it reaches the response as the other decoration
// services and scanoss.api report it.
const fallbackStatus domain.StatusCode = "WARNING"

// withFallback marks a status as answered by a licence fallback. A success
// becomes a WARNING naming where the licence came from; a status that already
// reports a problem (REQUIREMENT_NOT_MET) keeps its code, with the fallback added
// to its message, so neither is lost.
func withFallback(status domain.ComponentStatus, message string) domain.ComponentStatus {
	if status.StatusCode == domain.Success || len(status.StatusCode) == 0 {
		return domain.ComponentStatus{StatusCode: fallbackStatus, Message: message}
	}
	status.Message += "; " + strings.ToLower(message[:1]) + message[1:]
	return status
}

// resolveLicenses reports a component's licences: every licence of the release,
// each compound identifier ("MIT/Apache-2.0") split into its parts and looked up by
// name, duplicates dropped.
func (d DependencyUseCase) resolveLicenses(url models.AllURL) []dtos.DependencyLicense {
	var licenses []dtos.DependencyLicense
	seen := map[string]bool{}
	add := func(l dtos.DependencyLicense) {
		key := l.SpdxID + "\x00" + l.Name
		if !seen[key] {
			seen[key] = true
			licenses = append(licenses, l)
		}
	}
	for _, license := range url.Licenses {
		splitLicenses := strings.Split(license.LicenseID, "/")
		if len(splitLicenses) <= 1 {
			add(dtos.DependencyLicense{Name: license.LicenseName, SpdxID: license.LicenseID, IsSpdx: license.IsSpdx})
			continue
		}
		for _, splitLicense := range splitLicenses {
			spl := strings.TrimSpace(splitLicense)
			d.s.Debugf("Searching for split license: %v", spl)
			lic, licErr := d.lic.GetLicenseByName(spl)
			if licErr != nil || len(lic.LicenseName) == 0 {
				if licErr != nil {
					d.s.Warnf("Problem encountered searching for license %v (%v): %v", spl, splitLicense, licErr)
				}
				add(dtos.DependencyLicense{Name: spl, SpdxID: spl, IsSpdx: false})
			} else {
				add(dtos.DependencyLicense{Name: lic.LicenseName, SpdxID: lic.LicenseID, IsSpdx: lic.IsSpdx})
			}
		}
	}
	return licenses
}
