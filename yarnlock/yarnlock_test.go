package yarnlock

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsYarnV1_Valid(t *testing.T) {
	content := []byte(`# yarn lockfile v1

express@^4.18.2:
  version "4.18.2"
  resolved "https://registry.yarnpkg.com/express/-/express-4.18.2.tgz"
  integrity sha512-abc123
`)
	assert.True(t, IsYarnV1(content))
}

func TestIsYarnV1_ValidWithEmptyLines(t *testing.T) {
	content := []byte(`
# yarn lockfile v1

express@^4.18.2:
  version "4.18.2"
`)
	// First non-empty line is the header
	assert.True(t, IsYarnV1(content))
}

func TestIsYarnV1_Empty(t *testing.T) {
	content := []byte(``)
	assert.False(t, IsYarnV1(content))
}

func TestParseContent_BasicEntry(t *testing.T) {
	content := []byte(`# yarn lockfile v1

express@^4.18.2:
  version "4.18.2"
  resolved "https://registry.yarnpkg.com/express/-/express-4.18.2.tgz#abc123"
  integrity sha512-5/4oz/PTmN0HMkvzUHPsQ/oJc2EHSbZHa==
`)

	yarnLock, err := ParseContent(content)
	assert.NoError(t, err)
	assert.NotNil(t, yarnLock)

	// Check entry exists (keyed by name@version)
	entry, exists := yarnLock.Entries["express@4.18.2"]
	assert.True(t, exists)
	assert.Equal(t, "express", entry.Name)
	assert.Equal(t, "4.18.2", entry.Version)
	assert.Equal(t, "https://registry.yarnpkg.com/express/-/express-4.18.2.tgz#abc123", entry.Resolved)
	assert.Equal(t, "sha512-5/4oz/PTmN0HMkvzUHPsQ/oJc2EHSbZHa==", entry.Integrity)
}

func TestParseContent_ScopedPackage(t *testing.T) {
	content := []byte(`# yarn lockfile v1

"@types/node@^18.0.0":
  version "18.15.0"
  resolved "https://registry.yarnpkg.com/@types/node/-/node-18.15.0.tgz#abc"
  integrity sha512-xyz123
`)

	yarnLock, err := ParseContent(content)
	assert.NoError(t, err)
	assert.NotNil(t, yarnLock)

	entry, exists := yarnLock.Entries["@types/node@18.15.0"]
	assert.True(t, exists)
	assert.Equal(t, "@types/node", entry.Name)
	assert.Equal(t, "18.15.0", entry.Version)
}

func TestParseContent_WithDependencies(t *testing.T) {
	content := []byte(`# yarn lockfile v1

express@^4.18.2:
  version "4.18.2"
  resolved "https://registry.yarnpkg.com/express/-/express-4.18.2.tgz"
  integrity sha512-abc
  dependencies:
    accepts "~1.3.8"
    body-parser "~1.20.0"
`)

	yarnLock, err := ParseContent(content)
	assert.NoError(t, err)

	entry, exists := yarnLock.Entries["express@4.18.2"]
	assert.True(t, exists)
	assert.Equal(t, 2, len(entry.Dependencies))
	assert.Equal(t, "~1.3.8", entry.Dependencies["accepts"])
	assert.Equal(t, "~1.20.0", entry.Dependencies["body-parser"])
}

func TestParseContent_WithOptionalDependencies(t *testing.T) {
	content := []byte(`# yarn lockfile v1

chokidar@^3.5.2:
  version "3.6.0"
  resolved "https://registry.yarnpkg.com/chokidar/-/chokidar-3.6.0.tgz"
  integrity sha512-abc
  dependencies:
    anymatch "~3.1.2"
  optionalDependencies:
    fsevents "~2.3.2"
`)

	yarnLock, err := ParseContent(content)
	assert.NoError(t, err)

	entry, exists := yarnLock.Entries["chokidar@3.6.0"]
	assert.True(t, exists)
	assert.Equal(t, 1, len(entry.Dependencies))
	assert.Equal(t, "~3.1.2", entry.Dependencies["anymatch"])
	assert.Equal(t, 1, len(entry.OptionalDependencies))
	assert.Equal(t, "~2.3.2", entry.OptionalDependencies["fsevents"])
}

func TestParseContent_MultipleVersionSelectors(t *testing.T) {
	content := []byte(`# yarn lockfile v1

debug@2.6.9, debug@^2.6.0:
  version "2.6.9"
  resolved "https://registry.yarnpkg.com/debug/-/debug-2.6.9.tgz"
  integrity sha512-abc
`)

	yarnLock, err := ParseContent(content)
	assert.NoError(t, err)

	// Should be deduplicated to single entry
	entry, exists := yarnLock.Entries["debug@2.6.9"]
	assert.True(t, exists)
	assert.Equal(t, "debug", entry.Name)
	assert.Equal(t, "2.6.9", entry.Version)
}

func TestParseContent_MultipleEntries(t *testing.T) {
	content := []byte(`# yarn lockfile v1

accepts@~1.3.8:
  version "1.3.8"
  resolved "https://registry.yarnpkg.com/accepts/-/accepts-1.3.8.tgz"
  integrity sha512-abc

express@^4.18.2:
  version "4.18.2"
  resolved "https://registry.yarnpkg.com/express/-/express-4.18.2.tgz"
  integrity sha512-xyz
`)

	yarnLock, err := ParseContent(content)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(yarnLock.Entries))

	_, hasAccepts := yarnLock.Entries["accepts@1.3.8"]
	_, hasExpress := yarnLock.Entries["express@4.18.2"]
	assert.True(t, hasAccepts)
	assert.True(t, hasExpress)
}

func TestParseContent_InvalidFormat(t *testing.T) {
	content := []byte(`not a valid yarn lock file`)
	_, err := ParseContent(content)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only v1 is supported")
}

func TestExtractPackageName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"express@^4.18.2", "express"},
		{"express@4.18.2", "express"},
		{"@types/node@^18.0.0", "@types/node"},
		{"@babel/core@7.0.0", "@babel/core"},
		{"lodash", "lodash"},
		{"@scope/pkg", "@scope/pkg"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractPackageName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEntryByName(t *testing.T) {
	content := []byte(`# yarn lockfile v1

express@^4.18.2:
  version "4.18.2"
  resolved "https://registry.yarnpkg.com/express/-/express-4.18.2.tgz"
  integrity sha512-abc
`)

	yarnLock, err := ParseContent(content)
	assert.NoError(t, err)

	entry, found := yarnLock.GetEntryByName("express")
	assert.True(t, found)
	assert.Equal(t, "express", entry.Name)

	_, found = yarnLock.GetEntryByName("nonexistent")
	assert.False(t, found)
}

func TestGetAllPackageNames(t *testing.T) {
	content := []byte(`# yarn lockfile v1

accepts@~1.3.8:
  version "1.3.8"
  resolved "https://registry.yarnpkg.com/accepts/-/accepts-1.3.8.tgz"
  integrity sha512-abc

express@^4.18.2:
  version "4.18.2"
  resolved "https://registry.yarnpkg.com/express/-/express-4.18.2.tgz"
  integrity sha512-xyz
`)

	yarnLock, err := ParseContent(content)
	assert.NoError(t, err)

	names := yarnLock.GetAllPackageNames()
	assert.Equal(t, 2, len(names))
	assert.Contains(t, names, "accepts")
	assert.Contains(t, names, "express")
}

func TestParse_RealYarnLock(t *testing.T) {
	// Test with the real yarn.lock file from tests/express-yarn
	yarnLockPath := filepath.Join("..", "tests", "express-yarn", "yarn.lock")

	// Skip if file doesn't exist
	if _, err := os.Stat(yarnLockPath); os.IsNotExist(err) {
		t.Skip("tests/express-yarn/yarn.lock not found")
	}

	yarnLock, err := Parse(yarnLockPath)
	assert.NoError(t, err)
	if !assert.NotNil(t, yarnLock) {
		return // Stop test if parsing failed
	}

	// Should have multiple entries
	assert.Greater(t, len(yarnLock.Entries), 0)

	// Check for express entry
	entry, found := yarnLock.GetEntryByName("express")
	assert.True(t, found)
	if !found {
		return // Stop test if entry not found
	}
	assert.Equal(t, "express", entry.Name)
	assert.NotEmpty(t, entry.Version)
	assert.NotEmpty(t, entry.Resolved)
	assert.NotEmpty(t, entry.Integrity)

	// Express should have dependencies
	assert.Greater(t, len(entry.Dependencies), 0)
}
