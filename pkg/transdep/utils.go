package transdep

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/package-url/packageurl-go"
	"scanoss.com/dependencies/pkg/config"
	"scanoss.com/dependencies/pkg/shared"
)

// NPMJS range version is defined here: https://docs.npmjs.com/cli/v6/using-npm/semver#range-grammar

var (
	versionRegex = regexp.MustCompile(`(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)`)
)

// PickFirstVersionFromRange try to extract the first version from a version range string (no ecosystem dependent).
func PickFirstVersionFromRange(requirement string) (string, error) {
	version := versionRegex.FindString(requirement)
	if len(version) == 0 {
		return "", errors.New("cannot determine version from requirement")
	}
	return version, nil
}

// GetPurlFromPurlName convert packageName@version to PackageURL.
func GetPurlFromPurlName(packageName string, version string, ecosystem string) (*packageurl.PackageURL, error) {
	if ecosystem == "" {
		return nil, fmt.Errorf("empty ecosystem")
	}

	_, ok := shared.RegisteredEcosystems[ecosystem]
	if !ok {
		return nil, fmt.Errorf("invalid ecosystem: %s", ecosystem)
	}

	// Example with a specific version
	var versionedPurl = packageurl.NewPackageURL(
		ecosystem,   // type
		"",          // namespace
		packageName, // name
		version,     // version
		nil,         // qualifiers
		"",          // subpath
	)
	return versionedPurl, nil
}

// GetPackageNameFromPurl convert purl to package name.
func ExtractPackageIdentifierFromPurl(purl string) (string, error) {
	// Parse the purl string into a PackageURL object
	p, err := packageurl.FromString(purl)
	if err != nil {
		return "", fmt.Errorf("failed to parse package URL: %w", err)
	}

	// For Maven packages, combine namespace (groupId) and name (artifactId)
	if p.Type == "maven" && p.Namespace != "" {
		return fmt.Sprintf("%s/%s", p.Namespace, p.Name), nil
	}

	// Return just the name component
	return p.Name, nil
}

// GetPurlWithoutVersion convert PackageURL to purl without version.
func GetPurlWithoutVersion(p *packageurl.PackageURL) (string, error) {
	purl := p.String()
	if !strings.Contains(purl, "@") {
		return "", fmt.Errorf("package URL missing version information: %q", purl)
	}
	return strings.Split(purl, "@")[0], nil
}

func ExtractDependencyFromJob(job DependencyJob) (Dependency, error) {
	packageURL, err := GetPurlFromPurlName(job.PurlName, job.Version, job.Ecosystem)
	if err != nil {
		return Dependency{}, err
	}
	// Extract base purls without versions for parent dependency
	purl, err := GetPurlWithoutVersion(packageURL)
	if err != nil {
		return Dependency{}, fmt.Errorf("error extracting base purl from %v: %v", purl, err)
	}
	return Dependency{
		Purl:    purl,
		Version: packageURL.Version,
	}, nil
}

func GetMaxResponseLimit(config config.ServerConfig, limit *int) int {
	if limit != nil {
		if *limit > config.TransitiveResources.MaxResponseSize {
			return config.TransitiveResources.MaxResponseSize
		}
		if *limit == 0 {
			return config.TransitiveResources.DefaultResponseSize
		}
		return *limit
	}
	return config.TransitiveResources.DefaultResponseSize
}

func GetMaxDepthLimit(config config.ServerConfig, depth *int) int {
	if depth != nil {
		if *depth > config.TransitiveResources.MaxDepth {
			return config.TransitiveResources.MaxDepth
		}
		if *depth == 0 {
			return config.TransitiveResources.DefaultDepth
		}
		return *depth
	}
	return config.TransitiveResources.DefaultDepth
}
