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
	"github.com/jmoiron/sqlx"
	"scanoss.com/dependencies/pkg/dtos"
	zlog "scanoss.com/dependencies/pkg/logger"
	"scanoss.com/dependencies/pkg/models"
	"strings"
)

type DependencyUseCase struct {
	ctx     context.Context
	conn    *sqlx.Conn
	allUrls *models.AllUrlsModel
}

func NewDependencies(ctx context.Context, conn *sqlx.Conn) *DependencyUseCase {
	return &DependencyUseCase{ctx: ctx, conn: conn,
		allUrls: models.NewAllUrlModel(ctx, conn, models.NewProjectModel(ctx, conn)),
	}
}

func (d DependencyUseCase) GetDependencies(request dtos.DependencyInput) (dtos.DependencyOutput, error) {

	var depFileOutputs []dtos.DependencyFileOutput
	var problems = false
	for _, file := range request.Files {
		var fileOutput dtos.DependencyFileOutput
		fileOutput.File = file.File
		fileOutput.Id = "dependency"
		fileOutput.Status = "pending"
		var depOutputs []dtos.DependenciesOutput
		for _, purl := range file.Purls {
			if len(purl.Purl) == 0 {
				zlog.S.Infof("Empty Purl string supplied for: %v. Skipping", file.File)
				continue
			}
			var depOutput dtos.DependenciesOutput
			depOutput.Purl = strings.Split(purl.Purl, "@")[0] // Remove any version specif info from the PURL
			url, err := d.allUrls.GetUrlsByPurlString(purl.Purl, purl.Requirement)
			if err != nil {
				zlog.S.Errorf("Problem encountered extracting URLs for: %v - %v.", purl, err)
				problems = true
				continue
				// TODO add a placeholder in the response?
			}
			depOutput.Component = url.Component
			depOutput.Version = url.Version
			var licenses []dtos.DependencyLicense
			var license dtos.DependencyLicense
			license.Name = url.License
			license.SpdxId = url.LicenseId
			license.IsSpdx = url.IsSpdx
			licenses = append(licenses, license)
			depOutput.Licenses = licenses
			depOutputs = append(depOutputs, depOutput)
		}
		fileOutput.Dependencies = depOutputs
		depFileOutputs = append(depFileOutputs, fileOutput)
	}
	if problems {
		zlog.S.Errorf("Encountered issues while processing dependencies: %v", request)
		return dtos.DependencyOutput{}, errors.New("encountered issues while processing dependencies")
	}
	zlog.S.Debugf("Output dependencies: %v", depFileOutputs)

	return dtos.DependencyOutput{Files: depFileOutputs}, nil
}
