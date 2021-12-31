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

package utils

import (
	"github.com/package-url/packageurl-go"
	"reflect"
	"testing"
)

// Help with test details can be found here: https://go.dev/doc/code

func TestPurlFromString(t *testing.T) {

	w, _ := packageurl.FromString("pkg:maven/io.prestosql/presto-main@v1.0")
	tests := []struct {
		name    string
		input   string
		want    packageurl.PackageURL
		wantErr bool
	}{
		{
			name:  "Purl from String",
			input: "pkg:maven/io.prestosql/presto-main@v1.0",
			want:  w,
		},
		{
			name:    "Empty String",
			input:   "",
			want:    w,
			wantErr: true,
		},
		{
			name:    "Rubbish String",
			input:   "rubbish.string",
			want:    w,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PurlFromString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("utils.PurlFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			//t.Logf("Got: %v: '%v' '%v' '%v' '%v' '%v' '%v'", got, got.Type, got.Namespace, got.Name, got.Version, got.Qualifiers, got.Subpath)
			//t.Logf("Exp: %v: '%v' '%v' '%v' '%v' '%v' '%v'", tt.want, tt.want.Type, tt.want.Namespace, tt.want.Name, tt.want.Version, tt.want.Qualifiers, tt.want.Subpath)
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("utils.PurlFromString() = %v, want %v", got, tt.want)
			}
		})
	}

}
