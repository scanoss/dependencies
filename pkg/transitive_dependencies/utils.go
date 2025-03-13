package transitive_dependencies

import (
	"fmt"
	"regexp"
	"scanoss.com/dependencies/pkg/shared"
	"strings"

	"github.com/Masterminds/semver/v3"
	packageurl "github.com/package-url/packageurl-go"
)

// Grammar for the range version is defined on npmjs https://docs.npmjs.com/cli/v6/using-npm/semver#range-grammar

// PickFirstVersionFromNpmJsRange extracts the first version from a version range string
// TODO: Use https://www.npmjs.com/package/@npmcli/arborist ?
func PickFirstVersionFromNpmJsRange(requirement string) (string, error) {
	if requirement == "" {
		return "", fmt.Errorf("empty version requirement")
	}

	if requirement == "*" || requirement == "x" || requirement == "X" {
		return "", fmt.Errorf("cannot coerce version from wildcard ")
	}

	// Split on || and take first part only
	parts := strings.Split(requirement, "||")
	firstRange := strings.TrimSpace(parts[0])

	// Check if valid constraint to avoid further processing of invalid inputs
	_, err := semver.NewConstraint(firstRange)
	if err != nil {
		return "", fmt.Errorf("invalid version constraint: %s", firstRange)
	}

	// Extract the first version string, ignoring operators
	var versionStr string

	// Handle hyphen ranges (e.g., "1.2.3 - 2.3.4")
	hyphenRegex := regexp.MustCompile(`^(\S+)\s+-\s+(\S+)$`)
	if matches := hyphenRegex.FindStringSubmatch(firstRange); matches != nil && len(matches) >= 1 {
		return normalizeVersion(matches[1])
	}

	if strings.Contains(firstRange, " ") {
		// Handle combined ranges with spaces (e.g., ">=1.2.3 <2.0.0")
		spaceParts := strings.Fields(firstRange)
		if len(spaceParts) > 0 {
			// Extract version from the first part by removing any operators
			firstPart := spaceParts[0]
			versionStr = removeOperators(firstPart)
		}
	} else {
		// Handle simple version with possible operator
		versionStr = removeOperators(firstRange)
	}

	// Normalize the version
	return normalizeVersion(versionStr)
}

// removeOperators removes any version operators from the string
func removeOperators(input string) string {
	// Remove common operators
	input = strings.TrimPrefix(input, ">=")
	input = strings.TrimPrefix(input, "<=")
	input = strings.TrimPrefix(input, ">")
	input = strings.TrimPrefix(input, "<")
	input = strings.TrimPrefix(input, "=")
	input = strings.TrimPrefix(input, "^")
	input = strings.TrimPrefix(input, "~")

	return strings.TrimSpace(input)
}

// normalizeVersion converts partial versions, wildcards, etc. to a full semver version
func normalizeVersion(versionStr string) (string, error) {
	// Handle wildcards
	versionStr = strings.ReplaceAll(versionStr, "X", "0")
	versionStr = strings.ReplaceAll(versionStr, "x", "0")
	versionStr = strings.ReplaceAll(versionStr, "*", "0")

	// Complete partial versions
	parts := strings.Split(versionStr, ".")
	if len(parts) == 1 {
		versionStr = versionStr + ".0.0"
	} else if len(parts) == 2 {
		versionStr = versionStr + ".0"
	}

	// Check if we have a valid semver version
	version, err := semver.NewVersion(versionStr)
	if err != nil {
		return "", fmt.Errorf("cannot normalize version: %s", err)
	}

	return version.String(), nil
}

// GetPurlFromPackageName convert packageName@version to PackageURL
func GetPurlFromPackageName(packageName string, ecosystem string) (*packageurl.PackageURL, error) {
	if ecosystem == "" {
		return nil, fmt.Errorf("empty ecosystem")
	}

	_, ok := shared.SupportedEcosystems[ecosystem]
	if !ok {
		return nil, fmt.Errorf("invalid ecosystem: %s", ecosystem)
	}

	if !strings.Contains(packageName, "@") {
		return nil, fmt.Errorf("no version separator for: %s", packageName)
	}
	p := strings.Split(packageName, "@")
	// Example with a specific version
	var versionedPurl = packageurl.NewPackageURL(
		shared.SupportedEcosystems[ecosystem], // type
		"",                                    // namespace
		p[0],                                  // name
		p[1],                                  // version
		nil,                                   // qualifiers
		"",                                    // subpath
	)
	return versionedPurl, nil
}

// GetPackageNameFromPurl convert purl to package name
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

// GetPurlWithoutVersion convert PackageURL to purl without version
func GetPurlWithoutVersion(p *packageurl.PackageURL) (string, error) {
	purl := p.String()
	if !strings.Contains(purl, "@") {
		return "", fmt.Errorf("package URL missing version information: %q", purl)
	}
	return strings.Split(purl, "@")[0], nil
}

func ConvertResultToDependency(packageName string, ecosystem string) (Dependency, error) {
	packageUrl, err := GetPurlFromPackageName(packageName, ecosystem)
	if err != nil {
		return Dependency{}, err
	}
	// Extract base purls without versions for parent dependency
	purl, err := GetPurlWithoutVersion(packageUrl)
	if err != nil {
		return Dependency{}, fmt.Errorf("error extracting base purl from %v: %v", purl, err)
	}
	return Dependency{
		Purl:    purl,
		Version: packageUrl.Version,
	}, nil
}
