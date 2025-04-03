package shared

var EcosystemDBMapper = map[string]string{
	"composer": "composer",
	"crates":   "crates",
	"maven":    "maven",
	"npm":      "npmjs",
	"gem":      "ruby",
}

var SupportedEcosystems = map[string]struct{}{
	"composer": {},
	"crates":   {},
	"maven":    {},
	"npm":      {},
	"gem":      {},
}
