package transitive_dependencies

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// Grammar for the range version is defined on npmjs https://docs.npmjs.com/cli/v6/using-npm/semver#range-grammar

// PickFirstVersionFromNpmJsRange extracts the first version from a version range string
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
