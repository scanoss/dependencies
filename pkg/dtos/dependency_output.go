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

type DependencyOutput struct {
	Files []DependencyFileOutput `json:"files"`
}

type DependencyFileOutput struct {
	File         string               `json:"file"`
	ID           string               `json:"id"`
	Status       string               `json:"status"`
	Dependencies []DependenciesOutput `json:"dependencies"`
}

type DependenciesOutput struct {
	Component string              `json:"component"`
	Purl      string              `json:"purl"`
	Version   string              `json:"version"`
	URL       string              `json:"url"`
	Comment   string              `json:"comment"`
	Licenses  []DependencyLicense `json:"licenses"`
}

type DependencyLicense struct {
	Name   string `json:"name"`
	SpdxID string `json:"spdx_id"`
	IsSpdx bool   `json:"is_spdx_approved"`
}

// ExportDependencyOutput converts the DependencyOutput structure to a byte array.
func ExportDependencyOutput(s *zap.SugaredLogger, output DependencyOutput) ([]byte, error) {
	data, err := json.Marshal(output)
	if err != nil {
		s.Errorf("Parse failure: %v", err)
		return nil, errors.New("failed to produce JSON from dependency output data")
	}
	return data, nil
}

// ParseDependencyOutput converts the input byte array to a DependencyOutput structure.
func ParseDependencyOutput(s *zap.SugaredLogger, input []byte) (DependencyOutput, error) {
	if len(input) == 0 {
		return DependencyOutput{}, errors.New("no output dependency data supplied to parse")
	}
	var data DependencyOutput
	err := json.Unmarshal(input, &data)
	if err != nil {
		s.Errorf("Parse failure: %v", err)
		return DependencyOutput{}, fmt.Errorf("failed to parse dependency output data: %v", err)
	}
	return data, nil
}
