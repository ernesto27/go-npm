package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ernesto27/go-npm/packagejson"
	"github.com/ernesto27/go-npm/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddCLI(t *testing.T) {
	projectRoot, err := filepath.Abs("..")
	require.NoError(t, err)
	binaryPath := utils.BuildTestBinary(t, projectRoot)

	testCases := []struct {
		name        string
		setupFunc   func(t *testing.T, testDir string)
		args        []string
		expectError bool
		validate    func(t *testing.T, testDir string, cacheDir string, output string)
	}{
		{
			name: "successfully adds package",
			setupFunc: func(t *testing.T, testDir string) {
				packageJSON := `{
					"name": "test-project",
					"version": "1.0.0",
					"dependencies": {}
				}`
				err := os.WriteFile(filepath.Join(testDir, "package.json"), []byte(packageJSON), 0644)
				require.NoError(t, err)

				lockFile := `{
					"name": "test-project",
					"version": "1.0.0",
					"lockfileVersion": 3,
					"requires": true,
					"packages": {},
					"dependencies": {}
				}`
				err = os.WriteFile(filepath.Join(testDir, "go-npm-lock.json"), []byte(lockFile), 0644)
				require.NoError(t, err)
			},
			args:        []string{"add", "is-odd"},
			expectError: false,
			validate: func(t *testing.T, testDir string, cacheDir string, output string) {
				// Validate node_modules contains the package
				assert.DirExists(t, filepath.Join(testDir, "node_modules", "is-odd"),
					"is-odd should be installed in node_modules")

				// Validate package.json was updated
				pkgJSONContent, err := os.ReadFile(filepath.Join(testDir, "package.json"))
				require.NoError(t, err)
				var pkgJSON packagejson.PackageJSON
				err = json.Unmarshal(pkgJSONContent, &pkgJSON)
				require.NoError(t, err)

				deps := pkgJSON.GetDependencies()
				_, exists := deps["is-odd"]
				assert.True(t, exists, "is-odd should be in package.json dependencies")

				// Validate lock file was created/updated
				lockContent, err := os.ReadFile(filepath.Join(testDir, "go-npm-lock.json"))
				require.NoError(t, err)
				var lockFile packagejson.PackageLock
				err = json.Unmarshal(lockContent, &lockFile)
				require.NoError(t, err)

				_, exists = lockFile.Dependencies["is-odd"]
				assert.True(t, exists, "is-odd should be in lock file dependencies")

				_, exists = lockFile.Packages["node_modules/is-odd"]
				assert.True(t, exists, "lock file should contain node_modules/is-odd entry")

				// Validate sub-dependencies are also installed
				assert.DirExists(t, filepath.Join(testDir, "node_modules", "is-number"),
					"is-number (dependency of is-odd) should be installed in node_modules")
			},
		},
		{
			name: "successfully adds package with specific version",
			setupFunc: func(t *testing.T, testDir string) {
				packageJSON := `{
					"name": "test-project",
					"version": "1.0.0",
					"dependencies": {}
				}`
				err := os.WriteFile(filepath.Join(testDir, "package.json"), []byte(packageJSON), 0644)
				require.NoError(t, err)

				lockFile := `{
					"name": "test-project",
					"version": "1.0.0",
					"lockfileVersion": 3,
					"requires": true,
					"packages": {},
					"dependencies": {}
				}`
				err = os.WriteFile(filepath.Join(testDir, "go-npm-lock.json"), []byte(lockFile), 0644)
				require.NoError(t, err)
			},
			args:        []string{"add", "is-odd@3.0.1"},
			expectError: false,
			validate: func(t *testing.T, testDir string, cacheDir string, output string) {
				// Validate node_modules contains the package with correct version
				assert.DirExists(t, filepath.Join(testDir, "node_modules", "is-odd"),
					"is-odd should be installed in node_modules")

				installedPkgJSON, err := os.ReadFile(filepath.Join(testDir, "node_modules", "is-odd", "package.json"))
				require.NoError(t, err)
				assert.Contains(t, string(installedPkgJSON), `"version": "3.0.1"`,
					"installed package should be version 3.0.1")

				// Validate package.json was updated with the specific version
				mainPkgJSONContent, err := os.ReadFile(filepath.Join(testDir, "package.json"))
				require.NoError(t, err)
				var mainPkgJSON packagejson.PackageJSON
				err = json.Unmarshal(mainPkgJSONContent, &mainPkgJSON)
				require.NoError(t, err)

				deps := mainPkgJSON.GetDependencies()
				version, exists := deps["is-odd"]
				assert.True(t, exists, "is-odd should be in package.json dependencies")
				assert.Equal(t, "3.0.1", version, "package.json should specify version 3.0.1")

				// Validate lock file was updated with exact version
				lockContent, err := os.ReadFile(filepath.Join(testDir, "go-npm-lock.json"))
				require.NoError(t, err)
				var lockFile packagejson.PackageLock
				err = json.Unmarshal(lockContent, &lockFile)
				require.NoError(t, err)

				isOddPkg, exists := lockFile.Packages["node_modules/is-odd"]
				assert.True(t, exists, "lock file should contain node_modules/is-odd entry")
				assert.Equal(t, "3.0.1", isOddPkg.Version, "lock file should have version 3.0.1")
			},
		},
		{
			name: "adds package to existing dependencies",
			setupFunc: func(t *testing.T, testDir string) {
				packageJSON := `{
					"name": "test-project",
					"version": "1.0.0",
					"dependencies": {
						"lodash": "^4.17.21"
					}
				}`
				err := os.WriteFile(filepath.Join(testDir, "package.json"), []byte(packageJSON), 0644)
				require.NoError(t, err)

				lockFile := `{
					"name": "test-project",
					"version": "1.0.0",
					"lockfileVersion": 3,
					"requires": true,
					"packages": {
						"node_modules/lodash": {
							"version": "4.17.21"
						}
					},
					"dependencies": {
						"lodash": "^4.17.21"
					}
				}`
				err = os.WriteFile(filepath.Join(testDir, "go-npm-lock.json"), []byte(lockFile), 0644)
				require.NoError(t, err)
			},
			args:        []string{"add", "is-odd@3.0.1"},
			expectError: false,
			validate: func(t *testing.T, testDir string, cacheDir string, output string) {
				// Validate both packages are in node_modules
				assert.DirExists(t, filepath.Join(testDir, "node_modules", "is-odd"),
					"is-odd should be installed in node_modules")

				// Validate package.json has both dependencies
				mainPkgJSONContent, err := os.ReadFile(filepath.Join(testDir, "package.json"))
				require.NoError(t, err)
				var mainPkgJSON packagejson.PackageJSON
				err = json.Unmarshal(mainPkgJSONContent, &mainPkgJSON)
				require.NoError(t, err)

				deps := mainPkgJSON.GetDependencies()
				_, lodashExists := deps["lodash"]
				assert.True(t, lodashExists, "lodash should still be in package.json dependencies")

				isOddVersion, isOddExists := deps["is-odd"]
				assert.True(t, isOddExists, "is-odd should be in package.json dependencies")
				assert.Equal(t, "3.0.1", isOddVersion, "is-odd should have version 3.0.1")

				// Validate lock file has both packages
				lockContent, err := os.ReadFile(filepath.Join(testDir, "go-npm-lock.json"))
				require.NoError(t, err)
				var lockFile packagejson.PackageLock
				err = json.Unmarshal(lockContent, &lockFile)
				require.NoError(t, err)

				_, lodashInLock := lockFile.Packages["node_modules/lodash"]
				assert.True(t, lodashInLock, "lodash should be in lock file packages")

				isOddPkg, isOddInLock := lockFile.Packages["node_modules/is-odd"]
				assert.True(t, isOddInLock, "is-odd should be in lock file packages")
				assert.Equal(t, "3.0.1", isOddPkg.Version, "is-odd should have version 3.0.1 in lock file")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testDir := t.TempDir()

			tc.setupFunc(t, testDir)

			output, err, cacheDir := utils.RunWithIsolatedCache(t, binaryPath, testDir, tc.args...)

			t.Logf("CLI output:\n%s", string(output))

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err, "command failed with output: %s", string(output))
			}

			if tc.validate != nil {
				tc.validate(t, testDir, cacheDir, string(output))
			}
		})
	}
}
