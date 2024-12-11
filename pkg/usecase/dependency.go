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
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/scanoss/go-grpc-helper/pkg/grpc/database"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	myconfig "scanoss.com/dependencies/pkg/config"
	"scanoss.com/dependencies/pkg/dtos"
	"scanoss.com/dependencies/pkg/models"
)

type DependencyUseCase struct {
	ctx      context.Context
	s        *zap.SugaredLogger
	conn     *sqlx.Conn
	allUrls  *models.AllUrlsModel
	lic      *models.LicenseModel
	depModel *models.TransientDependencyModel
}

// NewDependencies creates a new instance of the Dependency Use Case.
func NewDependencies(ctx context.Context, s *zap.SugaredLogger, conn *sqlx.Conn, config *myconfig.ServerConfig) *DependencyUseCase {
	return &DependencyUseCase{ctx: ctx, s: s, conn: conn,
		allUrls: models.NewAllURLModel(ctx, s, conn, models.NewProjectModel(ctx, s, conn),
			models.NewGolangProjectModel(ctx, s, conn, config),
			database.NewDBSelectContext(s, conn, config.Database.Trace),
		),
		lic:      models.NewLicenseModel(ctx, s, conn),
		depModel: models.NewTransientModel(ctx, s, conn),
	}
}

// GetDependencies takes the Dependency Input request, searches for component details and returns a Dependency Output struct.
func (d DependencyUseCase) GetDependencies(request dtos.DependencyInput) (dtos.DependencyOutput, bool, error) {
	var depFileOutputs []dtos.DependencyFileOutput
	var problems = false
	d.s.Infof("Processing %v dependency files...", len(request.Files))
	for _, file := range request.Files {
		var fileOutput dtos.DependencyFileOutput
		fileOutput.File = file.File
		fileOutput.ID = "dependency"
		fileOutput.Status = "pending"
		var depOutputs []dtos.DependenciesOutput
		d.s.Infof("Processing %v purls for %v...", len(file.Purls), file.File)
		for _, purl := range file.Purls {
			if len(purl.Purl) == 0 {
				d.s.Infof("Empty Purl string supplied for: %v. Skipping", file.File)
				continue
			}
			var depOutput dtos.DependenciesOutput
			depOutput.Purl = strings.Split(purl.Purl, "@")[0] // Remove any version specific info from the PURL
			url, err := d.allUrls.GetURLsByPurlString(purl.Purl, purl.Requirement)
			if err != nil {
				d.s.Warnf("Problem encountered extracting URLs for: %v, %v - %v.", file.File, purl, err)
				problems = true // Record this as a warning
				continue
			}
			depOutput.Component = url.Component
			depOutput.Version = url.Version
			depOutput.URL = url.URL
			var licenses []dtos.DependencyLicense
			splitLicenses := strings.Split(url.LicenseID, "/") // Check to see if we have multiple licenses returned
			if len(splitLicenses) > 1 {
				for _, splitLicense := range splitLicenses {
					spl := strings.TrimSpace(splitLicense)
					d.s.Debugf("Searching for split license: %v", spl)
					lic, err := d.lic.GetLicenseByName(spl, false)
					if err != nil || len(lic.LicenseName) == 0 {
						if err != nil {
							d.s.Warnf("Problem encountered searching for license %v (%v): %v", spl, splitLicense, err)
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
	d.s.Debugf("Output dependencies: %v", depFileOutputs)
	if problems {
		d.s.Warnf("Encountered issues while processing dependencies: %v", request)
		return dtos.DependencyOutput{Files: depFileOutputs}, true, errors.New("encountered issues while processing dependencies")
	}

	return dtos.DependencyOutput{Files: depFileOutputs}, false, nil
}

// TreeMap is a wrapper around map to make it more similar to JavaScript Map
type TreeMap struct {
	data map[string]*dtos.Dependency
}

type DependencyData struct {
	Version  string `json:"dep_ver"`
	PurlName string `json:"dep_purl_name"`
}

func cleanVersion(version string) string {
	regex := regexp.MustCompile(`^[\^~>=<]+(.*)`)
	return regex.ReplaceAllString(version, "$1")
}

// GetDependencies takes the Dependency Input request, searches for component details and returns a Dependency Output struct.
func (d DependencyUseCase) GetTransientDependencies(purls []string, versions []string, visitedNodes map[string]bool, tree map[string]*dtos.Dependency, depth int) (dtos.DependencyOutput, bool, error) {
	fmt.Printf("Depth: %d", depth)
	var depFileOutputs []dtos.DependencyFileOutput
	if len(purls) <= 0 || depth <= 1 {
		return dtos.DependencyOutput{Files: depFileOutputs}, false, nil
	}

	for i, p := range purls {
		visited := fmt.Sprintf("%s@%s", p, cleanVersion(versions[i]))
		//fmt.Printf("VISITED %v", visited)
		visitedNodes[visited] = true
		if tree[visited] == nil {
			newDep := &dtos.Dependency{
				Purl:     p,
				Version:  cleanVersion(versions[i]),
				Children: []*dtos.Dependency{},
			}
			tree[visited] = newDep
		}
	}

	results, _ := d.depModel.GetTransientDependencies(purls, versions)
	var newPurls []string
	if len(results) > 0 {
		for _, r := range results {
			if r.Purl == "" {
				continue
			}
			purl_v := fmt.Sprintf("%s@%s", r.Purl, cleanVersion(r.Version))

			visitedNodes[purl_v] = true
			if tree[purl_v] == nil {
				newDep := &dtos.Dependency{
					Purl:     r.Purl,
					Version:  cleanVersion(r.Version),
					Children: []*dtos.Dependency{},
				}
				tree[purl_v] = newDep

			}

			childDependencies := []DependencyData{}
			if err := json.Unmarshal(r.Data, &childDependencies); err != nil {
				fmt.Printf("failed to unmarshal dependencies: %v", err)
			}

			addedPurls := make(map[string]bool)
			for _, child := range childDependencies {
				newPurl := fmt.Sprintf("%s@%s", child.PurlName, cleanVersion(child.Version))
				newPurls = append(newPurls, newPurl)
				purl_v_child := fmt.Sprintf("%s@%s", child.PurlName, cleanVersion(child.Version))

				if addedPurls[purl_v_child] {
					continue
				}

				if tree[purl_v_child] != nil {
					if tree[purl_v] != nil {
						tree[purl_v].Children = append(tree[purl_v].Children, tree[purl_v_child])
					}
				} else {

					newDep := &dtos.Dependency{
						Purl:     child.PurlName,
						Version:  cleanVersion(child.Version),
						Children: []*dtos.Dependency{},
					}

					tree[purl_v_child] = newDep
					if tree[purl_v] != nil {
						tree[purl_v].Children = append(tree[purl_v].Children, newDep)
						addedPurls[purl_v_child] = true
					}

				}

				/*	if addedPurls[purl_v_child] {
							fmt.Printf("Apending child dep %v to %v\n", purl_v_child, purl_v)
							tree[purl_v].Children = append(tree[purl_v_child].Children, newDep)
							addedPurls[purl_v_child] = true
						}
					} else {
						fmt.Printf("Purl NOT added: %v", purl_v_child)
						//var purl_version = fmt.Sprintf("%s@%s", child.PurlName, cleanVersion(child.Version))
						if !addedPurls[purl_v_child] {
							tree[purl_v].Children = append(tree[purl_v_child].Children, tree[purl_v_child])
							addedPurls[purl_v_child] = true
						}

					}*/
			}
		}

		var filteredPurls []string
		for _, p := range newPurls {
			if _, visited := visitedNodes[p]; !visited {
				filteredPurls = append(filteredPurls, p)
			}
		}

		var p []string
		var v []string
		for _, purl := range filteredPurls {
			parts := strings.Split(purl, "@")
			p = append(p, parts[0])
			v = append(v, parts[1])
		}
		d.GetTransientDependencies(p, v, visitedNodes, tree, depth-1)

	}

	return dtos.DependencyOutput{Files: depFileOutputs}, false, nil
}
