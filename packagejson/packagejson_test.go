package packagejson

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ernesto27/go-npm/config"

	"github.com/stretchr/testify/assert"
)

func TestPackageJSONParser_Parse(t *testing.T) {
	testCases := []struct {
		name        string
		setupFile   func(t *testing.T) string
		expectError bool
		validate    func(t *testing.T, result *PackageJSON)
	}{
		{
			name: "Valid basic package.json",
			setupFile: func(t *testing.T) string {
				tmpDir := t.TempDir()
				tmpFile := filepath.Join(tmpDir, "package.json")

				packageData := PackageJSON{
					Name:        "test-package",
					Description: "A test package",
					Version:     "1.2.3",
					Author:      "Test Author",
					License:     "MIT",
					Homepage:    "https://example.com",
					Keywords:    []string{"test", "example"},
					Dependencies: map[string]string{
						"express": "^4.18.0",
						"lodash":  "^4.17.21",
					},
					Scripts: map[string]string{
						"start": "node index.js",
						"test":  "jest",
					},
					Main:    "index.js",
					Types:   "index.d.ts",
					Private: false,
				}

				data, _ := json.MarshalIndent(packageData, "", "  ")
				os.WriteFile(tmpFile, data, 0644)
				return tmpDir
			},
			expectError: false,
			validate: func(t *testing.T, result *PackageJSON) {
				assert.Equal(t, "test-package", result.Name)
				assert.Equal(t, "1.2.3", result.Version)
				assert.Equal(t, "A test package", result.Description)
				assert.Equal(t, "MIT", result.License)
				assert.Equal(t, map[string]string{
					"express": "^4.18.0",
					"lodash":  "^4.17.21",
				}, result.GetDependencies())
				assert.Equal(t, map[string]string{
					"start": "node index.js",
					"test":  "jest",
				}, result.Scripts)
			},
		},
		{
			name: "Legacy format with array dependencies",
			setupFile: func(t *testing.T) string {
				tmpDir := t.TempDir()
				tmpFile := filepath.Join(tmpDir, "package.json")

				legacyJSON := []byte(`{
					"name": "JSV",
					"version": "4.0.2",
					"description": "A JavaScript implementation of a extendable, fully compliant JSON Schema validator.",
					"dependencies": [],
					"main": "lib/jsv.js"
				}`)

				os.WriteFile(tmpFile, legacyJSON, 0644)
				return tmpDir
			},
			expectError: false,
			validate: func(t *testing.T, result *PackageJSON) {
				assert.Equal(t, "JSV", result.Name)
				assert.Equal(t, "4.0.2", result.Version)
				// Array dependencies should be converted to empty map
				deps := result.GetDependencies()
				assert.NotNil(t, deps)
				assert.Equal(t, 0, len(deps))
			},
		},
		{
			name: "Non-existent file",
			setupFile: func(t *testing.T) string {
				return t.TempDir()
			},
			expectError: true,
			validate: func(t *testing.T, result *PackageJSON) {
				assert.Nil(t, result)
			},
		},
		{
			name: "Invalid JSON",
			setupFile: func(t *testing.T) string {
				tmpDir := t.TempDir()
				tmpFile := filepath.Join(tmpDir, "package.json")

				invalidJSON := []byte(`{
					"name": "test",
					"version": "1.0.0",
					"invalid":
				}`)

				os.WriteFile(tmpFile, invalidJSON, 0644)
				return tmpDir
			},
			expectError: true,
			validate: func(t *testing.T, result *PackageJSON) {
				assert.Nil(t, result)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := tc.setupFile(t)

			// Save current directory
			originalDir, err := os.Getwd()
			assert.NoError(t, err)
			defer os.Chdir(originalDir)

			// Change to temp directory
			err = os.Chdir(tmpDir)
			assert.NoError(t, err)

			cfg, err := config.New()
			assert.NoError(t, err)

			parser := NewPackageJSONParser(cfg)
			result, err := parser.Parse("package.json")

			if tc.expectError {
				assert.Error(t, err, "Expected an error")
			} else {
				assert.NoError(t, err, "Expected no error")
			}

			tc.validate(t, result)
		})
	}
}

func TestPackageJSONParser_MigrateFromPackageLock(t *testing.T) {
	testCases := []struct {
		name        string
		setupFunc   func(t *testing.T) string
		expectError bool
		validate    func(t *testing.T, parser *PackageJSONParser)
	}{
		{
			name: "Successful migration with root package dependencies",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()

				// Create a package-lock.json with root package (empty key)
				packageLockData := PackageLock{
					Name:            "test-project",
					Version:         "1.0.0",
					LockfileVersion: 2,
					Requires:        true,
					Packages: map[string]PackageItem{
						"": {
							Name:    "test-project",
							Version: "1.0.0",
							Dependencies: map[string]string{
								"express": "^4.18.0",
								"lodash":  "^4.17.21",
							},
							DevDependencies: map[string]string{
								"jest": "^29.0.0",
							},
						},
						"node_modules/express": {
							Version:  "4.18.2",
							Resolved: "https://registry.npmjs.org/express/-/express-4.18.2.tgz",
						},
						"node_modules/lodash": {
							Version:  "4.17.21",
							Resolved: "https://registry.npmjs.org/lodash/-/lodash-4.17.21.tgz",
						},
						"node_modules/jest": {
							Version:  "29.5.0",
							Resolved: "https://registry.npmjs.org/jest/-/jest-29.5.0.tgz",
							Dev:      true,
						},
					},
				}

				data, _ := json.MarshalIndent(packageLockData, "", "  ")
				lockFile := filepath.Join(tmpDir, LOCK_FILE_NAME_NPM)
				os.WriteFile(lockFile, data, 0644)

				return tmpDir
			},
			expectError: false,
			validate: func(t *testing.T, parser *PackageJSONParser) {
				// Verify the migration created the go-npm lock file (in current temp directory)
				assert.FileExists(t, LOCK_FILE_NAME_GO_NPM)

				// Verify the PackageLock was updated correctly
				assert.NotNil(t, parser.PackageLock)
				assert.Equal(t, "test-project", parser.PackageLock.Name)
				assert.Equal(t, "1.0.0", parser.PackageLock.Version)

				// Verify dependencies were extracted from root package
				assert.Equal(t, map[string]string{
					"express": "^4.18.0",
					"lodash":  "^4.17.21",
				}, parser.PackageLock.Dependencies)

				assert.Equal(t, map[string]string{
					"jest": "^29.0.0",
				}, parser.PackageLock.DevDependencies)

				// Verify root package (empty key) was removed
				_, exists := parser.PackageLock.Packages[""]
				assert.False(t, exists, "Root package with empty key should be removed")

				// Verify other packages are still present
				assert.Contains(t, parser.PackageLock.Packages, "node_modules/express")
				assert.Contains(t, parser.PackageLock.Packages, "node_modules/lodash")
				assert.Contains(t, parser.PackageLock.Packages, "node_modules/jest")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := tc.setupFunc(t)

			// Save current directory
			originalDir, err := os.Getwd()
			assert.NoError(t, err)
			defer os.Chdir(originalDir)

			// Change to temp directory
			err = os.Chdir(tmpDir)
			assert.NoError(t, err)

			cfg, err := config.New()
			assert.NoError(t, err)

			parser := NewPackageJSONParser(cfg)
			err = parser.MigrateFromPackageLock()

			if tc.expectError {
				assert.Error(t, err, "Expected an error")
			} else {
				assert.NoError(t, err, "Expected no error")
			}

			tc.validate(t, parser)
		})
	}
}
