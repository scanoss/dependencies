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
)

func TestDependencyInput(t *testing.T) {

	//var inputJson1 = `{
	// "vue-dev/packages/weex-template-compiler/package.json": {
	//   "purls": [
	//     {
	//       "purl": "pkg:npm/acorn",
	//       "requirement": "^5.2.1"
	//     },
	//     {
	//       "purl": "pkg:npm/escodegen",
	//       "requirement": "^1.8.1"
	//     },
	//     {
	//       "purl": "pkg:npm/he",
	//       "requirement": "^1.1.0"
	//     }
	//   ]
	// }
	//}
	//`
	//
	//data, err := ParseDependencyInput1(inputJson1)
	//if err != nil {
	//	t.Errorf("dtos.ParseDependencyInput() error = %v", err)
	//}
	//fmt.Printf("Parsed input data1: %v\n", data)

	var inputJson2 = `{
  "files": [
    {
      "file": "vue-dev/packages/weex-template-compiler/package.json",
      "purls": [
        {
          "purl": "pkg:npm/acorn",
          "requirement": "^5.2.1"
        },
        {
          "purl": "pkg:npm/escodegen",
          "requirement": "^1.8.1"
        },
        {
          "purl": "pkg:npm/he",
          "requirement": "^1.1.0"
        }
      ]
    }
  ],
  "depth": 2
}
`
	jsonText := []byte(inputJson2)
	data2, err := ParseDependencyInput(jsonText)
	if err != nil {
		t.Errorf("dtos.ParseDependencyInput() error = %v", err)
	}
	fmt.Println("Parsed input data2: ", data2)
}