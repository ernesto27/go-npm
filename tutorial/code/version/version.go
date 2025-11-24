package version

import (
	"go-npm/manifest"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
)

type VersionInfo struct {
}

func NewVersionInfo() *VersionInfo {
	return &VersionInfo{}
}

// getVersion resolves a version constraint to a specific version string
// It supports all npm semver ranges: ^, ~, >=, <=, >, <, ||, hyphen ranges, wildcards, and exact versions
func (v *VersionInfo) GetVersion(version string, npmPackage manifest.NPMPackage) string {
	// Handle empty version or "latest" keyword
	if version == "" || version == "latest" || version == "*" {
		return npmPackage.DistTags.Latest
	}

	// Check if version is a known dist-tag
	if version == "next" && npmPackage.DistTags.Next != "" {
		return npmPackage.DistTags.Next
	}

	// Try to parse as semver constraint
	constraint, err := semver.NewConstraint(version)
	if err != nil {
		// If parsing fails, try as exact version match
		if versionObj, exists := npmPackage.Versions[version]; exists {
			return versionObj.Version
		}
		// Fallback to latest for invalid constraints
		return npmPackage.DistTags.Latest
	}

	// Filter versions that match the constraint
	var matchingVersions []*semver.Version
	for vStr := range npmPackage.Versions {
		semverVersion, err := semver.NewVersion(vStr)
		if err != nil {
			continue // Skip invalid versions in registry
		}
		if constraint.Check(semverVersion) {
			matchingVersions = append(matchingVersions, semverVersion)
		}
	}

	// If no versions match, fallback to latest
	if len(matchingVersions) == 0 {
		return npmPackage.DistTags.Latest
	}

	// Sort versions and return the highest
	sort.Sort(semver.Collection(matchingVersions))
	bestVersion := matchingVersions[len(matchingVersions)-1]

	// Return the original version string (preserves exact format from registry)
	originalVersion := bestVersion.Original()

	// Fallback to String() if Original() doesn't exist in the map (normalization edge case)
	if _, exists := npmPackage.Versions[originalVersion]; exists {
		return originalVersion
	}

	stringVersion := bestVersion.String()
	if _, exists := npmPackage.Versions[stringVersion]; exists {
		return stringVersion
	}

	// If neither exists (shouldn't happen), try with "v" prefix removed
	trimmedOriginal := strings.TrimPrefix(originalVersion, "v")
	if _, exists := npmPackage.Versions[trimmedOriginal]; exists {
		return trimmedOriginal
	}

	trimmedString := strings.TrimPrefix(stringVersion, "v")
	if _, exists := npmPackage.Versions[trimmedString]; exists {
		return trimmedString
	}

	// Last resort: return the original format
	return trimmedOriginal
}
