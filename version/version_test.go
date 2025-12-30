package version

import (
	"github.com/ernesto27/go-npm/manifest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createTestPackage(versions []string, latest string) *manifest.NPMPackage {
	pkg := &manifest.NPMPackage{
		DistTags: manifest.DistTags{
			"latest": latest,
		},
		Versions: make(map[string]manifest.Version),
	}

	for _, v := range versions {
		pkg.Versions[v] = manifest.Version{
			Version: v,
		}
	}

	return pkg
}

func TestInfo_GetVersion(t *testing.T) {
	testCases := []struct {
		name     string
		version  string
		versions []string
		latest   string
		expected string
	}{
		// Empty version and latest keyword
		{
			name:     "Empty version should return latest",
			version:  "",
			versions: []string{"1.0.0", "1.1.0", "2.0.0"},
			latest:   "2.0.0",
			expected: "2.0.0",
		},
		{
			name:     "Asterisk wildcard",
			version:  "*",
			versions: []string{"1.0.0", "1.5.0", "2.3.1"},
			latest:   "2.3.1",
			expected: "2.3.1",
		},
		{
			name:     "Latest keyword",
			version:  "latest",
			versions: []string{"1.0.0", "1.5.0", "2.3.1"},
			latest:   "2.3.1",
			expected: "2.3.1",
		},

		// Exact versions
		{
			name:     "Exact version exists",
			version:  "1.2.3",
			versions: []string{"1.0.0", "1.2.3", "2.0.0"},
			latest:   "2.0.0",
			expected: "1.2.3",
		},
		{
			name:     "Exact version does not exist",
			version:  "1.2.4",
			versions: []string{"1.0.0", "1.2.3", "2.0.0"},
			latest:   "2.0.0",
			expected: "2.0.0", // Falls back to latest
		},

		// Caret ranges (^)
		{
			name:     "Caret allows minor and patch updates - major 1",
			version:  "^1.2.3",
			versions: []string{"1.0.0", "1.2.3", "1.2.5", "1.3.0", "1.9.9", "2.0.0", "2.1.0"},
			latest:   "2.1.0",
			expected: "1.9.9", // Highest in major version 1
		},
		{
			name:     "Caret with major version 0",
			version:  "^0.2.3",
			versions: []string{"0.1.0", "0.2.3", "0.2.5", "0.3.0", "1.0.0"},
			latest:   "1.0.0",
			expected: "0.2.5", // For 0.x, ^0.2.3 means >=0.2.3 <0.3.0 (only patch updates)
		},
		{
			name:     "Caret with exact match only",
			version:  "^1.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "1.0.0",
		},
		{
			name:     "Caret with multiple candidates",
			version:  "^2.0.0",
			versions: []string{"1.9.9", "2.0.0", "2.0.1", "2.1.0", "2.5.7", "3.0.0"},
			latest:   "3.0.0",
			expected: "2.5.7",
		},
		{
			name:     "Caret with no matching versions",
			version:  "^5.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0", "4.0.0"},
			latest:   "4.0.0",
			expected: "4.0.0", // Falls back to latest
		},
		{
			name:     "Caret with lower base version",
			version:  "^1.0.0",
			versions: []string{"0.9.0", "1.0.0", "1.1.0", "1.2.0", "2.0.0"},
			latest:   "2.0.0",
			expected: "1.2.0",
		},

		// Tilde ranges (~)
		{
			name:     "Tilde allows patch updates only",
			version:  "~1.2.3",
			versions: []string{"1.0.0", "1.2.3", "1.2.5", "1.2.9", "1.3.0", "2.0.0"},
			latest:   "2.0.0",
			expected: "1.2.9", // Highest patch in 1.2.x
		},
		{
			name:     "Tilde with exact match only",
			version:  "~1.2.3",
			versions: []string{"1.2.3", "1.3.0", "2.0.0"},
			latest:   "2.0.0",
			expected: "1.2.3",
		},
		{
			name:     "Tilde with no higher patch version",
			version:  "~2.1.5",
			versions: []string{"2.0.0", "2.1.0", "2.1.3", "2.1.5", "2.2.0"},
			latest:   "2.2.0",
			expected: "2.1.5",
		},
		{
			name:     "Tilde with multiple patch versions",
			version:  "~3.0.0",
			versions: []string{"2.9.9", "3.0.0", "3.0.1", "3.0.5", "3.0.10", "3.1.0"},
			latest:   "3.1.0",
			expected: "3.0.10",
		},
		{
			name:     "Tilde with no matching versions",
			version:  "~5.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0", "4.0.0"},
			latest:   "4.0.0",
			expected: "4.0.0", // Falls back to latest
		},
		{
			name:     "Tilde excludes minor version changes",
			version:  "~1.2.0",
			versions: []string{"1.1.9", "1.2.0", "1.2.1", "1.3.0", "2.0.0"},
			latest:   "2.0.0",
			expected: "1.2.1", // Does not include 1.3.0
		},

		// Complex ranges
		{
			name:     "Range with >= and <",
			version:  ">= 2.1.2 < 3.0.0",
			versions: []string{"2.0.0", "2.1.0", "2.1.2", "2.5.0", "2.9.9", "3.0.0", "3.1.0"},
			latest:   "3.1.0",
			expected: "2.9.9",
		},
		{
			name:     "Range with >= and <= (inclusive)",
			version:  ">= 1.0.0 <= 2.0.0",
			versions: []string{"0.9.0", "1.0.0", "1.5.0", "2.0.0", "2.1.0"},
			latest:   "2.1.0",
			expected: "2.0.0",
		},
		{
			name:     "Range with > and < (exclusive)",
			version:  "> 1.0.0 < 2.0.0",
			versions: []string{"1.0.0", "1.0.1", "1.5.0", "1.9.9", "2.0.0"},
			latest:   "2.0.0",
			expected: "1.9.9",
		},
		{
			name:     "Range with > and <= (mixed)",
			version:  "> 1.5.0 <= 2.5.0",
			versions: []string{"1.5.0", "1.6.0", "2.0.0", "2.5.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "2.5.0",
		},
		{
			name:     "Range with no matching versions",
			version:  ">= 5.0.0 < 6.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0", "4.0.0"},
			latest:   "4.0.0",
			expected: "4.0.0", // Falls back to latest
		},
		{
			name:     "Narrow range with one match",
			version:  ">= 1.2.3 < 1.2.5",
			versions: []string{"1.2.0", "1.2.3", "1.2.4", "1.2.5", "1.3.0"},
			latest:   "1.3.0",
			expected: "1.2.4",
		},
		{
			name:     "Range at boundary (lower bound inclusive)",
			version:  ">= 1.0.0 < 2.0.0",
			versions: []string{"0.9.9", "1.0.0", "1.5.0", "2.0.0"},
			latest:   "2.0.0",
			expected: "1.5.0",
		},

		// Wildcards
		{
			name:     "Single x returns latest",
			version:  "x",
			versions: []string{"1.0.0", "2.0.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "3.0.0",
		},
		{
			name:     "Major.x matches any minor/patch in that major",
			version:  "1.x",
			versions: []string{"1.0.0", "1.2.0", "1.5.9", "2.0.0", "2.1.0"},
			latest:   "2.1.0",
			expected: "1.5.9",
		},
		{
			name:     "Major.minor.x matches any patch",
			version:  "2.1.x",
			versions: []string{"2.0.0", "2.1.0", "2.1.5", "2.1.9", "2.2.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "2.1.9",
		},
		{
			name:     "Case insensitive X",
			version:  "1.X",
			versions: []string{"1.0.0", "1.3.0", "1.7.2", "2.0.0"},
			latest:   "2.0.0",
			expected: "1.7.2",
		},
		{
			name:     "Major.X.X pattern",
			version:  "2.X.X",
			versions: []string{"1.9.9", "2.0.0", "2.1.0", "2.5.7", "3.0.0"},
			latest:   "3.0.0",
			expected: "2.5.7",
		},
		{
			name:     "No matching versions for wildcard",
			version:  "5.x",
			versions: []string{"1.0.0", "2.0.0", "3.0.0", "4.0.0"},
			latest:   "4.0.0",
			expected: "4.0.0", // Falls back to latest
		},
		{
			name:     "Wildcard with exact major match",
			version:  "3.x",
			versions: []string{"3.0.0", "3.0.1", "3.1.0", "4.0.0"},
			latest:   "4.0.0",
			expected: "3.1.0",
		},

		// OR constraints (||)
		{
			name:     "OR with two caret ranges",
			version:  "^1.0.0 || ^2.0.0",
			versions: []string{"1.0.0", "1.2.0", "1.9.9", "2.0.0", "2.1.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "2.1.0", // Highest between 1.9.9 and 2.1.0
		},
		{
			name:     "OR with exact versions",
			version:  "1.0.0 || 2.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "2.0.0", // Higher of the two
		},
		{
			name:     "OR with one matching constraint",
			version:  "^1.0.0 || ^5.0.0",
			versions: []string{"1.0.0", "1.5.0", "2.0.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "1.5.0", // Only ^1.0.0 matches
		},
		{
			name:     "OR with tilde and caret",
			version:  "~1.2.3 || ^2.0.0",
			versions: []string{"1.2.3", "1.2.5", "1.3.0", "2.0.0", "2.5.0"},
			latest:   "2.5.0",
			expected: "2.5.0", // ^2.0.0 gives higher version
		},
		{
			name:     "OR with no matching constraints",
			version:  "^5.0.0 || ^6.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0", "4.0.0"},
			latest:   "4.0.0",
			expected: "4.0.0", // Falls back to latest
		},
		{
			name:     "OR with wildcards",
			version:  "1.x || 3.x",
			versions: []string{"1.0.0", "1.5.0", "2.0.0", "3.0.0", "3.2.0"},
			latest:   "3.2.0",
			expected: "3.2.0", // Highest between 1.5.0 and 3.2.0
		},
		{
			name:     "OR with multiple constraints (3 options)",
			version:  "^1.0.0 || ^2.0.0 || ^3.0.0",
			versions: []string{"1.0.0", "1.1.0", "2.0.0", "2.2.0", "3.0.0", "3.5.0"},
			latest:   "3.5.0",
			expected: "3.5.0",
		},

		// Simple ranges
		{
			name:     "Greater than or equal",
			version:  ">=1.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "3.0.0",
		},
		{
			name:     "Greater than or equal - returns highest version >= base",
			version:  ">=1.5.0",
			versions: []string{"1.0.0", "1.5.0", "1.8.0", "2.0.0", "2.5.0"},
			latest:   "2.5.0",
			expected: "2.5.0",
		},
		{
			name:     "Greater or equal - exact match at base",
			version:  ">=2.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "3.0.0",
		},
		{
			name:     "Greater or equal - no matching versions",
			version:  ">=5.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0", "4.0.0"},
			latest:   "4.0.0",
			expected: "4.0.0", // Falls back to latest
		},
		{
			name:     "Greater or equal - with patch versions",
			version:  ">=1.2.3",
			versions: []string{"1.2.0", "1.2.3", "1.2.5", "1.3.0", "2.0.0"},
			latest:   "2.0.0",
			expected: "2.0.0",
		},
		{
			name:     "Less than or equal",
			version:  "<=2.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "2.0.0",
		},
		{
			name:     "Less or equal - returns highest version <= base",
			version:  "<=2.0.0",
			versions: []string{"1.0.0", "1.5.0", "2.0.0", "2.5.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "2.0.0",
		},
		{
			name:     "Less or equal - no matching versions",
			version:  "<=0.5.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "3.0.0", // Falls back to latest
		},
		{
			name:     "Less or equal - with patch versions",
			version:  "<=1.2.5",
			versions: []string{"1.0.0", "1.2.3", "1.2.5", "1.2.7", "1.3.0"},
			latest:   "1.3.0",
			expected: "1.2.5",
		},
		{
			name:     "Greater than",
			version:  ">1.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "3.0.0",
		},
		{
			name:     "Greater than - returns highest version > base",
			version:  ">1.5.0",
			versions: []string{"1.0.0", "1.5.0", "1.8.0", "2.0.0", "2.5.0"},
			latest:   "2.5.0",
			expected: "2.5.0",
		},
		{
			name:     "Greater than - excludes exact match",
			version:  ">2.0.0",
			versions: []string{"1.0.0", "2.0.0", "2.0.1", "3.0.0"},
			latest:   "3.0.0",
			expected: "3.0.0",
		},
		{
			name:     "Greater than - no matching versions",
			version:  ">5.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0", "5.0.0"},
			latest:   "5.0.0",
			expected: "5.0.0", // Falls back to latest
		},
		{
			name:     "Greater than - with patch versions",
			version:  ">1.2.3",
			versions: []string{"1.2.0", "1.2.3", "1.2.4", "1.3.0", "2.0.0"},
			latest:   "2.0.0",
			expected: "2.0.0",
		},
		{
			name:     "Less than",
			version:  "<2.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "1.0.0",
		},
		{
			name:     "Less than - returns highest version < base",
			version:  "<2.0.0",
			versions: []string{"1.0.0", "1.5.0", "1.9.9", "2.0.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "1.9.9",
		},
		{
			name:     "Less than - excludes exact match",
			version:  "<2.0.0",
			versions: []string{"1.0.0", "1.5.0", "2.0.0"},
			latest:   "2.0.0",
			expected: "1.5.0",
		},
		{
			name:     "Less than - no matching versions",
			version:  "<1.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "3.0.0", // Falls back to latest
		},
		{
			name:     "Less than - with patch versions",
			version:  "<1.3.0",
			versions: []string{"1.0.0", "1.2.3", "1.2.9", "1.3.0", "2.0.0"},
			latest:   "2.0.0",
			expected: "1.2.9",
		},

		// Hyphen ranges
		{
			name:     "Hyphen range - inclusive on both ends",
			version:  "1.0.0 - 2.0.0",
			versions: []string{"0.9.0", "1.0.0", "1.5.0", "2.0.0", "2.1.0"},
			latest:   "2.1.0",
			expected: "2.0.0",
		},
		{
			name:     "Hyphen range - narrow range",
			version:  "1.2.3 - 1.2.5",
			versions: []string{"1.2.0", "1.2.3", "1.2.4", "1.2.5", "1.3.0"},
			latest:   "1.3.0",
			expected: "1.2.5",
		},
		{
			name:     "Hyphen range - no matching versions",
			version:  "5.0.0 - 6.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0", "4.0.0"},
			latest:   "4.0.0",
			expected: "4.0.0", // Falls back to latest
		},
		{
			name:     "Hyphen range - single version in range",
			version:  "1.5.0 - 2.5.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "2.0.0",
		},
		{
			name:     "Hyphen range - all versions in range",
			version:  "0.1.0 - 10.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "3.0.0",
		},
		{
			name:     "Hyphen range - exact boundaries",
			version:  "2.0.0 - 3.0.0",
			versions: []string{"1.9.9", "2.0.0", "2.5.0", "3.0.0", "3.0.1"},
			latest:   "3.0.1",
			expected: "3.0.0",
		},

		// Regression tests
		{
			name:     "Caret ^7.0.0 should NOT match 8.x (wrap-ansi bug)",
			version:  "^7.0.0",
			versions: []string{"6.0.0", "7.0.0", "7.1.0", "8.0.0", "8.1.0"},
			latest:   "8.1.0",
			expected: "7.1.0", // Should be highest 7.x, NOT 8.x
		},

		// Edge cases
		{
			name:     "Package with only one version",
			version:  "^1.0.0",
			versions: []string{"1.0.0"},
			latest:   "1.0.0",
			expected: "1.0.0",
		},
		{
			name:     "Empty versions map",
			version:  "^1.0.0",
			versions: []string{},
			latest:   "",
			expected: "",
		},
		{
			name:     "Very high version numbers",
			version:  "^100.200.300",
			versions: []string{"100.200.300", "100.200.400", "100.300.0", "200.0.0"},
			latest:   "200.0.0",
			expected: "100.300.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vi := New()
			pkg := createTestPackage(tc.versions, tc.latest)
			result := vi.GetVersion(tc.version, pkg)
			assert.Equal(t, tc.expected, result)
		})
	}
}
