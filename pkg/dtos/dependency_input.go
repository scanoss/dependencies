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

type TransitiveDependencyDTO struct {
	Depth     *int           `json:"depth,omitempty"`
	Ecosystem string         `json:"ecosystem"`
	Purls     []DepPurlInput `json:"purls"`
	Limit     *int           `json:"limit,omitempty"`
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

// ParseTransitiveReqDTOS converts the input byte array to a TransitiveDependencyDTO structure.
func ParseTransitiveReqDTOS(s *zap.SugaredLogger, input []byte) (TransitiveDependencyDTO, error) {
	if len(input) == 0 {
		return TransitiveDependencyDTO{}, errors.New("no input dependency data supplied to parse")
	}
	var data TransitiveDependencyDTO
	err := json.Unmarshal(input, &data)
	if err != nil {
		s.Errorf("Parse failure: %v", err)
		return TransitiveDependencyDTO{}, fmt.Errorf("failed to parse dependency input data: %v", err)
	}
	return data, nil
}
