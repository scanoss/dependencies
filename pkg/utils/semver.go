// SPDX-License-Identifier: GPL-2.0-or-later
/*
 * Copyright (C) 2026 SCANOSS.COM
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

// Package utils provides shared utility functions for the dependencies service.
package utils

import (
	"github.com/Masterminds/semver/v3"
)

// VersionMatchesRequirement checks if a version satisfies a semver constraint/range.
func VersionMatchesRequirement(version, requirement string) bool {
	v, err := semver.NewVersion(version)
	if err != nil {
		return false
	}
	constraint, err := semver.NewConstraint(requirement)
	if err != nil {
		return false
	}
	return constraint.Check(v)
}
