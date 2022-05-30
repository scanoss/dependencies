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

package models

import (
	"context"
	"errors"
	"fmt"
	pkggodevclient "github.com/guseggert/pkggodev-client"
	"github.com/jmoiron/sqlx"
	"github.com/package-url/packageurl-go"
	"regexp"
	zlog "scanoss.com/dependencies/pkg/logger"
	"scanoss.com/dependencies/pkg/utils"
)

type GolangProjects struct {
	ctx  context.Context
	conn *sqlx.Conn
	ver  *versionModel
	lic  *licenseModel
}

var vRegex = regexp.MustCompile(`^v\d+\.\d+\.\d+-\d+-\w+$`) // regex to check for commit based version

func NewGolangProjectModel(ctx context.Context, conn *sqlx.Conn) *GolangProjects {
	return &GolangProjects{ctx: ctx, conn: conn, ver: NewVersionModel(ctx, conn), lic: NewLicenseModel(ctx, conn)}
}

func (m *GolangProjects) GetGoLangUrlByPurlString(purlString, purlReq string) (AllUrl, error) {
	if len(purlString) == 0 {
		zlog.S.Errorf("Please specify a valid Purl String to query")
		return AllUrl{}, errors.New("please specify a valid Purl String to query")
	}
	purl, err := utils.PurlFromString(purlString)
	if err != nil {
		return AllUrl{}, err
	}
	purlName, err := utils.PurlNameFromString(purlString)
	if err != nil {
		return AllUrl{}, err
	}
	if len(purl.Version) == 0 && len(purlReq) > 0 { // No version specified, but we might have a specific version in the Requirement
		ver := utils.GetVersionFromReq(purlReq)
		if len(ver) > 0 {
			purl.Version = ver
			purlReq = ""
		}
	}
	return m.GetGoLangUrlByPurl(purl, purlName, purlReq)
}

func (m *GolangProjects) GetGoLangUrlByPurl(purl packageurl.PackageURL, purlName, purlReq string) (AllUrl, error) {
	if len(purl.Version) > 0 {
		return m.GetGolangUrlsByPurlNameTypeVersion(purlName, purl.Type, purl.Version)
	}
	return m.GetGolangUrlsByPurlNameType(purlName, purl.Type, purlReq)
}

func (m *GolangProjects) GetGolangUrlsByPurlNameType(purlName, purlType, purlReq string) (AllUrl, error) {
	if len(purlName) == 0 {
		zlog.S.Errorf("Please specify a valid Purl Name to query")
		return AllUrl{}, errors.New("please specify a valid Purl Name to query")
	}
	if len(purlType) == 0 {
		zlog.S.Errorf("Please specify a valid Purl Type to query: %v", purlName)
		return AllUrl{}, errors.New("please specify a valid Purl Type to query")
	}
	var golangUrls []AllUrl
	err := m.conn.SelectContext(m.ctx, &golangUrls,
		"SELECT component, v.version_name AS version, v.semver AS semver,"+
			" l.license_name AS license, l.spdx_id AS license_id, l.is_spdx AS is_spdx,"+
			" purl_name, mine_id FROM golang_projects u"+
			" LEFT JOIN mines m ON u.mine_id = m.id"+
			" LEFT JOIN licenses l ON u.license_id = l.id"+
			" LEFT JOIN versions v ON u.version_id = v.id"+
			" WHERE m.purl_type = $1 AND u.purl_name = $2 AND is_indexed = True"+
			" ORDER BY version_date DESC",
		purlType, purlName)
	if err != nil {
		zlog.S.Errorf("Failed to query golang projects table for %v - %v: %v", purlType, purlName, err)
		return AllUrl{}, fmt.Errorf("failed to query the golang projects table: %v", err)
	}
	zlog.S.Debugf("Found %v results for %v, %v.", len(golangUrls), purlType, purlName)
	// Pick the most appropriate version to return
	return pickOneUrl(nil, golangUrls, purlName, purlType, purlReq)
}

func (m *GolangProjects) GetGolangUrlsByPurlNameTypeVersion(purlName, purlType, purlVersion string) (AllUrl, error) {
	if len(purlName) == 0 {
		zlog.S.Errorf("Please specify a valid Purl Name to query")
		return AllUrl{}, errors.New("please specify a valid Purl Name to query")
	}
	if len(purlType) == 0 {
		zlog.S.Errorf("Please specify a valid Purl Type to query")
		return AllUrl{}, errors.New("please specify a valid Purl Type to query")
	}
	if len(purlVersion) == 0 {
		zlog.S.Errorf("Please specify a valid Purl Version to query")
		return AllUrl{}, errors.New("please specify a valid Purl Version to query")
	}
	var allUrls []AllUrl
	err := m.conn.SelectContext(m.ctx, &allUrls,
		"SELECT component, v.version_name AS version, v.semver AS semver,"+
			" l.license_name AS license, l.spdx_id AS license_id, l.is_spdx AS is_spdx,"+
			" purl_name, mine_id FROM golang_projects u"+
			" LEFT JOIN mines m ON u.mine_id = m.id"+
			" LEFT JOIN licenses l ON u.license_id = l.id"+
			" LEFT JOIN versions v ON u.version_id = v.id"+
			" WHERE m.purl_type = $1 AND u.purl_name = $2 AND v.version_name = $3 AND is_indexed = True"+
			" ORDER BY version_date DESC",
		purlType, purlName, purlVersion)
	if err != nil {
		zlog.S.Errorf("Failed to query golang projects table for %v - %v: %v", purlType, purlName, err)
		return AllUrl{}, fmt.Errorf("failed to query the golang projects table: %v", err)
	}
	zlog.S.Debugf("Found %v results for %v, %v.", len(allUrls), purlType, purlName)
	if len(allUrls) == 0 { // Check pkg.go.dev for the latest data
		allUrl, err := m.getLatestPkgGoDev(purlName, purlVersion)
		if err == nil {
			zlog.S.Debugf("Retrieved golang data from pkg.go.dev: %#v", allUrl)
			allUrls = append(allUrls, allUrl)
		} else {
			zlog.S.Infof("Ran into an issue looking up pkg.go.dev for: %v - %v. Ignoring", purlName, purlVersion)
		}
	}
	// Pick the most appropriate version to return
	return pickOneUrl(nil, allUrls, purlName, purlType, "")
}

func (m *GolangProjects) getLatestPkgGoDev(purlName, purlVersion string) (AllUrl, error) {

	allUrl, err := m.queryPkgGoDev(purlName, purlVersion)
	if err != nil {
		return allUrl, err
	}
	cleansedLicense, err := CleanseLicenseName(allUrl.License)
	if err != nil {
		return allUrl, err
	}
	license, err := m.lic.GetLicenseByName(cleansedLicense, false)
	if err != nil {
		return allUrl, err
	}
	if len(license.LicenseName) == 0 {
		zlog.S.Warnf("No license details in DB for: %v", cleansedLicense)
	} else {
		allUrl.License = license.LicenseName
		allUrl.LicenseId = license.LicenseId
		allUrl.IsSpdx = license.IsSpdx
	}
	version, err := m.ver.GetVersionByName(allUrl.Version, false)
	if err != nil {
		return allUrl, err
	}
	if len(version.VersionName) == 0 {
		zlog.S.Warnf("No version details in DB for: %v", allUrl.Version)
	}
	// TODO add to golang_projects table
	return allUrl, nil
}

func (m *GolangProjects) queryPkgGoDev(purlName, purlVersion string) (AllUrl, error) {
	if len(purlName) == 0 {
		zlog.S.Errorf("Please specify a valid Purl Name to query")
		return AllUrl{}, errors.New("please specify a valid Purl Name to query")
	}
	client := pkggodevclient.New()
	pkg := purlName
	if len(purlVersion) > 0 {
		pkg = fmt.Sprintf("%s@%s", purlName, purlVersion)
	}
	zlog.S.Debugf("Checking pkg.go.dev for the latest info: %v", pkg)
	d, err := client.DescribePackage(pkggodevclient.DescribePackageRequest{Package: pkg})
	if err != nil && len(purlVersion) > 0 && vRegex.MatchString(purlVersion) {
		// We have a version zero search, so look for the latest one
		zlog.S.Debugf("Failed to query pkg.go.dev for %v: %v. Trying without version...", pkg, err)
		d, err = client.DescribePackage(pkggodevclient.DescribePackageRequest{Package: purlName})
		if err == nil { // Return the details for the latest version
			allUrl := AllUrl{
				Component: purlName,
				Version:   purlVersion, // Force the version to be what was requested
				License:   d.License,
				PurlName:  purlName,
				Url:       fmt.Sprintf("https://%v", d.Repository),
			}
			return allUrl, nil
		}
	}
	if err != nil {
		zlog.S.Warnf("Failed to query pkg.go.dev for %v: %v", pkg, err)
		return AllUrl{}, fmt.Errorf("failed to query pkg.go.dev: %v", err)
	}
	allUrl := AllUrl{
		Component: purlName,
		Version:   d.Version,
		License:   d.License,
		PurlName:  purlName,
		Url:       fmt.Sprintf("https://%v", d.Repository),
	}
	return allUrl, nil
}
