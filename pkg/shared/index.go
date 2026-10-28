// Package shared provides common types and constants used across the application.
package shared

// RegisteredEcosystems are the purl types the transitive walk accepts. Mining kept
// each one's dependencies in its own <ecosystem>_dependencies table; serving keeps
// them all in component_version_dependency, so this is now only the accepted set.
var RegisteredEcosystems = map[string]struct{}{
	"composer": {},
	"crates":   {},
	"maven":    {},
	"npm":      {},
	"gem":      {},
}
