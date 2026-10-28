// SPDX-License-Identifier: GPL-2.0-or-later
/*
 * Copyright (C) 2025 SCANOSS.COM
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
	"slices"
	"strings"

	purlhelper "github.com/scanoss/go-purl-helper/pkg"
)

// candidateKeys lists the spellings of a purl's versionless `component.purl` key
// to look up, most specific first.
//
// The export does not store one canonical form. Scoped npm packages are kept
// percent-encoded (pkg:npm/%40scope/name), generic, conda and cran components keep
// their qualifiers in the key (pkg:generic/x?repository_url=...), and some rpm and
// deb names carry a raw '+' where the canonical form writes %2B. Parsing the purl
// and rebuilding it from its decoded parts matches none of those, so the key is
// tried as the caller wrote it, then in its canonical rendering, then decoded.
func candidateKeys(purl string, id Identity) []string {
	keys := make([]string, 0, 5)
	add := func(k string) {
		if len(k) > 0 && !slices.Contains(keys, k) {
			keys = append(keys, k)
		}
	}
	raw, rawQualifiers := rawKey(purl)
	if len(rawQualifiers) > 0 {
		add(raw + "?" + rawQualifiers)
	}
	add(raw)
	if parsed, err := purlhelper.PurlFromString(purl); err == nil {
		parsed.Version, parsed.Subpath = "", ""
		add(parsed.ToString())
		parsed.Qualifiers = nil
		add(parsed.ToString())
	}
	add(servingPurlKey(id))
	return keys
}

// rawKey strips the subpath and the version from a purl string without decoding
// anything, and returns the qualifiers separately. The version is whatever follows
// an '@' in the last path segment: a scoped npm namespace (@scope) is a segment of
// its own, so its '@' is never mistaken for a version separator.
func rawKey(purl string) (string, string) {
	key, qualifiers := strings.TrimSpace(purl), ""
	if i := strings.IndexByte(key, '#'); i >= 0 {
		key = key[:i]
	}
	if i := strings.IndexByte(key, '?'); i >= 0 {
		key, qualifiers = key[:i], key[i+1:]
	}
	if slash := strings.LastIndexByte(key, '/'); slash >= 0 {
		if at := strings.IndexByte(key[slash:], '@'); at >= 0 {
			key = key[:slash+at]
		}
	}
	return key, qualifiers
}

// pickKey returns the matched component for the most specific candidate key.
func pickKey(keys []string, found []component) (component, bool) {
	for _, k := range keys {
		for _, c := range found {
			if c.Purl == k {
				return c, true
			}
		}
	}
	return component{}, false
}
