// SPDX-License-Identifier: GPL-2.0-or-later
/*
 * Copyright (C) 2018-2025 SCANOSS.COM
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

package resolver

import (
	semver "github.com/Masterminds/semver/v3"
	compHelperUtils "github.com/scanoss/go-component-helper/componenthelper/utils"
)

// hasSemverOperator reports whether a requirement is a range rather than a plain
// version literal. Delegated so this service and the shared helper agree on what
// counts as an operator.
func hasSemverOperator(requirement string) bool {
	return compHelperUtils.HasSemverOperator(requirement)
}

// SelectVersion picks the highest version satisfying the constraint, or the
// highest version overall when the constraint is empty. It reports false when
// nothing qualifies.
//
// This is the port of go-models' pickOneUrl selection, minus the URL handling:
// a version that will not parse as semver is treated as v0.0.0 rather than
// dropped, so a component whose versions are all non-semver still resolves to
// one of them instead of failing outright. An unparseable constraint is ignored,
// which degrades to "highest version" - the same thing the mining path did.
func SelectVersion(versions []string, constraint string) (string, bool) {
	if len(versions) == 0 {
		return "", false
	}
	var constraints *semver.Constraints
	if len(constraint) > 0 {
		if parsed, err := semver.NewConstraint(constraint); err == nil {
			constraints = parsed
		}
	}

	zero, _ := semver.NewVersion("v0.0.0")
	var (
		best     *semver.Version
		bestName string
		found    bool
	)
	for _, version := range versions {
		if len(version) == 0 {
			continue
		}
		parsed, err := semver.NewVersion(version)
		if err != nil {
			parsed = zero
		}
		if constraints != nil && !constraints.Check(parsed) {
			continue
		}
		if !found || parsed.GreaterThan(best) {
			best, bestName, found = parsed, version, true
		}
	}
	return bestName, found
}
