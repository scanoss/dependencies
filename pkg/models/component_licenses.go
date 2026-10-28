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
	"strings"

	"github.com/jmoiron/sqlx"
	componentHelper "github.com/scanoss/go-component-helper/componenthelper"
	"github.com/scanoss/go-grpc-helper/pkg/grpc/database"
	purlutils "github.com/scanoss/go-purl-helper/pkg"
	"go.uber.org/zap"
	"scanoss.com/dependencies/pkg/resolver"
)

// ComponentLicensesModel reads the declared licences of a resolved component.
//
// It replaces the mining AllUrlsModel, which went `all_urls ⋈ mines ⋈ versions ⋈
// licenses`, fell back to `projects` for a component-level licence, and for golang
// to `golang_projects` and pkg.go.dev. Serving keys licences by (purl_id, version)
// in `component_version_license`, golang components are ordinary `component` rows,
// and there is no component-level licence table any more.
type ComponentLicensesModel struct {
	ctx    context.Context
	s      *zap.SugaredLogger
	db     *sqlx.DB
	q      *database.DBQueryContext
	res    *resolver.Resolver
	golang *GolangModel
}

// AllURL is the licence lookup's answer for one component: the version it was
// answered for, the project URL, and the release's licences. The name is kept from
// the mining model, whose rows were all_urls entries.
type AllURL struct {
	Component string
	Version   string
	PurlName  string
	URL       string
	Licenses  []License
	// FallbackMessage is set when the licences are not the release's own but were
	// borrowed from its source component or from another version (see
	// licenseFallback). The caller reports it with a WARNING info code.
	FallbackMessage string
}

// releaseLicensesQuery reads a release's declared licences.
//
// Mining kept one licence per all_urls row, so the service reported one; a serving
// release can carry several rows (about 1% of releases carry two). All of them are
// returned, in licence id order, for a stable answer. The SPDX identifier is not
// unique to a licence, and a compound one ("MIT/Apache-2.0") is split by the caller.
const releaseLicensesQuery = `SELECT COALESCE(l.long_name, '') AS license_name,
	COALESCE(l."SPDX", '') AS spdx_id,
	CASE WHEN sld."SPDX_id" IS NOT NULL THEN 1 ELSE 0 END AS is_spdx
FROM component_version_license cvl
JOIN license l ON l.id = cvl.license_id
LEFT JOIN "SPDX_license_data" sld ON sld."SPDX_id" = l."SPDX"
WHERE cvl.purl_id = ? AND cvl.version = ?
ORDER BY l.id`

// placeholderLicenseNames are the licence names that record the absence of a
// licence rather than a licence, when they carry no SPDX identifier. They are the
// four such rows of the serving `license` table: 523 (empty, which 43% of
// component_version_license points at - mining's "no licence" row), "None
// detected", "UNKNOWN" and "unknown". Keep this in step with componentLicensesQuery.
var placeholderLicenseNames = map[string]bool{"": true, "none detected": true, "unknown": true}

// isPlaceholder reports whether a licence row says only that there is no licence.
func isPlaceholder(l License) bool {
	return len(l.LicenseID) == 0 && placeholderLicenseNames[strings.ToLower(strings.TrimSpace(l.LicenseName))]
}

// hasLicence reports whether any of a release's licence rows is an actual licence.
func hasLicence(licenses []License) bool {
	for _, l := range licenses {
		if !isPlaceholder(l) {
			return true
		}
	}
	return false
}

// componentLicensesQuery reads every licensed release of a component, for the
// nearest-version fallback. It reads by purl_id, which the per-release table
// prunes well (10-40 ms), and drops the placeholder rows (see isPlaceholder): a
// borrowed "None detected" would say nothing the release did not.
const componentLicensesQuery = `SELECT cvl.version AS version,
	COALESCE(l.long_name, '') AS license_name,
	COALESCE(l."SPDX", '') AS spdx_id,
	CASE WHEN sld."SPDX_id" IS NOT NULL THEN 1 ELSE 0 END AS is_spdx
FROM component_version_license cvl
JOIN license l ON l.id = cvl.license_id
LEFT JOIN "SPDX_license_data" sld ON sld."SPDX_id" = l."SPDX"
WHERE cvl.purl_id = ? AND cvl.version <> ''
	AND NOT (COALESCE(l."SPDX", '') = '' AND lower(trim(COALESCE(l.long_name, ''))) IN ('', 'none detected', 'unknown'))
ORDER BY cvl.version, l.id`

// versionLicense is one licence row of componentLicensesQuery.
type versionLicense struct {
	Version string `db:"version"`
	License
}

// The info messages of the licence fallbacks, worded as scanoss.api words them.
func sourceFallbackMessage(sourcePurl string) string {
	return fmt.Sprintf("Showing results from the source code purl (%s)", sourcePurl)
}

func versionFallbackMessage(version string) string {
	return fmt.Sprintf("Showing results from version %s", version)
}

func sourceVersionFallbackMessage(sourcePurl, version string) string {
	return fmt.Sprintf("Showing results from the source code purl (%s) version %s", sourcePurl, version)
}

// NewComponentLicensesModel creates the licence lookup over the serving database.
// golang may be nil, which disables the live pkg.go.dev lookup.
func NewComponentLicensesModel(ctx context.Context, s *zap.SugaredLogger, db *sqlx.DB,
	q *database.DBQueryContext, res *resolver.Resolver, golang *GolangModel,
) *ComponentLicensesModel {
	return &ComponentLicensesModel{ctx: ctx, s: s, db: db, q: q, res: res, golang: golang}
}

// GetURLsByComponent returns the licences of a resolved component. key is what
// the resolver found for it: its purl_id (empty when it did not resolve) and its
// source component.
//
// The version is the one the resolver pinned. When it pinned none - the requested
// version is not published - the highest published version answers instead, as
// mining's "latest URL by date" did; the caller reports that as REQUIREMENT_NOT_MET.
//
// A golang module the knowledge base cannot answer for is looked up live on
// pkg.go.dev, then as the GitHub repository its path names, as mining did. A
// release that still has no licence borrows one (see licenseFallback).
func (m *ComponentLicensesModel) GetURLsByComponent(component componentHelper.Component, key resolver.ComponentKey) (AllURL, error) {
	purlID := key.PurlID
	purlName := component.Name
	url := AllURL{Component: component.Name, PurlName: purlName}
	url.URL, _ = purlutils.ProjectUrl(purlName, component.PurlType)
	isGolang := component.PurlType == "golang"
	// The export carries golang modules at pseudo-versions only, not at their tags,
	// so a requested golang version is usually not there. Mining answered those from
	// golang_projects; here they go to the live and GitHub lookups first, before
	// settling for the module's latest pseudo-version.
	if isGolang && len(component.Version) == 0 && (len(purlID) == 0 || len(component.Requirement) > 0) {
		if found, ok, err := m.getURLsByGolangPurl(component); err != nil || ok {
			return found, err
		}
	}
	if len(purlID) > 0 {
		url.Version = component.Version
		if len(url.Version) == 0 {
			url.Version, _ = resolver.SelectVersion(component.Versions, "")
		}
		if len(url.Version) > 0 {
			licenses, err := m.releaseLicenses(purlID, url.Version)
			if err != nil {
				return AllURL{}, err
			}
			url.Licenses = licenses
		}
	}
	if len(url.Licenses) == 0 && isGolang && len(component.Version) > 0 {
		if found, ok, err := m.getURLsByGolangPurl(component); err != nil || ok {
			return found, err
		}
	}
	// A release whose only rows are placeholders ("None detected") also borrows; if
	// nothing can be borrowed it keeps them, as mining reported them.
	if !hasLicence(url.Licenses) && len(purlID) > 0 {
		target := url.Version
		if len(target) == 0 {
			target = purlutils.GetVersionFromReq(component.Requirement)
		}
		licenses, message, err := m.licenseFallback(purlID, key.SourcePurl, target)
		if err != nil {
			return AllURL{}, err
		}
		if len(licenses) > 0 {
			url.Licenses, url.FallbackMessage = licenses, message
		}
	}
	return url, nil
}

// licenseFallback finds a licence for a release that has none of its own, in the
// order scanoss.api's licence lookup uses:
//
//  1. the source component (component.source_purl_id) at the same version;
//  2. the nearest version of the component itself that has a licence;
//  3. the nearest version of the source component that has a licence.
//
// The licence is reported against the requested release, so the message names
// where it came from. A licence can change between versions: that is why the
// borrowed one is flagged rather than passed off as the release's own. Returns no
// licences, and no message, when none of the three has one.
func (m *ComponentLicensesModel) licenseFallback(purlID string, source *resolver.SourcePurl, version string) ([]License, string, error) {
	if source != nil && len(version) > 0 && m.res != nil {
		resolved, err := m.res.Resolve(m.ctx, source.Purl, version)
		if err != nil && !errors.Is(err, resolver.ErrComponentNotFound) && !errors.Is(err, resolver.ErrInvalidPurl) {
			return nil, "", err
		}
		if err == nil && resolved.StatusCode == resolver.StatusSuccess && len(resolved.Version) > 0 {
			licenses, lerr := m.releaseLicenses(resolved.PurlID, resolved.Version)
			if lerr != nil {
				return nil, "", lerr
			}
			if hasLicence(licenses) {
				return licenses, sourceFallbackMessage(source.Purl), nil
			}
		}
	}
	nearest, licenses, err := m.nearestLicensedVersion(purlID, version)
	if err != nil || len(licenses) > 0 {
		return licenses, versionFallbackMessage(nearest), err
	}
	if source != nil && len(source.PurlID) > 0 {
		nearest, licenses, err = m.nearestLicensedVersion(source.PurlID, version)
		if err != nil || len(licenses) > 0 {
			return licenses, sourceVersionFallbackMessage(source.Purl, nearest), err
		}
	}
	return nil, "", nil
}

// nearestLicensedVersion returns the licensed release of a component nearest to
// version (resolver.NearestVersions' order: by semver distance, the higher side on
// ties, unparseable versions last) and its licences. One query reads every
// licensed release, rather than one per candidate version.
func (m *ComponentLicensesModel) nearestLicensedVersion(purlID, version string) (string, []License, error) {
	var rows []versionLicense
	err := m.q.SelectContext(m.ctx, &rows, m.db.Rebind(componentLicensesQuery), purlID)
	if err != nil {
		m.s.Errorf("Failed to query the licensed versions of %v: %v", purlID, err)
		return "", nil, fmt.Errorf("failed to query the component licences: %v", err)
	}
	byVersion := map[string][]License{}
	versions := make([]string, 0, len(rows))
	for _, r := range rows {
		if _, seen := byVersion[r.Version]; !seen {
			versions = append(versions, r.Version)
		}
		byVersion[r.Version] = append(byVersion[r.Version], r.License)
	}
	if len(versions) == 0 {
		return "", nil, nil
	}
	// With no version to be near to, the latest licensed release answers, as a
	// bare purl resolves to the latest release.
	if len(version) == 0 {
		latest, _ := resolver.SelectVersion(versions, "")
		return latest, byVersion[latest], nil
	}
	candidates := resolver.NearestVersions(version, versions)
	if len(candidates) == 0 {
		return "", nil, nil
	}
	return candidates[0], byVersion[candidates[0]], nil
}

// releaseLicenses reads the declared licences of one release.
func (m *ComponentLicensesModel) releaseLicenses(purlID, version string) ([]License, error) {
	var licenses []License
	err := m.q.SelectContext(m.ctx, &licenses, m.db.Rebind(releaseLicensesQuery), purlID, version)
	if err != nil {
		m.s.Errorf("Failed to query the licences of %v@%v: %v", purlID, version, err)
		return nil, fmt.Errorf("failed to query the component licences: %v", err)
	}
	// A licence with neither a name nor an identifier says nothing.
	kept := licenses[:0]
	for _, l := range licenses {
		if len(l.LicenseName) > 0 || len(l.LicenseID) > 0 {
			kept = append(kept, l)
		}
	}
	return kept, nil
}

// getURLsByGolangPurl answers a golang module the knowledge base has no licence
// for: first live from pkg.go.dev (skipped for github.com module paths, whose
// repository is the better source), then from the GitHub repository the module
// path names. ok is false when neither has anything.
func (m *ComponentLicensesModel) getURLsByGolangPurl(component componentHelper.Component) (AllURL, bool, error) {
	if m.golang != nil && !strings.HasPrefix(component.Purl, "pkg:golang/github.com/") {
		result, err := m.golang.GetGoLangURLByPurlString(component.Purl, component.Requirement)
		if err == nil && len(result.Licenses) > 0 {
			return result, true, nil
		}
	}
	m.s.Debugf("Falling back to Github URL %v, %v", component.Purl, component.Requirement)
	githubPurl := purlutils.ConvertGoPurlStringToGithub(component.Purl)
	if githubPurl == component.Purl || !strings.HasPrefix(githubPurl, "pkg:github/") {
		return AllURL{}, false, nil
	}
	version := component.Version
	if len(version) == 0 {
		version = purlutils.GetVersionFromReq(component.Requirement)
	}
	resolved, err := m.res.Resolve(m.ctx, githubPurl, version)
	if errors.Is(err, resolver.ErrComponentNotFound) || errors.Is(err, resolver.ErrInvalidPurl) {
		return AllURL{}, false, nil
	}
	if err != nil {
		return AllURL{}, false, err
	}
	if resolved.StatusCode != resolver.StatusSuccess || len(resolved.Version) == 0 {
		return AllURL{}, false, nil
	}
	licenses, err := m.releaseLicenses(resolved.PurlID, resolved.Version)
	if err != nil || len(licenses) == 0 {
		return AllURL{}, false, err
	}
	url := AllURL{Component: resolved.PurlName, Version: resolved.Version, PurlName: resolved.PurlName, Licenses: licenses}
	url.URL, _ = purlutils.ProjectUrl(resolved.PurlName, resolved.PurlType)
	return url, true, nil
}
