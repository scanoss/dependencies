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

package resolver

import (
	"sort"

	semver "github.com/Masterminds/semver/v3"
)

// NearestVersions orders `versions` by semver proximity to `target`, nearest
// first, excluding `target` itself and empty/duplicate entries. Unparseable
// versions are appended last (lowest priority). When `target` itself isn't a
// valid semver, the parseable versions are returned in ascending order followed
// by the unparseable ones. Used to drive the version-fallback search outward
// from the requested/resolved version. A port of scanoss.api's license.NearestVersions,
// with one difference: versions semver ranks equal - which it does whenever they
// differ only in build metadata, as the export's golang pseudo-versions
// (0.0.0+20230818062718-4ef9b55571c8) do - are ordered by that metadata instead of
// arbitrarily, so the nearest in time wins rather than whichever came first.
func NearestVersions(target string, versions []string) []string {
	type pv struct {
		orig string
		v    *semver.Version
	}
	var parsed []pv
	var unparsed []string
	seen := map[string]bool{}
	for _, ver := range versions {
		if ver == "" || ver == target || seen[ver] {
			continue
		}
		seen[ver] = true
		if sv, err := semver.NewVersion(ver); err == nil {
			parsed = append(parsed, pv{ver, sv})
		} else {
			unparsed = append(unparsed, ver)
		}
	}
	sort.SliceStable(parsed, func(i, j int) bool { return versionLess(parsed[i].v, parsed[j].v) })

	out := make([]string, 0, len(parsed)+len(unparsed))

	tv, err := semver.NewVersion(target)
	if err != nil {
		for _, p := range parsed {
			out = append(out, p.orig)
		}
		return append(out, unparsed...)
	}

	// Insertion index of target in the ascending list, then expand outward,
	// preferring the higher side on ties.
	idx := sort.Search(len(parsed), func(i int) bool { return !versionLess(parsed[i].v, tv) })
	lo, hi := idx-1, idx
	for lo >= 0 || hi < len(parsed) {
		switch {
		case lo < 0:
			out = append(out, parsed[hi].orig)
			hi++
		case hi >= len(parsed):
			out = append(out, parsed[lo].orig)
			lo--
		default:
			distLo := idx - lo
			distHi := hi - idx + 1
			if distHi <= distLo { // tie → prefer the higher version
				out = append(out, parsed[hi].orig)
				hi++
			} else {
				out = append(out, parsed[lo].orig)
				lo--
			}
		}
	}
	return append(out, unparsed...)
}

// versionLess orders by semver precedence, then by build metadata, which semver
// precedence ignores.
func versionLess(a, b *semver.Version) bool {
	if c := a.Compare(b); c != 0 {
		return c < 0
	}
	return a.Metadata() < b.Metadata()
}
