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

package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/guseggert/pkggodev-client"
	"github.com/jmoiron/sqlx"
	"github.com/package-url/packageurl-go"
	"github.com/scanoss/go-grpc-helper/pkg/grpc/database"
	"github.com/scanoss/go-models-helper/pkg/helpers"
	"github.com/scanoss/go-models-helper/pkg/models"
	purlutils "github.com/scanoss/go-purl-helper/pkg"
	"go.uber.org/zap"
	myconfig "scanoss.com/dependencies/pkg/config"
)

type GolangProjects struct {
	ctx     context.Context
	s       *zap.SugaredLogger
	conn    *sqlx.Conn
	config  *myconfig.ServerConfig
	q       *database.DBQueryContext
	ver     *VersionModel
	lic     *LicenseModel
	mine    *MineModel
	project *ProjectModel // TODO Do we add golang component to the projects table?
}

// NewGolangProjectModel creates a new instance of Golang Project Model.
func NewGolangProjectModel(ctx context.Context, s *zap.SugaredLogger, db *sqlx.DB, conn *sqlx.Conn, config *myconfig.ServerConfig) *GolangProjects {
	return &GolangProjects{ctx: ctx, s: s, conn: conn, config: config,
		q:   database.NewDBSelectContext(s, db, conn, config.Database.Trace),
		ver: NewVersionModel(ctx, s, conn), lic: NewLicenseModel(ctx, s, conn), mine: NewMineModel(ctx, s, conn),
		project: NewProjectModel(ctx, s, conn),
	}
}

// GetGoLangURLByPurlString searches the Golang Projects for the specified Purl (and requirement).
func (m *GolangProjects) GetGoLangURLByPurlString(purlString, purlReq string) (AllURL, error) {
	if len(purlString) == 0 {
		m.s.Error("Please specify a valid Purl String to query")
		return AllURL{}, errors.New("please specify a valid Purl String to query")
	}
	purl, err := purlutils.PurlFromString(purlString)
	if err != nil {
		return AllURL{}, err
	}
	purlName, err := purlutils.PurlNameFromString(purlString)
	if err != nil {
		return AllURL{}, err
	}
	if len(purl.Version) == 0 && len(purlReq) > 0 { // No version specified, but we might have a specific version in the Requirement
		ver := purlutils.GetVersionFromReq(purlReq)
		if len(ver) > 0 {
			purl.Version = ver
			purlReq = ""
		}
	}
	return m.GetGoLangURLByPurl(purl, purlName, purlReq)
}

// GetGoLangURLByPurl searches the Golang Projects for the specified Purl Package (and optional requirement).
func (m *GolangProjects) GetGoLangURLByPurl(purl packageurl.PackageURL, purlName, purlReq string) (AllURL, error) {
	if len(purl.Version) > 0 {
		return m.GetGolangUrlsByPurlNameTypeVersion(purlName, purl.Type, purl.Version)
	}
	return m.GetGolangUrlsByPurlNameType(purlName, purl.Type, purlReq)
}

// GetGolangUrlsByPurlNameType searches Golang Project for the specified Purl by Purl Type (and optional requirement).
func (m *GolangProjects) GetGolangUrlsByPurlNameType(purlName, purlType, purlReq string) (AllURL, error) {
	if len(purlName) == 0 {
		m.s.Error("Please specify a valid Purl Name to query")
		return AllURL{}, errors.New("please specify a valid Purl Name to query")
	}
	if len(purlType) == 0 {
		m.s.Errorf("Please specify a valid Purl Type to query: %v", purlName)
		return AllURL{}, errors.New("please specify a valid Purl Type to query")
	}
	query := "SELECT component, v.version_name AS version, v.semver AS semver," +
		" l.license_name AS license, l.spdx_id AS license_id, l.is_spdx AS is_spdx," +
		" purl_name, mine_id FROM golang_projects u" +
		" LEFT JOIN mines m ON u.mine_id = m.id" +
		" LEFT JOIN licenses l ON u.license_id = l.id" +
		" LEFT JOIN versions v ON u.version_id = v.id" +
		" WHERE m.purl_type = $1 AND u.purl_name = $2 AND is_indexed = True" +
		" ORDER BY version_date DESC"
	var allURLs []AllURL
	err := m.q.SelectContext(m.ctx, &allURLs, query, purlType, purlName)
	if err != nil {
		m.s.Errorf("Failed to query golang projects table for %v - %v: %v", purlType, purlName, err)
		return AllURL{}, fmt.Errorf("failed to query the golang projects table: %v", err)
	}
	m.s.Debugf("Found %v results for %v, %v.", len(allURLs), purlType, purlName)
	if len(allURLs) == 0 { // Check pkg.go.dev for the latest data
		m.s.Debugf("Checking PkgGoDev for live info...")
		allURL, err := m.getLatestPkgGoDev(purlName, purlType, "")
		if err == nil {
			m.s.Debugf("Retrieved golang data from pkg.go.dev: %#v", allURL)
			allURLs = append(allURLs, allURL)
		} else {
			m.s.Infof("Ran into an issue looking up pkg.go.dev for: %v. Ignoring", purlName)
		}
	}

	// Convert to helpers.AllURL slice and call shared helper
	helperURLs := convertToHelperAllURLs(allURLs)
	var projRepo helpers.ProjectRepository
	if m.project != nil {
		projRepo = m.project
	}
	result, err := helpers.PickOneURL(m.s, projRepo, helperURLs, purlName, purlType, purlReq)
	if err != nil {
		return AllURL{}, err
	}
	// Convert back to local AllURL
	return convertFromHelperAllURL(result), nil
}

// GetGolangUrlsByPurlNameTypeVersion searches Golang Projects for specified Purl, Type and Version.
func (m *GolangProjects) GetGolangUrlsByPurlNameTypeVersion(purlName, purlType, purlVersion string) (AllURL, error) {
	if len(purlName) == 0 {
		m.s.Error("Please specify a valid Purl Name to query")
		return AllURL{}, errors.New("please specify a valid Purl Name to query")
	}
	if len(purlType) == 0 {
		m.s.Error("Please specify a valid Purl Type to query")
		return AllURL{}, errors.New("please specify a valid Purl Type to query")
	}
	if len(purlVersion) == 0 {
		m.s.Error("Please specify a valid Purl Version to query")
		return AllURL{}, errors.New("please specify a valid Purl Version to query")
	}
	query := "SELECT component, v.version_name AS version, v.semver AS semver," +
		" l.license_name AS license, l.spdx_id AS license_id, l.is_spdx AS is_spdx," +
		" purl_name, mine_id FROM golang_projects u" +
		" LEFT JOIN mines m ON u.mine_id = m.id" +
		" LEFT JOIN licenses l ON u.license_id = l.id" +
		" LEFT JOIN versions v ON u.version_id = v.id" +
		" WHERE m.purl_type = $1 AND u.purl_name = $2 AND v.version_name = $3 AND is_indexed = True" +
		" ORDER BY version_date DESC"
	var allURLs []AllURL
	err := m.q.SelectContext(m.ctx, &allURLs, query, purlType, purlName, purlVersion)
	if err != nil {
		m.s.Errorf("Failed to query golang projects table for %v - %v: %v", purlType, purlName, err)
		return AllURL{}, fmt.Errorf("failed to query the golang projects table: %v", err)
	}
	m.s.Debugf("Found %v results for %v, %v.", len(allURLs), purlType, purlName)
	if len(allURLs) > 0 { // We found an entry. Let's check if it has license data
		helperURLs := convertToHelperAllURLs(allURLs)
		var projRepo helpers.ProjectRepository
		if m.project != nil {
			projRepo = m.project
		}
		helperResult, err2 := helpers.PickOneURL(m.s, projRepo, helperURLs, purlName, purlType, "")
		if err2 != nil {
			return AllURL{}, err2
		}
		allURL := convertFromHelperAllURL(helperResult)
		if len(allURL.License) == 0 { // No license data found. Need to search for live info
			m.s.Debugf("Couldn't find license data for component. Need to search live data")
			allURLs = allURLs[:0]
		} else {
			return allURL, nil // Return the component details
		}
	}
	if len(allURLs) == 0 { // Check pkg.go.dev for the latest data
		m.s.Debugf("Checking PkgGoDev for live info...")
		allURL, err := m.getLatestPkgGoDev(purlName, purlType, purlVersion)
		if err == nil {
			m.s.Debugf("Retrieved golang data from pkg.go.dev: %#v", allURL)
			allURLs = append(allURLs, allURL)
		} else {
			m.s.Infof("Ran into an issue looking up pkg.go.dev for: %v - %v. Ignoring", purlName, purlVersion)
		}
	}
	// Convert to helpers.AllURL slice and call shared helper
	helperURLs := convertToHelperAllURLs(allURLs)
	var projRepo helpers.ProjectRepository
	if m.project != nil {
		projRepo = m.project
	}
	result, err := helpers.PickOneURL(m.s, projRepo, helperURLs, purlName, purlType, "")
	if err != nil {
		return AllURL{}, err
	}
	// Convert back to local AllURL
	return convertFromHelperAllURL(result), nil
}

// savePkg writes the given package details to the Golang Projects table.
//
//goland:noinspection ALL
func (m *GolangProjects) savePkg(allURL AllURL, version Version, license License, comp *pkggodevclient.Package) error {
	if len(allURL.PurlName) == 0 {
		m.s.Error("Please specify a valid Purl to save")
		return errors.New("please specify a valid Purl to save")
	}
	if allURL.MineID <= 0 {
		m.s.Error("Please specify a valid mine id to save")
		return errors.New("please specify a valid mine id to save")
	}
	if version.ID <= 0 || len(version.VersionName) == 0 {
		m.s.Error("Please specify a valid version to save")
		return errors.New("please specify a valid version to save")
	}
	if license.ID <= 0 || len(license.LicenseName) == 0 {
		m.s.Error("Please specify a valid license to save")
		return errors.New("please specify a valid license to save")
	}
	if comp == nil {
		m.s.Error("Please specify a valid component package to save")
		return errors.New("please specify a valid component package to save")
	}
	m.s.Debugf("Attempting to save '%#v' - %#v to the golang_projects table...", allURL, version)
	// Search for an existing entry first
	var existingPurl string
	err := m.conn.QueryRowxContext(m.ctx,
		"SELECT purl_name FROM golang_projects"+
			" WHERE purl_name = $1 AND version = $2",
		allURL.PurlName, allURL.Version,
	).Scan(&existingPurl)
	if err != nil && err != sql.ErrNoRows {
		m.s.Warnf("Error: Problem encountered searching golang_projects table for %v: %v", allURL, err)
	}
	var purlName string
	sqlQueryType := "insert"
	if len(existingPurl) > 0 {
		// update entry
		sqlQueryType = "update"
		m.s.Debugf("Updating new Golang project: %#v", comp)
		//goland:noinspection ALL
		err = m.conn.QueryRowxContext(m.ctx,
			"UPDATE golang_projects SET component = $1, version = $2, version_id = $3, version_date = $4,"+
				" is_module = $5, is_package = $6, license = $7, license_id = $8, has_valid_go_mod_file = $9,"+
				" has_redistributable_license = $10, has_tagged_version = $11, has_stable_version = $12,"+
				" repository = $13, is_indexed = $14, purl_name = $15, mine_id = $16"+
				" WHERE purl_name = $17 AND version = $18"+
				" RETURNING purl_name",
			allURL.Component, allURL.Version, version.ID, comp.Published,
			comp.IsModule, comp.IsPackage, license.LicenseName, license.ID, comp.HasValidGoModFile,
			comp.HasRedistributableLicense, comp.HasTaggedVersion, comp.HasStableVersion,
			comp.Repository, true, allURL.PurlName, allURL.MineID,
			allURL.PurlName, allURL.Version,
		).Scan(&purlName)
	} else {
		m.s.Debugf("Inserting new Golang project: %#v", comp)
		// insert new entry
		err = m.conn.QueryRowxContext(m.ctx,
			"INSERT INTO golang_projects (component, version, version_id, version_date, is_module, is_package,"+
				" license, license_id, has_valid_go_mod_file, has_redistributable_license, has_tagged_version,"+
				" has_stable_version, repository, is_indexed, purl_name, mine_id, index_timestamp)"+
				" VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)"+
				" RETURNING purl_name",
			allURL.Component, allURL.Version, version.ID, comp.Published,
			comp.IsModule, comp.IsPackage, license.LicenseName, license.ID, comp.HasValidGoModFile,
			comp.HasRedistributableLicense, comp.HasTaggedVersion, comp.HasStableVersion,
			comp.Repository, true, allURL.PurlName, allURL.MineID, "",
		).Scan(&purlName)
	}
	if err != nil {
		m.s.Errorf("Error: Failed to %v new component into golang_projects table for %v - %#v: %v", sqlQueryType, allURL, comp, err)
		return fmt.Errorf("failed to %v new component into golang projects: %v", sqlQueryType, err)
	}
	m.s.Debugf("Completed %v of %v", sqlQueryType, purlName)
	return nil
}

// getLatestPkgGoDev retrieves the latest information about a Golang Package from https://pkg.go.dev
// If requested (via config), it will commit that data to the Golang Projects table.
func (m *GolangProjects) getLatestPkgGoDev(purlName, purlType, purlVersion string) (AllURL, error) {
	allURL, pkg, latest, err := m.queryPkgGoDev(purlName, purlVersion)
	if err != nil {
		return allURL, err
	}
	cleansedLicense, err := models.CleanseLicenseName(allURL.License)
	if err != nil {
		return allURL, err
	}
	license, _ := m.lic.GetLicenseByName(cleansedLicense, m.config.Components.CommitMissing)
	if len(license.LicenseName) == 0 {
		m.s.Warnf("No license details in DB for: %v", cleansedLicense)
	} else {
		allURL.License = license.LicenseName
		allURL.LicenseID = license.LicenseID
		allURL.IsSpdx = license.IsSpdx
	}
	version, _ := m.ver.GetVersionByName(allURL.Version, m.config.Components.CommitMissing)
	if len(version.VersionName) == 0 {
		m.s.Warnf("No version details in DB for: %v", allURL.Version)
	}
	mineIDs, _ := m.mine.GetMineIdsByPurlType(purlType)
	if len(mineIDs) > 0 {
		allURL.MineID = mineIDs[0] // Assign the first mine id
	} else {
		m.s.Warnf("No mine details in DB for purl type: %v", purlType)
	}
	// Package is not the "latest" version (i.e. queried with a version) and we've been requested to save it
	if !latest && m.config.Components.CommitMissing {
		_ = m.savePkg(allURL, version, license, pkg)
	}
	return allURL, nil
}

// queryPkgGoDev retrieves the latest information about a Golang Package from https://pkg.go.dev
func (m *GolangProjects) queryPkgGoDev(purlName, purlVersion string) (AllURL, *pkggodevclient.Package, bool, error) {
	if len(purlName) == 0 {
		m.s.Errorf("Please specify a valid Purl Name to query")
		return AllURL{}, nil, false, errors.New("please specify a valid Purl Name to query")
	}
	client := pkggodevclient.New()
	pkg := purlName
	if len(purlVersion) > 0 {
		pkg = fmt.Sprintf("%s@%s", purlName, purlVersion)
	}
	latest := false
	m.s.Debugf("Checking pkg.go.dev for the latest info: %v", pkg)
	comp, err := client.DescribePackage(pkggodevclient.DescribePackageRequest{Package: pkg})
	if err != nil && len(purlVersion) > 0 {
		// We have a version zero search, so look for the latest one
		m.s.Debugf("Failed to query pkg.go.dev for %v: %v. Trying without version...", pkg, err)
		comp, err = client.DescribePackage(pkggodevclient.DescribePackageRequest{Package: purlName})
		latest = true // Mark that this information is from the latest package and not a specific version
	}
	if err != nil {
		m.s.Warnf("Failed to query pkg.go.dev for %v: %v", pkg, err)
		return AllURL{}, nil, latest, fmt.Errorf("failed to query pkg.go.dev: %v", err)
	}
	var version = comp.Version
	if len(purlVersion) > 0 {
		version = purlVersion // Force the requested version if specified (the returned value can be concatenated)
	}
	allURL := models.AllURL{
		Component: purlName,
		Version:   version,
		License:   comp.License,
		PurlName:  purlName,
		URL:       fmt.Sprintf("https://%v", comp.Repository),
	}
	return allURL, comp, latest, nil
}
