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

package resolver

import (
	"context"
	"errors"
	"strings"
	"sync"

	compHelper "github.com/scanoss/go-component-helper/componenthelper"
	"github.com/scanoss/go-grpc-helper/pkg/grpc/domain"
	purlhelper "github.com/scanoss/go-purl-helper/pkg"
	"go.uber.org/zap"
)

// defaultMaxWorkers matches go-component-helper's default fan-out.
const defaultMaxWorkers = 5

// ResolveComponents turns the request DTOs into resolved components.
//
// It is the serving replacement for compHelper.GetComponentsVersion, which
// resolved against the mining tables `all_urls`, `mines` and `versions` - none of
// which the serving schema has. The output type is unchanged, so everything
// downstream of it carries over untouched.
func ResolveComponents(ctx context.Context, s *zap.SugaredLogger, r *Resolver,
	componentDTOs []compHelper.ComponentDTO, maxWorkers int,
) []compHelper.Component {
	components, _ := ResolveComponentsWithIDs(ctx, s, r, componentDTOs, maxWorkers)
	return components
}

// ResolveComponentsWithIDs is ResolveComponents that also returns each
// component's purl_id, index for index, so a caller reading per-release tables
// does not resolve the purl a second time. The id is empty for a component that
// did not resolve.
func ResolveComponentsWithIDs(ctx context.Context, s *zap.SugaredLogger, r *Resolver,
	componentDTOs []compHelper.ComponentDTO, maxWorkers int,
) ([]compHelper.Component, []string) {
	components := SanitiseComponents(s, componentDTOs)
	if len(components) == 0 {
		return nil, nil
	}
	purlIDs := make([]string, len(components))
	if maxWorkers <= 0 {
		maxWorkers = defaultMaxWorkers
	}

	jobs := make(chan int, len(components))
	wg := sync.WaitGroup{}
	for i := 0; i < min(maxWorkers, len(components)); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				purlIDs[idx] = resolveOne(ctx, s, r, &components[idx])
			}
		}()
	}
	for i := range components {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return components, purlIDs
}

// resolveOne fills in the version, the version set and the resolution status for
// one already-sanitised component, and returns its purl_id (empty when it did not
// resolve). Components that failed sanitisation keep the status they were given.
func resolveOne(ctx context.Context, s *zap.SugaredLogger, r *Resolver, component *compHelper.Component) string {
	if component.Status.StatusCode != domain.Success {
		return ""
	}
	component.Versions = []string{}

	resolved, err := r.Resolve(ctx, component.Purl, component.Requirement)
	switch {
	case errors.Is(err, ErrComponentNotFound), errors.Is(err, ErrInvalidPurl), errors.Is(err, ErrEmptyPurl):
		s.Warnf("Purl doesn't exist: %s, %v", component.Purl, err)
		component.Status = domain.ComponentStatus{Message: "Component not found", StatusCode: domain.ComponentNotFound}
		return ""
	case err != nil:
		s.Errorf("Failed to resolve component %s (%s): %v", component.Purl, component.Requirement, err)
		component.Status = domain.ComponentStatus{Message: "Component not found", StatusCode: domain.ComponentNotFound}
		return ""
	}

	// The component exists, so its version list is reported either way; only the
	// status and the pinned version differ.
	component.Versions = resolved.Versions
	if resolved.StatusCode == StatusVersionNotFound {
		s.Warnf("Failed to get component version: %s, %s", component.Purl, component.Requirement)
		component.Status = domain.ComponentStatus{Message: "Component version not found", StatusCode: domain.VersionNotFound}
		return resolved.PurlID
	}
	if len(resolved.Version) > 0 {
		component.Version = resolved.Version
	}
	return resolved.PurlID
}

// SanitiseComponents validates and normalises the request DTOs.
//
// A port of go-component-helper's unexported sanitiseComponents, kept faithful to
// it because it defines the service's input contract: an empty or unparseable
// purl is rejected with INVALID_PURL, and a version carried in the purl is moved
// into the requirement, taking precedence over any requirement supplied.
//
// The version is stripped with a string replace on the original purl rather than
// by re-rendering the parsed purl, so percent-encoding survives: scoped npm
// packages are stored as pkg:npm/%40scope/name, not pkg:npm/@scope/name, and
// re-rendering would break the database lookup.
func SanitiseComponents(s *zap.SugaredLogger, componentDTOs []compHelper.ComponentDTO) []compHelper.Component {
	var components []compHelper.Component
	for _, dto := range componentDTOs {
		if len(dto.Purl) == 0 {
			components = append(components, invalidComponent(dto.Purl, dto.Purl, dto.Requirement, dto.Requirement, "Empty Purl"))
			continue
		}
		originalPurl, originalRequirement := dto.Purl, dto.Requirement

		packageURL, err := purlhelper.PurlFromString(dto.Purl)
		if err != nil {
			s.Warnf("Failed to parse PURL %q (requirement: %q): %v", dto.Purl, dto.Requirement, err)
			components = append(components, invalidComponent(dto.Purl, originalPurl, dto.Requirement, originalRequirement, "Invalid Purl"))
			continue
		}
		if len(packageURL.Version) > 0 {
			dto.Requirement = packageURL.Version
			dto.Purl = strings.Replace(dto.Purl, "@"+packageURL.Version, "", 1)
		}

		componentName, err := compHelper.ComponentNameFromString(dto.Purl)
		if err != nil {
			s.Warnf("Failed to extract component name from PURL %q (requirement: %q): %v", dto.Purl, dto.Requirement, err)
			components = append(components, invalidComponent(dto.Purl, originalPurl, dto.Requirement, originalRequirement, "Invalid Purl"))
			continue
		}

		projectURL, err := purlhelper.ProjectUrl(componentName, packageURL.Type)
		if err != nil {
			s.Warnf("Failed to derive project URL for PURL %q (requirement: %q): %v. URL will be empty.", dto.Purl, dto.Requirement, err)
			projectURL = ""
		}

		qualifiers := make(map[string]string, len(packageURL.Qualifiers))
		for _, q := range packageURL.Qualifiers {
			qualifiers[q.Key] = q.Value
		}

		components = append(components, compHelper.Component{
			Purl:                dto.Purl,
			OriginalPurl:        originalPurl,
			Requirement:         dto.Requirement,
			OriginalRequirement: originalRequirement,
			PurlType:            packageURL.Type,
			PurlName:            packageURL.Name,
			PurlNamespace:       packageURL.Namespace,
			PurlQualifiers:      qualifiers,
			PurlSubpath:         packageURL.Subpath,
			Name:                componentName,
			URL:                 projectURL,
			Status:              domain.ComponentStatus{StatusCode: domain.Success},
		})
	}
	return components
}

// invalidComponent builds the rejected-input result for a purl that never made it
// as far as a database lookup.
func invalidComponent(purl, originalPurl, requirement, originalRequirement, message string) compHelper.Component {
	return compHelper.Component{
		Purl:                purl,
		OriginalPurl:        originalPurl,
		Requirement:         requirement,
		OriginalRequirement: originalRequirement,
		Status:              domain.ComponentStatus{Message: message, StatusCode: domain.InvalidPurl},
	}
}
