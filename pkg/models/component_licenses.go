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

// NewComponentLicensesModel creates the licence lookup over the serving database.
// golang may be nil, which disables the live pkg.go.dev lookup.
func NewComponentLicensesModel(ctx context.Context, s *zap.SugaredLogger, db *sqlx.DB,
	q *database.DBQueryContext, res *resolver.Resolver, golang *GolangModel,
) *ComponentLicensesModel {
	return &ComponentLicensesModel{ctx: ctx, s: s, db: db, q: q, res: res, golang: golang}
}

// GetURLsByComponent returns the licences of a resolved component. purlID is the
// component's purl_id as the resolver found it, empty when it did not resolve.
//
// The version is the one the resolver pinned. When it pinned none - the requested
// version is not published - the highest published version answers instead, as
// mining's "latest URL by date" did; the caller reports that as REQUIREMENT_NOT_MET.
//
// A golang module the knowledge base cannot answer for is looked up live on
// pkg.go.dev, then as the GitHub repository its path names, as mining did.
func (m *ComponentLicensesModel) GetURLsByComponent(component componentHelper.Component, purlID string) (AllURL, error) {
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
	return url, nil
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
