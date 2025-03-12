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

package dtos

import (
	"encoding/json"
	"errors"
	"fmt"
	transitiveDp "scanoss.com/dependencies/pkg/transitive_dependencies"

	"go.uber.org/zap"
)

type DependencyInput struct {
	Files []DependencyFileInput `json:"files"`
	Depth int                   `json:"depth"`
}

type DependencyFileInput struct {
	File  string         `json:"file,omitempty"`
	Purls []DepPurlInput `json:"purls"`
}

type DepPurlInput struct {
	Purl        string `json:"purl"`
	Requirement string `json:"requirement,omitempty"`
}

// ParseDependencyInput converts the input byte array to a DependencyInput structure.
func ParseDependencyInput(s *zap.SugaredLogger, input []byte) (DependencyInput, error) {
	if len(input) == 0 {
		return DependencyInput{}, errors.New("no input dependency data supplied to parse")
	}
	var data DependencyInput
	err := json.Unmarshal(input, &data)
	if err != nil {
		s.Errorf("Parse failure: %v", err)
		return DependencyInput{}, fmt.Errorf("failed to parse dependency input data: %v", err)
	}
	return data, nil
}

// ParseTransitiveDependencyInput converts the input byte array to a []string structure.
func ParseComponentsInput(s *zap.SugaredLogger, input []byte) ([]transitiveDp.Component, error) {
	if len(input) == 0 {
		return []transitiveDp.Component{}, errors.New("no input dependency data supplied to parse")
	}
	var data []DepPurlInput
	err := json.Unmarshal(input, &data)
	if err != nil {
		s.Errorf("Parse failure: %v", err)
		return []transitiveDp.Component{}, fmt.Errorf("failed to parse dependency input data: %v", err)
	}
	packageNames := []transitiveDp.Component{}
	for _, entry := range data {
		pName, pError := transitiveDp.ExtractPackageIdentifierFromPurl(entry.Purl)
		if pError != nil {
			s.Warnf("Failed to get package identifier  %s: %s", entry.Purl, err)
			continue
		}
		packageNames = append(packageNames, transitiveDp.Component{
			PackageName: pName,
			Version:     entry.Requirement,
		})
	}
	return packageNames, nil
}
