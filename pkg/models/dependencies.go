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

// Handle the declared dependencies of a release.

package models

import (
	"context"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/package-url/packageurl-go"
	"go.uber.org/zap"
	"scanoss.com/dependencies/pkg/resolver"
	"scanoss.com/dependencies/pkg/shared"
)

type DependencyModel struct {
	ctx context.Context
	s   *zap.SugaredLogger
	db  *sqlx.DB
	res *resolver.Resolver
}

// UnresolvedDependency is one declared dependency: the package identifier
// (namespace/name for a namespaced package, as the request takes it) and the
// requirement as the release declares it.
type UnresolvedDependency struct {
	Purl        string `json:"dep_purl_name"`
	Requirement string `json:"dep_ver"`
}

// NewDependencyModel create a new instance of the Dependency Model.
func NewDependencyModel(ctx context.Context, s *zap.SugaredLogger, db *sqlx.DB) *DependencyModel {
	return &DependencyModel{ctx: ctx, s: s, db: db, res: resolver.New(s, db)}
}

// dependencyRow is one row of component_version_dependency.
type dependencyRow struct {
	DepPurl    string `db:"dep_purl"`
	DepVersion string `db:"dep_version"`
}

// dependenciesQuery reads the declared dependencies of a release.
//
// Mining kept them per ecosystem, as a JSON array of {dep_purl_name, dep_ver} on
// <ecosystem>_dependencies keyed by (purl_name, version). Serving has one table,
// keyed by (purl_id, version), with a row per dependency carrying its full purl.
// Every scope is returned (dep, dev, peer, test, ...), as the mining arrays held
// them all. dep_resolved_version is not used: the walk resolves a requirement to
// the first version it names, as it always has.
const dependenciesQuery = `SELECT dep_purl, COALESCE(dep_version, '') AS dep_version
FROM component_version_dependency
WHERE purl_id = ? AND version = ?
ORDER BY dep_purl, dep_version`

// GetDependencies returns the declared dependencies of a release, given its
// package identifier (namespace/name, or name), version and ecosystem (the purl
// type). An unknown package or release has none.
func (m *DependencyModel) GetDependencies(purl string, version string, ecosystem string) ([]UnresolvedDependency, error) {
	// Check if ecosystem is supported
	// (already verified in the constructor but there is no harm to check it again)
	if _, isEcosystemSupported := shared.RegisteredEcosystems[ecosystem]; !isEcosystemSupported {
		return nil, errors.New("ecosystem not supported")
	}
	purlID, err := m.res.ComponentID(m.ctx, "pkg:"+servingPurlType(ecosystem)+"/"+purl)
	if errors.Is(err, resolver.ErrComponentNotFound) || errors.Is(err, resolver.ErrInvalidPurl) {
		return []UnresolvedDependency{}, nil
	}
	if err != nil {
		m.s.Errorf("Error: Failed to resolve %v/%v: %v", ecosystem, purl, err)
		return nil, err
	}
	rows, err := m.releaseDependencies(purlID, version)
	if err != nil {
		m.s.Errorf("Error: Failed to query the dependencies of %v/%v@%v: %v", ecosystem, purl, version, err)
		return nil, err
	}
	dependencies := make([]UnresolvedDependency, 0, len(rows))
	for _, row := range rows {
		name, nameErr := packageIdentifier(row.DepPurl)
		if nameErr != nil {
			m.s.Debugf("Skipping dependency %q of %v@%v: %v", row.DepPurl, purl, version, nameErr)
			continue
		}
		dependencies = append(dependencies, UnresolvedDependency{Purl: name, Requirement: row.DepVersion})
	}
	return dependencies, nil
}

// servingPurlType maps an ecosystem onto the purl type the export files it under.
// Mining called Rust crates "crates"; the export uses the purl spec's "cargo", for
// the components and for the dependency purls alike.
func servingPurlType(ecosystem string) string {
	if ecosystem == "crates" {
		return "cargo"
	}
	return ecosystem
}

// releaseDependencies reads the dependency rows of a release, accepting the other
// spelling of a leading "v" when the version as given has none: the walk derives
// versions from requirements, which do not carry the prefix a release may publish.
func (m *DependencyModel) releaseDependencies(purlID, version string) ([]dependencyRow, error) {
	var rows []dependencyRow
	if err := m.db.SelectContext(m.ctx, &rows, m.db.Rebind(dependenciesQuery), purlID, version); err != nil {
		return nil, fmt.Errorf("failed to query component_version_dependency: %w", err)
	}
	if len(rows) > 0 || len(version) == 0 {
		return rows, nil
	}
	toggled := "v" + version
	if version[0] == 'v' {
		toggled = version[1:]
	}
	if err := m.db.SelectContext(m.ctx, &rows, m.db.Rebind(dependenciesQuery), purlID, toggled); err != nil {
		return nil, fmt.Errorf("failed to query component_version_dependency: %w", err)
	}
	return rows, nil
}

// packageIdentifier turns a dependency purl into the identifier the walk carries:
// namespace/name when the purl has a namespace (maven group, npm scope, composer
// vendor), the bare name otherwise - the same form mining's dep_purl_name held.
func packageIdentifier(purl string) (string, error) {
	p, err := packageurl.FromString(purl)
	if err != nil {
		return "", err
	}
	if len(p.Name) == 0 {
		return "", errors.New("purl has no name")
	}
	if len(p.Namespace) > 0 {
		return p.Namespace + "/" + p.Name, nil
	}
	return p.Name, nil
}
