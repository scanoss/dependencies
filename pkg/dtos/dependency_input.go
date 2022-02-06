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
	zlog "scanoss.com/dependencies/pkg/logger"
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

func ParseDependencyInput(input []byte) (DependencyInput, error) {
	if input == nil || len(input) == 0 {
		return DependencyInput{}, errors.New("no input dependency data supplied to parse")
	}
	var data DependencyInput
	err := json.Unmarshal(input, &data)
	if err != nil {
		zlog.S.Errorf("Parse failure: %v", err)
		return DependencyInput{}, errors.New(fmt.Sprintf("failed to parse dependency input data: %v", err))
	}
	zlog.S.Debugf("Parsed data2: %v", data)
	return data, nil
}
