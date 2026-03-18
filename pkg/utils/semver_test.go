// SPDX-License-Identifier: GPL-2.0-or-later
/*
 * Copyright (C) 2018-2023 SCANOSS.COM
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

import "testing"

func TestVersionMatchesRequirement(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		requirement string
		want        bool
	}{
		{name: "exact match", version: "1.0.0", requirement: "1.0.0", want: true},
		{name: "greater than satisfied", version: "2.0.0", requirement: ">1.0.0", want: true},
		{name: "greater than not satisfied", version: "0.5.0", requirement: ">1.0.0", want: false},
		{name: "less than satisfied", version: "0.5.0", requirement: "<1.0.0", want: true},
		{name: "less than not satisfied", version: "2.0.0", requirement: "<1.0.0", want: false},
		{name: "gte satisfied", version: "1.0.0", requirement: ">=1.0.0", want: true},
		{name: "lte satisfied", version: "1.0.0", requirement: "<=1.0.0", want: true},
		{name: "not equal satisfied", version: "2.0.0", requirement: "!=1.0.0", want: true},
		{name: "not equal not satisfied", version: "1.0.0", requirement: "!=1.0.0", want: false},
		{name: "caret range satisfied", version: "1.2.3", requirement: "^1.0.0", want: true},
		{name: "caret range not satisfied", version: "2.0.0", requirement: "^1.0.0", want: false},
		{name: "tilde range satisfied", version: "1.0.5", requirement: "~1.0.0", want: true},
		{name: "tilde range not satisfied", version: "1.1.0", requirement: "~1.0.0", want: false},
		{name: "range constraint", version: "1.5.0", requirement: ">=1.0.0, <2.0.0", want: true},
		{name: "invalid version", version: "not-a-version", requirement: ">1.0.0", want: false},
		{name: "invalid requirement", version: "1.0.0", requirement: "invalid", want: false},
		{name: "empty version", version: "", requirement: ">1.0.0", want: false},
		{name: "empty requirement", version: "1.0.0", requirement: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VersionMatchesRequirement(tt.version, tt.requirement); got != tt.want {
				t.Errorf("VersionMatchesRequirement(%q, %q) = %v, want %v", tt.version, tt.requirement, got, tt.want)
			}
		})
	}
}
