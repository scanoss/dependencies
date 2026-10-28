// SPDX-License-Identifier: GPL-2.0-or-later
/*
 * Copyright (C) 2018-2025 SCANOSS.COM
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

// Package resolver turns a (purl, requirement) pair into a resolved component
// identity from the serving schema.
//
// It replaces the mining-backed resolution the service used to get from
// go-component-helper, which reached for `all_urls`, `mines` and `versions` -
// tables the serving schema does not have. Serving keys components by the UUID
// `component.purl_id`, with the version set in `component_version` and the
// linked source repository in `component.source_purl_id`.
//
// Ported from scanoss/scanoss.api pkg/resolver. Kept local for now; once the
// shape settles it belongs in go-models so the other legacy services can share it.
package resolver

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	semver "github.com/Masterminds/semver/v3"
	"github.com/jmoiron/sqlx"
	purlhelper "github.com/scanoss/go-purl-helper/pkg"
	"go.uber.org/zap"
)

// Resolution status codes, reported on Resolved.StatusCode.
const (
	StatusSuccess           = "SUCCESS"
	StatusComponentNotFound = "COMPONENT_NOT_FOUND"
	StatusVersionNotFound   = "VERSION_NOT_FOUND"
)

// Errors returned when a purl cannot be resolved at all.
var (
	ErrEmptyPurl         = errors.New("please specify a valid purl to query")
	ErrInvalidPurl       = errors.New("invalid purl")
	ErrComponentNotFound = errors.New("component not found")
	ErrNoDatabase        = errors.New("no serving database configured")
)

// Identity is the (name, type, version) tuple parsed straight out of a purl
// string, with no database lookup.
type Identity struct {
	PurlName string
	PurlType string
	Version  string
}

// Resolved is the outcome of a resolution: the component identity plus the
// version set and the linked source component.
type Resolved struct {
	Purl       string
	PurlID     string // component.purl_id (uuid, read as a string)
	PurlName   string
	PurlType   string
	Version    string
	Versions   []string
	StatusCode string
	SourcePurl *SourcePurl
}

// SourcePurl is the source-code component linked from component.source_purl_id.
// It is the fallback lookup target when the requested purl carries no data.
type SourcePurl struct {
	Purl     string
	PurlID   string
	PurlName string
	PurlType string
}

// Resolver resolves purls against the serving schema. One instance is safe to
// share across the service.
type Resolver struct {
	db *sqlx.DB
	s  *zap.SugaredLogger
}

// New builds a Resolver over the serving database handle.
//
// A nil logger is accepted and replaced with a no-op: the resolver only logs
// query failures that it also returns as errors, so a caller that has no logger
// at construction time (the handlers are built before a request context exists)
// loses nothing by omitting one.
func New(s *zap.SugaredLogger, database *sqlx.DB) *Resolver {
	if s == nil {
		s = zap.NewNop().Sugar()
	}
	return &Resolver{db: database, s: s}
}

// ParseIdentity decomposes a purl into the tuple used as the lookup key.
// purl_name is "namespace/name" when the purl carries a namespace (github,
// maven, bitbucket, ...) and just "name" otherwise.
func ParseIdentity(purl string) (Identity, error) {
	parsed, err := purlhelper.PurlFromString(purl)
	if err != nil {
		return Identity{}, err
	}
	name := parsed.Name
	if len(parsed.Namespace) > 0 {
		name = parsed.Namespace + "/" + parsed.Name
	}
	return Identity{PurlName: name, PurlType: parsed.Type, Version: parsed.Version}, nil
}

// servingPurlKey rebuilds the versionless `component.purl` lookup key from a
// parsed identity. purl_name already carries the namespace when there is one.
func servingPurlKey(id Identity) string {
	return "pkg:" + id.PurlType + "/" + id.PurlName
}

// component mirrors the identity columns of a `component` row.
type component struct {
	PurlID       string         `db:"purl_id"`
	Purl         string         `db:"purl"`
	SourcePurlID sql.NullString `db:"source_purl_id"`
}

// identityQuery reads the components matching any of the candidate spellings of
// a versionless purl (see candidateKeys).
//
// The CAST is not cosmetic: purl_id and source_purl_id are native DuckDB `uuid`
// columns, and without it the driver hands back the raw 16 bytes instead of the
// canonical string. It is a no-op on SQLite (TEXT) and PostgreSQL.
const identityQuery = `SELECT CAST(purl_id AS VARCHAR) AS purl_id, purl,
	CAST(source_purl_id AS VARCHAR) AS source_purl_id
FROM component WHERE purl IN (?)`

// sourceQuery reads the component a source_purl_id points at.
const sourceQuery = `SELECT CAST(purl_id AS VARCHAR) AS purl_id, purl,
	CAST(source_purl_id AS VARCHAR) AS source_purl_id
FROM component WHERE purl_id = ?`

// versionsQuery lists the known versions of a component.
const versionsQuery = `SELECT version FROM component_version WHERE purl_id = ?`

// Resolve maps a request onto a serving component. A version that is absent from
// the component's version set does not fail the resolution: the identity still
// resolves and StatusCode reports VERSION_NOT_FOUND, so the caller can decide
// whether to keep going. An unknown component is an error.
func (r *Resolver) Resolve(ctx context.Context, purl, requirement string) (Resolved, error) {
	if len(strings.TrimSpace(purl)) == 0 {
		return Resolved{}, ErrEmptyPurl
	}
	if r == nil || r.db == nil {
		return Resolved{}, ErrNoDatabase
	}
	id, err := ParseIdentity(purl)
	if err != nil || len(id.PurlType) == 0 || len(id.PurlName) == 0 {
		return Resolved{}, ErrInvalidPurl
	}

	comp, err := r.lookup(ctx, purl, id)
	if err != nil {
		return Resolved{}, err
	}

	var versions []string
	if err = r.db.SelectContext(ctx, &versions, r.db.Rebind(versionsQuery), comp.PurlID); err != nil {
		r.s.Errorf("Failed to read versions for %v: %v", purl, err)
		return Resolved{}, fmt.Errorf("failed to read the component versions: %w", err)
	}

	out := Resolved{
		Purl:       comp.Purl,
		PurlID:     comp.PurlID,
		PurlName:   id.PurlName,
		PurlType:   id.PurlType,
		Versions:   versions,
		StatusCode: StatusSuccess,
		SourcePurl: r.sourcePurl(ctx, comp.PurlID, comp.SourcePurlID),
	}

	r.pinVersion(&out, id, requirement)
	return out, nil
}

// ComponentID returns the purl_id of the component a versionless purl names,
// trying every spelling the export stores (see candidateKeys). It reads nothing
// else, so a caller that only needs the key - the transitive walk, one lookup per
// node - does not pay for the version list.
func (r *Resolver) ComponentID(ctx context.Context, purl string) (string, error) {
	if len(strings.TrimSpace(purl)) == 0 {
		return "", ErrEmptyPurl
	}
	if r == nil || r.db == nil {
		return "", ErrNoDatabase
	}
	id, err := ParseIdentity(purl)
	if err != nil || len(id.PurlType) == 0 || len(id.PurlName) == 0 {
		return "", ErrInvalidPurl
	}
	comp, err := r.lookup(ctx, purl, id)
	if err != nil {
		return "", err
	}
	return comp.PurlID, nil
}

// lookup reads the component row for a purl, by its stored purl only: the export
// indexes component(purl) and nothing else.
func (r *Resolver) lookup(ctx context.Context, purl string, id Identity) (component, error) {
	keys := candidateKeys(purl, id)
	query, args, err := sqlx.In(identityQuery, keys)
	if err != nil {
		return component{}, fmt.Errorf("failed to build the component lookup: %w", err)
	}
	var found []component
	if err = r.db.SelectContext(ctx, &found, r.db.Rebind(query), args...); err != nil {
		r.s.Errorf("Failed to resolve component for %v: %v", purl, err)
		return component{}, fmt.Errorf("failed to resolve the component: %w", err)
	}
	comp, ok := pickKey(keys, found)
	if !ok {
		return component{}, ErrComponentNotFound
	}
	return comp, nil
}

// pinVersion decides which version the lookup runs against, reproducing what the
// mining resolution did through go-models' pickOneUrl:
//
//   - an exact version (from the purl, or a requirement that is a plain literal)
//     must be one the component actually publishes, otherwise VERSION_NOT_FOUND;
//   - a semver range selects the highest published version satisfying it;
//   - no version and no requirement selects the highest published version.
//
// That last case is why this is not simply left empty as the serving reference
// does: a bare purl has always been answered with the latest version's data, and
// callers depend on it.
func (r *Resolver) pinVersion(out *Resolved, id Identity, requirement string) {
	requirement = strings.TrimSpace(requirement)

	exact := strings.TrimSpace(id.Version)
	if len(exact) == 0 && len(requirement) > 0 && !hasSemverOperator(requirement) {
		exact = requirement
	}
	if len(exact) > 0 {
		out.Version = exact
		// A "v" prefix is a labelling convention, not part of the version: a caller
		// asking for 5.4.7 means the component's v5.4.7, and vice versa. The mining
		// lookup matched both spellings, so this does too, and answers with the
		// spelling the component actually publishes.
		if stored, ok := matchVersion(out.Versions, exact); ok {
			out.Version = stored
		} else {
			out.StatusCode = StatusVersionNotFound
		}
		return
	}

	constraint := ""
	if hasSemverOperator(requirement) {
		constraint = requirement
	}
	picked, ok := SelectVersion(out.Versions, constraint)
	if !ok {
		out.StatusCode = StatusVersionNotFound
		return
	}
	out.Version = picked
}

// SourcePurlFor resolves the source component linked to a purl, or nil when
// there is none. Version-independent on purpose: a version missing from the
// original component must not stop the source lookup.
func (r *Resolver) SourcePurlFor(ctx context.Context, purl string) *SourcePurl {
	resolved, err := r.Resolve(ctx, purl, "")
	if err != nil {
		return nil
	}
	return resolved.SourcePurl
}

// sourcePurl follows component.source_purl_id to the component it references.
// Best-effort: a dangling reference simply means "no source".
//
// A component that names itself as its source has none to fall back to. That is
// most of them - 11.4 million of the 13.9 million links point back at their own
// row - and following one cost a scan of `component` by purl_id, which the export
// does not index: about 2.5 s of CPU, for an answer that could only repeat the
// lookup that had just come back empty.
func (r *Resolver) sourcePurl(ctx context.Context, purlID string, sourceID sql.NullString) *SourcePurl {
	if !sourceID.Valid || len(sourceID.String) == 0 || sourceID.String == purlID {
		return nil
	}
	var comp component
	if err := r.db.GetContext(ctx, &comp, r.db.Rebind(sourceQuery), sourceID.String); err != nil {
		return nil
	}
	id, err := ParseIdentity(comp.Purl)
	if err != nil {
		return &SourcePurl{Purl: comp.Purl, PurlID: comp.PurlID}
	}
	return &SourcePurl{Purl: comp.Purl, PurlID: comp.PurlID, PurlName: id.PurlName, PurlType: id.PurlType}
}

// matchVersion finds a version in the set, accepting the other spelling of a
// leading "v", and returns it as the component publishes it.
func matchVersion(versions []string, wanted string) (string, bool) {
	if slices.Contains(versions, wanted) {
		return wanted, true
	}
	toggled := toggleVersionPrefix(wanted)
	if toggled != wanted && slices.Contains(versions, toggled) {
		return toggled, true
	}
	return "", false
}

// toggleVersionPrefix adds a leading "v" to a version that has none and removes
// it from one that has it. A value that is not a version is returned unchanged.
// Mirrors go-models' SemverTogglePrefix, which the mining lookup used.
func toggleVersionPrefix(version string) string {
	if len(version) == 0 {
		return version
	}
	if _, err := semver.NewVersion(version); err != nil {
		return version
	}
	if version[0] != 'v' {
		return "v" + version
	}
	return strings.TrimLeft(version, "v")
}
