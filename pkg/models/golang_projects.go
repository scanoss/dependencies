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
	"database/sql"
	"errors"
	"fmt"
	pkggodevclient "github.com/guseggert/pkggodev-client"
	"github.com/jmoiron/sqlx"
	"github.com/package-url/packageurl-go"
	"regexp"
	myconfig "scanoss.com/dependencies/pkg/config"
	zlog "scanoss.com/dependencies/pkg/logger"
	"scanoss.com/dependencies/pkg/utils"
)

type GolangProjects struct {
	ctx    context.Context
	conn   *sqlx.Conn
	config *myconfig.ServerConfig
	ver    *versionModel
	lic    *licenseModel
	mine   *mineModel
}

var vRegex = regexp.MustCompile(`^v\d+\.\d+\.\d+-\d+-\w+$`) // regex to check for commit based version

// NewGolangProjectModel creates a new instance of Golang Project Model
func NewGolangProjectModel(ctx context.Context, conn *sqlx.Conn, config *myconfig.ServerConfig) *GolangProjects {
	return &GolangProjects{ctx: ctx, conn: conn, config: config,
		ver: NewVersionModel(ctx, conn), lic: NewLicenseModel(ctx, conn), mine: NewMineModel(ctx, conn),
	}
}

// GetGoLangUrlByPurlString searches the Golang Projects for the specified Purl (and requirement)
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

// GetGoLangUrlByPurl searches the Golang Projects for the specified Purl Package (and optional requirement)
func (m *GolangProjects) GetGoLangUrlByPurl(purl packageurl.PackageURL, purlName, purlReq string) (AllUrl, error) {
	if len(purl.Version) > 0 {
		return m.GetGolangUrlsByPurlNameTypeVersion(purlName, purl.Type, purl.Version)
	}
	return m.GetGolangUrlsByPurlNameType(purlName, purl.Type, purlReq)
}

// GetGolangUrlsByPurlNameType searches Golang Project for the specified Purl by Purl Type (and optional requirement)
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

// GetGolangUrlsByPurlNameTypeVersion searches Golang Projects for specified Purl, Type and Version
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
		allUrl, err := m.getLatestPkgGoDev(purlName, purlType, purlVersion)
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

// savePkg writes the given package details to the Golang Projects table
func (m *GolangProjects) savePkg(allUrl AllUrl, version Version, license License, comp *pkggodevclient.Package) error {
	if len(allUrl.PurlName) == 0 {
		zlog.S.Error("Please specify a valid Purl to save")
		return errors.New("please specify a valid Purl to save")
	}
	if allUrl.MineId <= 0 {
		zlog.S.Error("Please specify a valid mine id to save")
		return errors.New("please specify a valid mine id to save")
	}
	if version.Id <= 0 || len(version.VersionName) == 0 {
		zlog.S.Error("Please specify a valid version to save")
		return errors.New("please specify a valid version to save")
	}
	if license.Id <= 0 || len(license.LicenseName) == 0 {
		zlog.S.Error("Please specify a valid license to save")
		return errors.New("please specify a valid license to save")
	}
	if comp == nil {
		zlog.S.Error("Please specify a valid component package to save")
		return errors.New("please specify a valid component package to save")
	}
	zlog.S.Debugf("Attempting to save '%#v' - %#v to the golang_projects table...", allUrl, version)
	// Search for an existing entry first
	var existingPurl string
	err := m.conn.QueryRowxContext(m.ctx,
		"SELECT purl_name FROM golang_projects"+
			" WHERE purl_name = $1 AND version = $2",
		allUrl.PurlName, allUrl.Version,
	).Scan(&existingPurl)
	if err != nil && err != sql.ErrNoRows {
		zlog.S.Warnf("Error: Problem encountered searching golang_projects table for %v: %v", allUrl, err)
	}
	var purlName string
	sqlQueryType := "insert"
	if len(existingPurl) > 0 {
		// update entry
		sqlQueryType = "update"
		zlog.S.Debugf("Updating new Golang project: %#v", comp)
		err = m.conn.QueryRowxContext(m.ctx,
			"UPDATE golang_projects SET component = $1, version = $2, version_id = $3, version_date = $4,"+
				" is_module = $5, is_package = $6, license = $7, license_id = $8, has_valid_go_mod_file = $9,"+
				" has_redistributable_license = $10, has_tagged_version = $11, has_stable_version = $12,"+
				" repository = $13, is_indexed = $14, purl_name = $15, mine_id = $16"+
				" WHERE purl_name = $17 AND version = $18"+
				" RETURNING purl_name",
			allUrl.Component, allUrl.Version, version.Id, comp.Published,
			comp.IsModule, comp.IsPackage, license.LicenseName, license.Id, comp.HasValidGoModFile,
			comp.HasRedistributableLicense, comp.HasTaggedVersion, comp.HasStableVersion,
			comp.Repository, true, allUrl.PurlName, allUrl.MineId,
			allUrl.PurlName, allUrl.Version,
		).Scan(&purlName)
	} else {
		zlog.S.Debugf("Inserting new Golang project: %#v", comp)
		// insert new entry
		err = m.conn.QueryRowxContext(m.ctx,
			"INSERT INTO golang_projects (component, version, version_id, version_date, is_module, is_package,"+
				" license, license_id, has_valid_go_mod_file, has_redistributable_license, has_tagged_version,"+
				" has_stable_version, repository, is_indexed, purl_name, mine_id, index_timestamp)"+
				" VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)"+
				" RETURNING purl_name",
			allUrl.Component, allUrl.Version, version.Id, comp.Published,
			comp.IsModule, comp.IsPackage, license.LicenseName, license.Id, comp.HasValidGoModFile,
			comp.HasRedistributableLicense, comp.HasTaggedVersion, comp.HasStableVersion,
			comp.Repository, true, allUrl.PurlName, allUrl.MineId, "",
		).Scan(&purlName)
	}
	if err != nil {
		zlog.S.Errorf("Error: Failed to %v new component into golang_projects table for %v - %#v: %v", sqlQueryType, allUrl, comp, err)
		return fmt.Errorf("failed to %v new component into golang projects: %v", sqlQueryType, err)
	}
	zlog.S.Debugf("Completed %v of %v", sqlQueryType, purlName)
	return nil
}

// getLatestPkgGoDev retrieves the latest information about a Golang Package from https://pkg.go.dev
// If requested (via config), it will commit that data to the Golang Projects table
func (m *GolangProjects) getLatestPkgGoDev(purlName, purlType, purlVersion string) (AllUrl, error) {

	allUrl, pkg, latest, err := m.queryPkgGoDev(purlName, purlVersion)
	if err != nil {
		return allUrl, err
	}
	cleansedLicense, err := CleanseLicenseName(allUrl.License)
	if err != nil {
		return allUrl, err
	}
	license, _ := m.lic.GetLicenseByName(cleansedLicense, m.config.Components.CommitMissing)
	if len(license.LicenseName) == 0 {
		zlog.S.Warnf("No license details in DB for: %v", cleansedLicense)
	} else {
		allUrl.License = license.LicenseName
		allUrl.LicenseId = license.LicenseId
		allUrl.IsSpdx = license.IsSpdx
	}
	version, _ := m.ver.GetVersionByName(allUrl.Version, m.config.Components.CommitMissing)
	if len(version.VersionName) == 0 {
		zlog.S.Warnf("No version details in DB for: %v", allUrl.Version)
	}
	mineIds, _ := m.mine.GetMineIdsByPurlType(purlType)
	if mineIds != nil && len(mineIds) > 0 {
		allUrl.MineId = mineIds[0] // Assign the first mine id
	} else {
		zlog.S.Warnf("No mine details in DB for purl type: %v", purlType)
	}
	// Package is not the "latest" version (i.e. queried with a version) and we've been requested to save it
	if !latest && m.config.Components.CommitMissing {
		_ = m.savePkg(allUrl, version, license, pkg)
	}
	return allUrl, nil
}

// queryPkgGoDev retrieves the latest information about a Golang Package from https://pkg.go.dev
func (m *GolangProjects) queryPkgGoDev(purlName, purlVersion string) (AllUrl, *pkggodevclient.Package, bool, error) {
	if len(purlName) == 0 {
		zlog.S.Errorf("Please specify a valid Purl Name to query")
		return AllUrl{}, nil, false, errors.New("please specify a valid Purl Name to query")
	}
	client := pkggodevclient.New()
	pkg := purlName
	if len(purlVersion) > 0 {
		pkg = fmt.Sprintf("%s@%s", purlName, purlVersion)
	}
	latest := false
	zlog.S.Debugf("Checking pkg.go.dev for the latest info: %v", pkg)
	comp, err := client.DescribePackage(pkggodevclient.DescribePackageRequest{Package: pkg})
	if err != nil && len(purlVersion) > 0 && vRegex.MatchString(purlVersion) {
		// We have a version zero search, so look for the latest one
		zlog.S.Debugf("Failed to query pkg.go.dev for %v: %v. Trying without version...", pkg, err)
		comp, err = client.DescribePackage(pkggodevclient.DescribePackageRequest{Package: purlName})
		latest = true // Mark that this information is from the latest package and not a specific version
	}
	if err != nil {
		zlog.S.Warnf("Failed to query pkg.go.dev for %v: %v", pkg, err)
		return AllUrl{}, nil, latest, fmt.Errorf("failed to query pkg.go.dev: %v", err)
	}
	var version = comp.Version
	if len(purlVersion) > 0 {
		version = purlVersion // Force the requested version if specified (the returned value can be concatenated)
	}
	allUrl := AllUrl{
		Component: purlName,
		Version:   version,
		License:   comp.License,
		PurlName:  purlName,
		Url:       fmt.Sprintf("https://%v", comp.Repository),
	}
	return allUrl, comp, latest, nil
}
