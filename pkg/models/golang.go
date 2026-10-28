// SPDX-License-Identifier: GPL-2.0-or-later
/*
 * Copyright (C) 2018-2026 SCANOSS.COM
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
	"net/http"
	"time"

	pkggodevclient "github.com/guseggert/pkggodev-client"
	purlutils "github.com/scanoss/go-purl-helper/pkg"
	"go.uber.org/zap"
)

// GolangModel looks a golang module up live on pkg.go.dev, for modules the
// knowledge base has no licence for.
//
// Mining kept the answers it got in a `golang_projects` table, written back when
// COMP_COMMIT_MISSING was set. Serving has no such table - golang modules are
// ordinary `component` rows - and is read-only, so the live answer is used for the
// request and never stored.
type GolangModel struct {
	ctx      context.Context
	s        *zap.SugaredLogger
	lic      *LicenseModel
	describe func(pkggodevclient.DescribePackageRequest) (*pkggodevclient.Package, error)
	timeout  time.Duration // zero means pkgGoDevTimeout
}

// pkgGoDevTimeout bounds the live lookup of one module, both of its attempts
// (with the version, then without) together. The export has golang modules at
// pseudo-versions only, so every tagged golang dependency goes live, and a slow
// pkg.go.dev - one call was seen taking 19 s - would otherwise hold the whole
// request. Past it, the lookup gives up and the GitHub repository and the
// knowledge base answer instead.
const pkgGoDevTimeout = 5 * time.Second

// errPkgGoDevTimeout is the live lookup running out of pkgGoDevTimeout.
var errPkgGoDevTimeout = errors.New("pkg.go.dev did not answer in time")

// NewGolangModel creates the live golang lookup. lic resolves the licence name
// pkg.go.dev reports against the serving licence catalogue.
func NewGolangModel(ctx context.Context, s *zap.SugaredLogger, lic *LicenseModel) *GolangModel {
	// The HTTP timeout ends a call the lookup has stopped waiting for, so an
	// abandoned one does not linger.
	client := pkggodevclient.New(pkggodevclient.WithHTTPClient(&http.Client{Timeout: pkgGoDevTimeout}))
	return &GolangModel{ctx: ctx, s: s, lic: lic, describe: client.DescribePackage, timeout: pkgGoDevTimeout}
}

// describeBy asks pkg.go.dev about a package, giving up at deadline. The client
// takes no context, so the call runs on its own and is abandoned past the
// deadline; the HTTP client's timeout then ends it.
func (m *GolangModel) describeBy(deadline time.Time, req pkggodevclient.DescribePackageRequest) (*pkggodevclient.Package, error) {
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return nil, errPkgGoDevTimeout
	}
	type answer struct {
		pkg *pkggodevclient.Package
		err error
	}
	answered := make(chan answer, 1)
	go func() {
		pkg, err := m.describe(req)
		answered <- answer{pkg, err}
	}()
	timer := time.NewTimer(remaining)
	defer timer.Stop()
	select {
	case a := <-answered:
		return a.pkg, a.err
	case <-timer.C:
		return nil, errPkgGoDevTimeout
	}
}

// GetGoLangURLByPurlString looks the module of a golang purl up on pkg.go.dev, at
// the purl's version, or the requirement's, or the latest.
func (m *GolangModel) GetGoLangURLByPurlString(purlString, purlReq string) (AllURL, error) {
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
	version := purl.Version
	if len(version) == 0 && len(purlReq) > 0 { // No version specified, but we might have a specific version in the Requirement
		version = purlutils.GetVersionFromReq(purlReq)
	}
	return m.getLatestPkgGoDev(purlName, version)
}

// getLatestPkgGoDev retrieves the information about a Golang Package from
// https://pkg.go.dev, and resolves its licence against the licence catalogue.
func (m *GolangModel) getLatestPkgGoDev(purlName, purlVersion string) (AllURL, error) {
	allURL, licenseName, err := m.queryPkgGoDev(purlName, purlVersion)
	if err != nil {
		return allURL, err
	}
	cleansedLicense, err := CleanseLicenseName(licenseName)
	if err != nil {
		return allURL, err
	}
	if len(cleansedLicense) == 0 {
		return allURL, nil
	}
	license, err := m.lic.GetLicenseByName(cleansedLicense)
	if err != nil {
		return allURL, err
	}
	if len(license.LicenseName) == 0 {
		// Reported as pkg.go.dev names it, with no identifier, as mining did.
		m.s.Warnf("No license details in DB for: %v", cleansedLicense)
		license = License{LicenseName: licenseName}
	}
	allURL.Licenses = []License{license}
	return allURL, nil
}

// queryPkgGoDev retrieves the information about a Golang Package from
// https://pkg.go.dev, falling back to the latest version when the requested one
// is not there. It returns the licence name pkg.go.dev reports.
func (m *GolangModel) queryPkgGoDev(purlName, purlVersion string) (AllURL, string, error) {
	if len(purlName) == 0 {
		m.s.Errorf("Please specify a valid Purl Name to query")
		return AllURL{}, "", errors.New("please specify a valid Purl Name to query")
	}
	pkg := purlName
	if len(purlVersion) > 0 {
		pkg = fmt.Sprintf("%s@%s", purlName, purlVersion)
	}
	m.s.Debugf("Checking pkg.go.dev for the latest info: %v", pkg)
	timeout := m.timeout
	if timeout <= 0 {
		timeout = pkgGoDevTimeout
	}
	deadline := time.Now().Add(timeout)
	comp, err := m.describeBy(deadline, pkggodevclient.DescribePackageRequest{Package: pkg})
	// We have a version zero search, so look for the latest one - unless pkg.go.dev
	// has already used up the time: then the fallbacks answer.
	if err != nil && len(purlVersion) > 0 && !errors.Is(err, errPkgGoDevTimeout) {
		m.s.Debugf("Failed to query pkg.go.dev for %v: %v. Trying without version...", pkg, err)
		comp, err = m.describeBy(deadline, pkggodevclient.DescribePackageRequest{Package: purlName})
	}
	if err != nil {
		m.s.Warnf("Failed to query pkg.go.dev for %v: %v", pkg, err)
		return AllURL{}, "", fmt.Errorf("failed to query pkg.go.dev: %v", err)
	}
	var version = comp.Version
	if len(purlVersion) > 0 {
		version = purlVersion // Force the requested version if specified (the returned value can be concatenated)
	}
	allURL := AllURL{
		Component: purlName,
		Version:   version,
		PurlName:  purlName,
		URL:       fmt.Sprintf("https://%v", comp.Repository),
	}
	return allURL, comp.License, nil
}
