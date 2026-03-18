// Package shared provides common types and constants used across the application.
package shared

type EcosystemStorageLocation struct {
	Table string
}

var RegisteredEcosystems = map[string]EcosystemStorageLocation{
	"composer": {
		Table: "composer",
	},
	"crates": {
		Table: "crates",
	},
	"maven": {
		Table: "maven",
	},
	"npm": {
		Table: "npmjs",
	},
	"gem": {
		Table: "ruby",
	},
}
