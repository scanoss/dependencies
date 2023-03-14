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
	"fmt"
	"testing"

	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
)

func TestDependencyOutput(t *testing.T) {
	var outputJson = `{
  "files": [
    {
      "file": "audit-workbench-master/package.json",
      "id": "dependency",
      "status": "pending",
      "dependencies": [
        {
          "component": "abort-controller",
          "purl": "abort-controller",
          "version": "",
          "licenses": [
            {
              "name": "MIT",
              "spdx_id": "MIT",
              "is_spdx_approved": true
            }
          ]
        },
        {
          "component": "chart.js",
          "purl": "chart.js",
          "version": "",
          "licenses": [
            {
              "name": "MIT",
              "spdx_id": "MIT",
              "is_spdx_approved": true
            }
          ]
        }
      ]
    }
  ]
}
`
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer zlog.SyncZap()
	output, err := ParseDependencyOutput(zlog.S, []byte(outputJson))
	if err != nil {
		t.Errorf("dtos.ParseDependencyInput() error = %v", err)
	}
	fmt.Println("Parsed output object: ", output)

	data, err := ExportDependencyOutput(zlog.S, output)
	if err != nil {
		t.Errorf("dtos.ParseDependencyInput() error = %v", err)
	}
	fmt.Println("Exported output data: ", data)

	_, err = ParseDependencyOutput(zlog.S, nil)
	if err == nil {
		t.Errorf("dtos.ParseDependencyOutput() did not fail")
	}
	fmt.Println("get expected error: ", err)

	var badJson = `{ "files": [ `
	_, err = ParseDependencyOutput(zlog.S, []byte(badJson))
	if err == nil {
		t.Errorf("dtos.ParseDependencyOutput() did not fail")
	}
	fmt.Println("get expected error: ", err)
}
