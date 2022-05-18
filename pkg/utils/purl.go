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
	"errors"
	"fmt"
	"github.com/package-url/packageurl-go"
	"regexp"
	zlog "scanoss.com/dependencies/pkg/logger"
	"strings"
)

var pkgRegex = regexp.MustCompile(`^pkg:(?P<type>\w+)/(?P<name>.+)$`) // regex to parse purl name from purl string
var typeRegex = regexp.MustCompile(`^(npm|nuget)$`)                   // regex to parse purl types that should not be lower cased
var vRegex = regexp.MustCompile(`^(=|==|)(?P<name>\w+\S+)$`)          // regex to parse purl name from purl string

// PurlFromString takes an input Purl string and returns a decomposed structure of all the elements
func PurlFromString(purlString string) (packageurl.PackageURL, error) {
	if len(purlString) == 0 {
		return packageurl.PackageURL{}, errors.New("no Purl string specified to parse")
	}
	purl, err := packageurl.FromString(purlString)
	if err != nil {
		return packageurl.PackageURL{}, err
	}
	return purl, nil
}

// PurlNameFromString take an input Purl string and returns the Purl Name only
func PurlNameFromString(purlString string) (string, error) {
	if len(purlString) == 0 {
		return "", fmt.Errorf("no purl string supplied to parse")
	}
	matches := pkgRegex.FindStringSubmatch(purlString)
	if matches != nil && len(matches) > 0 {
		ti := pkgRegex.SubexpIndex("type")
		ni := pkgRegex.SubexpIndex("name")
		if ni >= 0 {
			// Remove any version@/subpath?/qualifiers# info from the PURL
			pn := strings.Split(strings.Split(strings.Split(matches[ni], "@")[0], "?")[0], "#")[0]
			// Lowercase the purl name if it's not on the exclusion list (defined in the regex)
			if ti >= 0 && !typeRegex.MatchString(matches[ti]) {
				pn = strings.ToLower(pn)
			}
			return pn, nil
		}
	}
	return "", fmt.Errorf("no purl name found in '%v'", purlString)
}

// ConvertPurlString takes an input PURL and checks to see if anything needs to be modified before search the KB
func ConvertPurlString(purlString string) string { // TODO remove now that we have golang?
	// Replace Golang GitHub package reference with just GitHub
	if len(purlString) > 0 && strings.HasPrefix(purlString, "pkg:golang/github.com/") {
		s := strings.Replace(purlString, "pkg:golang/github.com/", "pkg:github/", -1)
		p := strings.Split(s, "/")
		if len(p) >= 3 {
			return fmt.Sprintf("%s/%s/%s", p[0], p[1], p[2]) // Only return the GitHub part of the url
		}
		return s
	}
	return purlString
}

func GetVersionFromReq(purlReq string) string {
	matches := vRegex.FindStringSubmatch(purlReq)
	if matches != nil && len(matches) > 0 {
		ni := vRegex.SubexpIndex("name")
		if ni >= 0 {
			zlog.S.Debugf("Changing requirement %v to Version %v", purlReq, matches[ni])
			return matches[ni]
		}
	}
	return ""
}

// ProjectUrl returns a browsable URL for the given purl type and name
func ProjectUrl(purlName, purlType string) (string, error) {
	if len(purlName) == 0 {
		return "", fmt.Errorf("no purl name supplied")
	}
	if len(purlType) == 0 {
		return "", fmt.Errorf("no purl type supplied")
	}
	switch purlType {
	case "github":
		return fmt.Sprintf("https://github.com/%v", purlName), nil
	case "npm":
		return fmt.Sprintf("https://www.npmjs.com/package/%v", purlName), nil
	case "maven":
		return fmt.Sprintf("https://mvnrepository.com/artifact/%v", purlName), nil
	case "gem":
		return fmt.Sprintf("https://rubygems.org/gems/%v", purlName), nil
	case "pypi":
		return fmt.Sprintf("https://pypi.org/project/%v", purlName), nil
	case "golang":
		return fmt.Sprintf("https://pkg.go.dev/%v", purlName), nil
	}
	return "", fmt.Errorf("no url prefix found for '%v': %v", purlType, purlName)
}
