package manager

import (
	"github.com/ernesto27/go-npm/binlink"
	"github.com/ernesto27/go-npm/config"
	"github.com/ernesto27/go-npm/etag"
	"github.com/ernesto27/go-npm/extractor"
	"github.com/ernesto27/go-npm/manifest"
	"github.com/ernesto27/go-npm/packagecopy"
	"github.com/ernesto27/go-npm/packagejson"
	"github.com/ernesto27/go-npm/tarball"
	"github.com/ernesto27/go-npm/utils"
	"github.com/ernesto27/go-npm/version"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// createMockDependencies creates a Dependencies struct with mock/test instances
func createMockDependencies(t *testing.T, baseDir string) *Dependencies {
	t.Helper()

	cfg, err := config.New()
	if err != nil {
		t.Fatalf("failed to create config: %v", err)
	}

	manifestInst, err := manifest.NewManifest(baseDir, npmRegistryURL)
	if err != nil {
		t.Fatalf("failed to create manifest: %v", err)
	}

	etagInst, err := etag.NewEtag(baseDir)
	if err != nil {
		t.Fatalf("failed to create etag: %v", err)
	}

	return &Dependencies{
		Config:            cfg,
		Manifest:          manifestInst,
		Etag:              etagInst,
		Tarball:           tarball.NewTarball(cfg.TarballDir),
		Extractor:         extractor.NewTGZExtractor(),
		PackageCopy:       packagecopy.NewPackageCopy(),
		ParseJsonManifest: newParseJsonManifest(),
		VersionInfo:       version.New(),
		PackageJsonParse:  packagejson.NewPackageJSONParser(cfg),
		BinLinker:         binlink.NewBinLinker(cfg.LocalNodeModules),
	}
}

// setupTestPackageManager creates a test PackageManager with temp directory isolation
func setupTestPackageManager(t *testing.T) (*PackageManager, string, string) {
	t.Helper()

	tmpDir := t.TempDir()

	origDir, err := os.Getwd()
	assert.NoError(t, err)
	err = os.Chdir(tmpDir)
	assert.NoError(t, err)

	deps := createMockDependencies(t, tmpDir)
	pm, err := New(deps)
	assert.NoError(t, err)

	return pm, tmpDir, origDir
}

func TestNew(t *testing.T) {
	testCases := []struct {
		name        string
		setupFunc   func(t *testing.T) (*Dependencies, string)
		expectError bool
		validate    func(t *testing.T, pm *PackageManager, origDir string)
	}{
		{
			name: "successfully creates PackageManager with valid dependencies",
			setupFunc: func(t *testing.T) (*Dependencies, string) {
				t.Helper()
				tmpDir := t.TempDir()

				// Change to temp directory so config.New() works properly
				origDir, err := os.Getwd()
				assert.NoError(t, err)
				err = os.Chdir(tmpDir)
				assert.NoError(t, err)

				deps := createMockDependencies(t, tmpDir)
				return deps, origDir
			},
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager, origDir string) {
				// Verify PackageManager is not nil
				assert.NotNil(t, pm)

				// Verify all fields are initialized
				assert.NotNil(t, pm.dependencies)
				assert.NotEmpty(t, pm.extractedPath)
				assert.NotNil(t, pm.processedPackages)
				assert.NotEmpty(t, pm.configPath)
				assert.NotEmpty(t, pm.packagesPath)
				assert.NotNil(t, pm.config)
				assert.NotNil(t, pm.packages)
				assert.NotNil(t, pm.tarball)
				assert.NotNil(t, pm.extractor)
				assert.NotNil(t, pm.packageCopy)
				assert.NotNil(t, pm.manifest)
				assert.NotNil(t, pm.parseJsonManifest)
				assert.NotNil(t, pm.versionInfo)
				assert.NotNil(t, pm.packageJsonParse)
				assert.NotNil(t, pm.binLinker)
				assert.NotNil(t, pm.downloadLocks)

				// Verify maps are initialized and empty
				assert.Equal(t, 0, len(pm.dependencies))
				assert.Equal(t, 0, len(pm.processedPackages))
				assert.Equal(t, 0, len(pm.packages))
				assert.Equal(t, 0, len(pm.downloadLocks))

				// Verify boolean flags
				assert.False(t, pm.isAdd)
				assert.False(t, pm.isGlobal)

				// Verify directories were created
				assert.DirExists(t, pm.configPath)
				assert.DirExists(t, pm.packagesPath)

				// Verify extractedPath matches config.LocalNodeModules
				assert.Equal(t, pm.config.LocalNodeModules, pm.extractedPath)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			deps, origDir := tc.setupFunc(t)

			// Ensure we always restore the original directory
			defer func() {
				if origDir != "" {
					os.Chdir(origDir)
				}
			}()

			pm, err := New(deps)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, pm)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, pm)
				if tc.validate != nil {
					tc.validate(t, pm, origDir)
				}
			}
		})
	}
}

func TestSetupGlobal(t *testing.T) {
	testCases := []struct {
		name        string
		setupFunc   func(t *testing.T) (*PackageManager, string)
		expectError bool
		validate    func(t *testing.T, pm *PackageManager)
	}{
		{
			name: "successfully sets up global installation mode",
			setupFunc: func(t *testing.T) (*PackageManager, string) {
				t.Helper()
				pm, _, origDir := setupTestPackageManager(t)
				return pm, origDir
			},
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager) {
				// Verify isGlobal flag is set to true
				assert.True(t, pm.isGlobal)

				// Verify extractedPath is updated to GlobalNodeModules
				assert.Equal(t, pm.config.GlobalNodeModules, pm.extractedPath)

				// Verify global directories were created
				assert.DirExists(t, pm.config.GlobalDir)
				assert.DirExists(t, pm.config.GlobalNodeModules)
				assert.DirExists(t, pm.config.GlobalBinDir)

				// Verify binLinker was updated (non-nil check)
				assert.NotNil(t, pm.binLinker)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pm, origDir := tc.setupFunc(t)

			// Ensure we always restore the original directory
			defer func() {
				if origDir != "" {
					os.Chdir(origDir)
				}
			}()

			err := pm.SetupGlobal()

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.validate != nil {
					tc.validate(t, pm)
				}
			}
		})
	}
}

func TestRemovePackagesFromNodeModules(t *testing.T) {
	testCases := []struct {
		name        string
		setupFunc   func(t *testing.T) (*PackageManager, []string, string)
		expectError bool
		validate    func(t *testing.T, pm *PackageManager, packages []string)
	}{
		{
			name: "successfully removes single package",
			setupFunc: func(t *testing.T) (*PackageManager, []string, string) {
				t.Helper()
				pm, tmpDir, origDir := setupTestPackageManager(t)

				// Override extractedPath to explicitly use temp directory
				pm.extractedPath = filepath.Join(tmpDir, "node_modules")
				err := os.MkdirAll(pm.extractedPath, 0755)
				assert.NoError(t, err)

				// Verify we're using temp directory, not current ./node_modules
				assert.Contains(t, pm.extractedPath, tmpDir, "extractedPath should be within temp directory")

				// Create a test package directory
				pkgPath := filepath.Join(pm.extractedPath, "test-package")
				err = os.MkdirAll(pkgPath, 0755)
				assert.NoError(t, err)

				// Verify it exists
				assert.DirExists(t, pkgPath)

				return pm, []string{"test-package"}, origDir
			},
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager, packages []string) {
				// Verify the package was removed
				pkgPath := filepath.Join(pm.extractedPath, packages[0])
				assert.NoDirExists(t, pkgPath)
			},
		},
		{
			name: "successfully removes multiple packages concurrently",
			setupFunc: func(t *testing.T) (*PackageManager, []string, string) {
				t.Helper()
				pm, tmpDir, origDir := setupTestPackageManager(t)

				// Override extractedPath to explicitly use temp directory
				pm.extractedPath = filepath.Join(tmpDir, "node_modules")
				err := os.MkdirAll(pm.extractedPath, 0755)
				assert.NoError(t, err)

				// Verify we're using temp directory, not current ./node_modules
				assert.Contains(t, pm.extractedPath, tmpDir, "extractedPath should be within temp directory")

				// Create multiple test package directories
				packages := []string{"package-one", "package-two", "package-three"}
				for _, pkg := range packages {
					pkgPath := filepath.Join(pm.extractedPath, pkg)
					err = os.MkdirAll(pkgPath, 0755)
					assert.NoError(t, err)
					assert.DirExists(t, pkgPath)
				}

				return pm, packages, origDir
			},
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager, packages []string) {
				// Verify all packages were removed
				for _, pkg := range packages {
					pkgPath := filepath.Join(pm.extractedPath, pkg)
					assert.NoDirExists(t, pkgPath)
				}
			},
		},
		{
			name: "handles non-existent package gracefully",
			setupFunc: func(t *testing.T) (*PackageManager, []string, string) {
				t.Helper()
				pm, tmpDir, origDir := setupTestPackageManager(t)

				// Override extractedPath to explicitly use temp directory
				pm.extractedPath = filepath.Join(tmpDir, "node_modules")
				err := os.MkdirAll(pm.extractedPath, 0755)
				assert.NoError(t, err)

				// Verify we're using temp directory, not current ./node_modules
				assert.Contains(t, pm.extractedPath, tmpDir, "extractedPath should be within temp directory")

				return pm, []string{"non-existent-package"}, origDir
			},
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager, packages []string) {
				// No error should occur for non-existent packages
			},
		},
		{
			name: "handles empty package list",
			setupFunc: func(t *testing.T) (*PackageManager, []string, string) {
				t.Helper()
				pm, tmpDir, origDir := setupTestPackageManager(t)

				// Override extractedPath to explicitly use temp directory
				pm.extractedPath = filepath.Join(tmpDir, "node_modules")
				err := os.MkdirAll(pm.extractedPath, 0755)
				assert.NoError(t, err)

				// Verify we're using temp directory, not current ./node_modules
				assert.Contains(t, pm.extractedPath, tmpDir, "extractedPath should be within temp directory")

				return pm, []string{}, origDir
			},
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager, packages []string) {
				// Should complete successfully with no operations
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pm, packages, origDir := tc.setupFunc(t)

			// Ensure we always restore the original directory
			defer func() {
				if origDir != "" {
					os.Chdir(origDir)
				}
			}()

			err := pm.removePackagesFromNodeModules(packages)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.validate != nil {
					tc.validate(t, pm, packages)
				}
			}
		})
	}
}

func TestInstallGlobal(t *testing.T) {
	testCases := []struct {
		name        string
		setupFunc   func(t *testing.T) (*PackageManager, string)
		pkgName     string
		version     string
		expectError bool
		validate    func(t *testing.T, pm *PackageManager)
	}{
		{
			name: "returns error when not in global mode",
			setupFunc: func(t *testing.T) (*PackageManager, string) {
				t.Helper()
				pm, _, origDir := setupTestPackageManager(t)
				return pm, origDir
			},
			pkgName:     "test-package",
			version:     "1.0.0",
			expectError: true,
			validate: func(t *testing.T, pm *PackageManager) {
				// Verify isGlobal is still false
				assert.False(t, pm.isGlobal)
			},
		},
		{
			name: "successfully installs package globally to temp directory",
			setupFunc: func(t *testing.T) (*PackageManager, string) {
				t.Helper()
				pm, tmpDir, origDir := setupTestPackageManager(t)

				// Override HOME environment variable to use temp directory
				// This prevents addBinToPath() from modifying user's actual ~/.bashrc
				originalHome := os.Getenv("HOME")
				err := os.Setenv("HOME", tmpDir)
				assert.NoError(t, err)

				// Store original HOME in origDir string (reuse the return value)
				// We'll restore it in the test cleanup
				t.Cleanup(func() {
					os.Setenv("HOME", originalHome)
				})

				// Override global paths to use temp directory instead of user's home
				pm.config.GlobalDir = filepath.Join(tmpDir, ".go-npm-global")
				pm.config.GlobalNodeModules = filepath.Join(pm.config.GlobalDir, "node_modules")
				pm.config.GlobalBinDir = filepath.Join(pm.config.GlobalDir, "bin")

				// Setup global mode with overridden paths
				err = pm.SetupGlobal()
				assert.NoError(t, err)

				// Verify setup
				assert.True(t, pm.isGlobal)
				assert.Equal(t, pm.config.GlobalNodeModules, pm.extractedPath)

				return pm, origDir
			},
			pkgName:     "is-odd",
			version:     "3.0.1",
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager) {
				// Verify the package was installed in the temp global directory
				pkgPath := filepath.Join(pm.extractedPath, "is-odd")
				assert.DirExists(t, pkgPath, "package directory should exist in temp global node_modules")

				// Verify package.json exists in the installed package
				packageJSONPath := filepath.Join(pkgPath, "package.json")
				assert.FileExists(t, packageJSONPath, "package.json should exist in installed package")

				// Verify .bashrc was created in temp HOME, not actual user home
				// Note: HOME is set to tmpDir in setupFunc
				bashrcPath := filepath.Join(os.Getenv("HOME"), ".bashrc")
				assert.FileExists(t, bashrcPath, ".bashrc should exist in temp HOME directory")

				// Read and verify .bashrc content
				bashrcContent, err := os.ReadFile(bashrcPath)
				assert.NoError(t, err)
				assert.Contains(t, string(bashrcContent), pm.config.GlobalBinDir, ".bashrc should contain global bin path")
				assert.Contains(t, string(bashrcContent), "# Added by go-npm", ".bashrc should have comment")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pm, origDir := tc.setupFunc(t)

			// Ensure we always restore the original directory
			defer func() {
				if origDir != "" {
					os.Chdir(origDir)
				}
			}()

			err := pm.InstallGlobal(tc.pkgName, tc.version)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not in global mode")
			} else {
				assert.NoError(t, err)
			}

			if tc.validate != nil {
				tc.validate(t, pm)
			}
		})
	}
}

func TestParseAliasVersion(t *testing.T) {
	testCases := []struct {
		name            string
		version         string
		expectedPkg     string
		expectedVersion string
		expectedIsAlias bool
	}{
		{
			name:            "simple alias with version",
			version:         "npm:lodash@^4.17.21",
			expectedPkg:     "lodash",
			expectedVersion: "^4.17.21",
			expectedIsAlias: true,
		},
		{
			name:            "scoped package alias",
			version:         "npm:@babel/traverse@^7.25.3",
			expectedPkg:     "@babel/traverse",
			expectedVersion: "^7.25.3",
			expectedIsAlias: true,
		},
		{
			name:            "alias without version defaults to latest",
			version:         "npm:lodash",
			expectedPkg:     "lodash",
			expectedVersion: "latest",
			expectedIsAlias: true,
		},
		{
			name:            "scoped package alias without version",
			version:         "npm:@babel/core",
			expectedPkg:     "@babel/core",
			expectedVersion: "latest",
			expectedIsAlias: true,
		},
		{
			name:            "regular version not an alias",
			version:         "^4.17.21",
			expectedPkg:     "",
			expectedVersion: "^4.17.21",
			expectedIsAlias: false,
		},
		{
			name:            "exact version not an alias",
			version:         "1.0.0",
			expectedPkg:     "",
			expectedVersion: "1.0.0",
			expectedIsAlias: false,
		},
		{
			name:            "alias with exact version",
			version:         "npm:is-odd@3.0.1",
			expectedPkg:     "is-odd",
			expectedVersion: "3.0.1",
			expectedIsAlias: true,
		},
		{
			name:            "alias with tilde version",
			version:         "npm:express@~4.18.0",
			expectedPkg:     "express",
			expectedVersion: "~4.18.0",
			expectedIsAlias: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualPkg, actualVersion, isAlias := parseAliasVersion(tc.version)

			assert.Equal(t, tc.expectedPkg, actualPkg, "package name should match")
			assert.Equal(t, tc.expectedVersion, actualVersion, "version should match")
			assert.Equal(t, tc.expectedIsAlias, isAlias, "isAlias flag should match")
		})
	}
}

func TestFetchToCache(t *testing.T) {
	testCases := []struct {
		name        string
		setupFunc   func(t *testing.T) (*PackageManager, string)
		packageJSON packagejson.PackageJSON
		expectError bool
		validate    func(t *testing.T, pm *PackageManager)
	}{
		{
			name: "successfully fetches single package with no dependencies",
			setupFunc: func(t *testing.T) (*PackageManager, string) {
				t.Helper()
				pm, _, origDir := setupTestPackageManager(t)
				return pm, origDir
			},
			packageJSON: packagejson.PackageJSON{
				Dependencies: map[string]string{
					"is-odd": "3.0.1",
				},
			},
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager) {
				// Verify package was downloaded to cache (packagesPath)
				pkgPath := filepath.Join(pm.packagesPath, "is-odd@3.0.1")
				assert.DirExists(t, pkgPath, "package should be cached in packagesPath")

				// Verify package.json exists in cached package
				packageJSONPath := filepath.Join(pkgPath, "package.json")
				assert.FileExists(t, packageJSONPath, "package.json should exist in cached package")

				// Verify packageLock was created and populated
				assert.NotNil(t, pm.packageLock, "packageLock should be created")
				assert.NotNil(t, pm.packageLock.Packages, "packageLock.Packages should be initialized")
				assert.NotNil(t, pm.packageLock.Dependencies, "packageLock.Dependencies should be initialized")

				// Verify package is in packageLock
				assert.Contains(t, pm.packageLock.Dependencies, "is-odd")
				assert.Equal(t, "3.0.1", pm.packageLock.Dependencies["is-odd"])

				// Verify package entry exists in Packages
				_, exists := pm.packageLock.Packages["node_modules/is-odd"]
				assert.True(t, exists, "package should exist in packageLock.Packages")
			},
		},
		{
			name: "successfully fetches package with dependencies",
			setupFunc: func(t *testing.T) (*PackageManager, string) {
				t.Helper()
				pm, _, origDir := setupTestPackageManager(t)
				return pm, origDir
			},
			packageJSON: packagejson.PackageJSON{
				Dependencies: map[string]string{
					"is-even": "1.0.0",
				},
			},
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager) {
				// Verify main package was cached
				pkgPath := filepath.Join(pm.packagesPath, "is-even@1.0.0")
				assert.DirExists(t, pkgPath, "is-even should be cached")

				// Verify its dependency (is-odd) was also cached
				depPkgPath := filepath.Join(pm.packagesPath, "is-odd@3.0.1")
				assert.DirExists(t, depPkgPath, "is-odd dependency should be cached")

				// Verify packageLock contains both packages
				assert.Contains(t, pm.packageLock.Dependencies, "is-even")
				assert.NotNil(t, pm.packageLock.Packages["node_modules/is-even"])
			},
		},
		{
			name: "successfully handles empty dependencies",
			setupFunc: func(t *testing.T) (*PackageManager, string) {
				t.Helper()
				pm, _, origDir := setupTestPackageManager(t)
				return pm, origDir
			},
			packageJSON: packagejson.PackageJSON{
				Dependencies: map[string]string{},
			},
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager) {
				// Verify packageLock was still created
				assert.NotNil(t, pm.packageLock)
				assert.NotNil(t, pm.packageLock.Packages)
				assert.NotNil(t, pm.packageLock.Dependencies)

				// Should have no dependencies
				assert.Equal(t, 0, len(pm.packageLock.Dependencies))
			},
		},
		{
			name: "returns error for non-existent package",
			setupFunc: func(t *testing.T) (*PackageManager, string) {
				t.Helper()
				pm, _, origDir := setupTestPackageManager(t)
				return pm, origDir
			},
			packageJSON: packagejson.PackageJSON{
				Dependencies: map[string]string{
					"this-package-definitely-does-not-exist-12345": "1.0.0",
				},
			},
			expectError: true,
			validate: func(t *testing.T, pm *PackageManager) {
				// No validation needed for error case
			},
		},
		{
			name: "successfully fetches alias package",
			setupFunc: func(t *testing.T) (*PackageManager, string) {
				t.Helper()
				pm, _, origDir := setupTestPackageManager(t)
				return pm, origDir
			},
			packageJSON: packagejson.PackageJSON{
				Dependencies: map[string]string{
					"my-is-odd": "npm:is-odd@3.0.1",
				},
			},
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager) {
				// Verify actual package was downloaded to cache using real name
				pkgPath := filepath.Join(pm.packagesPath, "is-odd@3.0.1")
				assert.DirExists(t, pkgPath, "actual package (is-odd) should be cached")

				// Verify package.json exists in cached package
				packageJSONPath := filepath.Join(pkgPath, "package.json")
				assert.FileExists(t, packageJSONPath, "package.json should exist in cached package")

				// Verify packageLock contains alias name in dependencies
				assert.Contains(t, pm.packageLock.Dependencies, "my-is-odd")
				assert.Equal(t, "3.0.1", pm.packageLock.Dependencies["my-is-odd"])

				// Verify package entry exists in Packages under alias name
				_, exists := pm.packageLock.Packages["node_modules/my-is-odd"]
				assert.True(t, exists, "package should exist in packageLock.Packages under alias name")
			},
		},
		{
			name: "successfully fetches scoped package alias",
			setupFunc: func(t *testing.T) (*PackageManager, string) {
				t.Helper()
				pm, _, origDir := setupTestPackageManager(t)
				return pm, origDir
			},
			packageJSON: packagejson.PackageJSON{
				Dependencies: map[string]string{
					"babel-traverse": "npm:@babel/traverse@7.25.3",
				},
			},
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager) {
				// Verify actual scoped package was downloaded
				// Note: The package should be cached with its actual name
				pkgPath := filepath.Join(pm.packagesPath, "@babel", "traverse@7.25.3")
				assert.DirExists(t, pkgPath, "scoped package should be cached")

				// Verify packageLock contains alias name
				assert.Contains(t, pm.packageLock.Dependencies, "babel-traverse")
				assert.Equal(t, "7.25.3", pm.packageLock.Dependencies["babel-traverse"])

				// Verify package entry exists under alias name
				_, exists := pm.packageLock.Packages["node_modules/babel-traverse"]
				assert.True(t, exists, "package should exist in packageLock.Packages under alias name")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pm, origDir := tc.setupFunc(t)

			// Ensure we always restore the original directory
			defer func() {
				if origDir != "" {
					os.Chdir(origDir)
				}
			}()

			err := pm.fetchToCache(tc.packageJSON, false)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.validate != nil {
					tc.validate(t, pm)
				}
			}
		})
	}
}

func TestInstallFromCache(t *testing.T) {
	testCases := []struct {
		name        string
		setupFunc   func(t *testing.T) (*PackageManager, string)
		expectError bool
		validate    func(t *testing.T, pm *PackageManager)
	}{
		{
			name: "successfully installs single package from cache to node_modules",
			setupFunc: func(t *testing.T) (*PackageManager, string) {
				t.Helper()
				pm, _, origDir := setupTestPackageManager(t)

				// First fetch to cache to populate packageLock
				packageJSON := packagejson.PackageJSON{
					Dependencies: map[string]string{
						"is-odd": "3.0.1",
					},
				}
				err := pm.fetchToCache(packageJSON, false)
				assert.NoError(t, err)

				// Verify package is in cache but not in node_modules yet
				cachedPath := filepath.Join(pm.packagesPath, "is-odd@3.0.1")
				assert.DirExists(t, cachedPath)

				nodemodulesPath := filepath.Join(pm.extractedPath, "is-odd")
				assert.NoDirExists(t, nodemodulesPath, "package should not be in node_modules before InstallFromCache")

				return pm, origDir
			},
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager) {
				// Verify package was copied to node_modules
				nodemodulesPath := filepath.Join(pm.extractedPath, "is-odd")
				assert.DirExists(t, nodemodulesPath, "package should be in node_modules after install")

				// Verify package.json exists
				packageJSONPath := filepath.Join(nodemodulesPath, "package.json")
				assert.FileExists(t, packageJSONPath)

				// Verify .bin directory was created
				binPath := filepath.Join(pm.extractedPath, ".bin")
				assert.DirExists(t, binPath, ".bin directory should be created")
			},
		},
		{
			name: "successfully installs package with dependencies from cache",
			setupFunc: func(t *testing.T) (*PackageManager, string) {
				t.Helper()
				pm, _, origDir := setupTestPackageManager(t)

				// Fetch package with dependencies
				packageJSON := packagejson.PackageJSON{
					Dependencies: map[string]string{
						"is-even": "1.0.0",
					},
				}
				err := pm.fetchToCache(packageJSON, false)
				assert.NoError(t, err)

				return pm, origDir
			},
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager) {
				// Verify main package is in node_modules
				isEvenPath := filepath.Join(pm.extractedPath, "is-even")
				assert.DirExists(t, isEvenPath)

				// Verify dependency is also in node_modules
				isOddPath := filepath.Join(pm.extractedPath, "is-odd")
				assert.DirExists(t, isOddPath)

				// Verify both have package.json
				assert.FileExists(t, filepath.Join(isEvenPath, "package.json"))
				assert.FileExists(t, filepath.Join(isOddPath, "package.json"))
			},
		},
		{
			name: "skips packages that already exist in node_modules",
			setupFunc: func(t *testing.T) (*PackageManager, string) {
				t.Helper()
				pm, _, origDir := setupTestPackageManager(t)

				// Fetch to cache
				packageJSON := packagejson.PackageJSON{
					Dependencies: map[string]string{
						"is-odd": "3.0.1",
					},
				}
				err := pm.fetchToCache(packageJSON, false)
				assert.NoError(t, err)

				// Install once
				err = pm.InstallFromCache()
				assert.NoError(t, err)

				// Create a marker file to verify no re-copy happens
				markerPath := filepath.Join(pm.extractedPath, "is-odd", "test-marker.txt")
				err = os.WriteFile(markerPath, []byte("test"), 0644)
				assert.NoError(t, err)

				return pm, origDir
			},
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager) {
				// Verify marker file still exists (wasn't overwritten)
				markerPath := filepath.Join(pm.extractedPath, "is-odd", "test-marker.txt")
				assert.FileExists(t, markerPath, "existing packages should not be re-copied")

				content, err := os.ReadFile(markerPath)
				assert.NoError(t, err)
				assert.Equal(t, "test", string(content))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pm, origDir := tc.setupFunc(t)

			// Ensure we always restore the original directory
			defer func() {
				if origDir != "" {
					os.Chdir(origDir)
				}
			}()

			err := pm.InstallFromCache()

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.validate != nil {
					tc.validate(t, pm)
				}
			}
		})
	}
}

func TestRemove(t *testing.T) {
	testCases := []struct {
		name                  string
		setupFunc             func(t *testing.T) (*PackageManager, string)
		pkgToRemove           string
		removeFromPackageJson bool
		expectError           bool
		validate              func(t *testing.T, pm *PackageManager, tmpDir string)
	}{
		{
			name: "successfully removes package from node_modules and package.json",
			setupFunc: func(t *testing.T) (*PackageManager, string) {
				t.Helper()
				pm, tmpDir, origDir := setupTestPackageManager(t)

				// Create package.json with dependencies
				packageJSONContent := `{
  "name": "test-project",
  "version": "1.0.0",
  "dependencies": {
    "is-odd": "3.0.1",
    "is-even": "1.0.0"
  }
}`
				packageJSONPath := filepath.Join(tmpDir, "package.json")
				err := os.WriteFile(packageJSONPath, []byte(packageJSONContent), 0644)
				assert.NoError(t, err)

				// Create go-npm-lock.json (the custom lock file format)
				packageLockContent := `{
  "name": "test-project",
  "version": "1.0.0",
  "lockfileVersion": 3,
  "requires": true,
  "packages": {
    "": {
      "name": "test-project",
      "version": "1.0.0",
      "dependencies": {
        "is-odd": "3.0.1",
        "is-even": "1.0.0"
      }
    },
    "node_modules/is-odd": {
      "name": "is-odd",
      "version": "3.0.1",
      "resolved": "https://registry.npmjs.org/is-odd/-/is-odd-3.0.1.tgz",
      "etag": "test-etag"
    },
    "node_modules/is-even": {
      "name": "is-even",
      "version": "1.0.0",
      "resolved": "https://registry.npmjs.org/is-even/-/is-even-1.0.0.tgz",
      "etag": "test-etag",
      "dependencies": {
        "is-odd": "3.0.1"
      }
    }
  },
  "dependencies": {
    "is-odd": "3.0.1",
    "is-even": "1.0.0"
  }
}`
				packageLockPath := filepath.Join(tmpDir, packagejson.LOCK_FILE_NAME_GO_NPM)
				err = os.WriteFile(packageLockPath, []byte(packageLockContent), 0644)
				assert.NoError(t, err)

				// Create node_modules with packages
				nodeModulesPath := filepath.Join(tmpDir, "node_modules")
				err = os.MkdirAll(nodeModulesPath, 0755)
				assert.NoError(t, err)

				isOddPath := filepath.Join(nodeModulesPath, "is-odd")
				err = os.MkdirAll(isOddPath, 0755)
				assert.NoError(t, err)

				isEvenPath := filepath.Join(nodeModulesPath, "is-even")
				err = os.MkdirAll(isEvenPath, 0755)
				assert.NoError(t, err)

				// Reload package.json and package-lock.json into packageJsonParse
				_, err = pm.packageJsonParse.ParseDefault()
				assert.NoError(t, err)

				// Load package-lock.json
				_, err = pm.packageJsonParse.ParseLockFile()
				assert.NoError(t, err)

				return pm, origDir
			},
			pkgToRemove:           "is-even",
			removeFromPackageJson: true,
			expectError:           false,
			validate: func(t *testing.T, pm *PackageManager, tmpDir string) {
				// Verify is-even was removed from node_modules
				isEvenPath := filepath.Join(tmpDir, "node_modules", "is-even")
				assert.NoDirExists(t, isEvenPath, "is-even should be removed from node_modules")

				// Verify is-even was removed from package.json
				packageJSONContent, err := os.ReadFile(filepath.Join(tmpDir, "package.json"))
				assert.NoError(t, err)
				assert.NotContains(t, string(packageJSONContent), "is-even", "is-even should be removed from package.json")

				// Verify is-odd still exists (it's still listed in dependencies)
				isOddPath := filepath.Join(tmpDir, "node_modules", "is-odd")
				assert.DirExists(t, isOddPath, "is-odd should still exist as it's still a dependency")
			},
		},
		{
			name: "removes package from node_modules but not package.json when removeFromPackageJson is false",
			setupFunc: func(t *testing.T) (*PackageManager, string) {
				t.Helper()
				pm, tmpDir, origDir := setupTestPackageManager(t)

				// Create package.json with dependencies
				packageJSONContent := `{
  "name": "test-project",
  "version": "1.0.0",
  "dependencies": {
    "is-odd": "3.0.1"
  }
}`
				packageJSONPath := filepath.Join(tmpDir, "package.json")
				err := os.WriteFile(packageJSONPath, []byte(packageJSONContent), 0644)
				assert.NoError(t, err)

				// Create go-npm-lock.json (the custom lock file format)
				packageLockContent := `{
  "name": "test-project",
  "version": "1.0.0",
  "lockfileVersion": 3,
  "requires": true,
  "packages": {
    "": {
      "name": "test-project",
      "version": "1.0.0",
      "dependencies": {
        "is-odd": "3.0.1"
      }
    },
    "node_modules/is-odd": {
      "name": "is-odd",
      "version": "3.0.1",
      "resolved": "https://registry.npmjs.org/is-odd/-/is-odd-3.0.1.tgz",
      "etag": "test-etag"
    }
  },
  "dependencies": {
    "is-odd": "3.0.1"
  }
}`
				packageLockPath := filepath.Join(tmpDir, packagejson.LOCK_FILE_NAME_GO_NPM)
				err = os.WriteFile(packageLockPath, []byte(packageLockContent), 0644)
				assert.NoError(t, err)

				// Create node_modules with package
				nodeModulesPath := filepath.Join(tmpDir, "node_modules")
				err = os.MkdirAll(nodeModulesPath, 0755)
				assert.NoError(t, err)

				isOddPath := filepath.Join(nodeModulesPath, "is-odd")
				err = os.MkdirAll(isOddPath, 0755)
				assert.NoError(t, err)

				// Reload package.json and package-lock.json
				_, err = pm.packageJsonParse.ParseDefault()
				assert.NoError(t, err)

				// Load package-lock.json
				_, err = pm.packageJsonParse.ParseLockFile()
				assert.NoError(t, err)

				return pm, origDir
			},
			pkgToRemove:           "is-odd",
			removeFromPackageJson: false,
			expectError:           false,
			validate: func(t *testing.T, pm *PackageManager, tmpDir string) {
				// Verify is-odd was removed from node_modules
				isOddPath := filepath.Join(tmpDir, "node_modules", "is-odd")
				assert.NoDirExists(t, isOddPath, "is-odd should be removed from node_modules")

				// Verify is-odd still exists in package.json
				packageJSONContent, err := os.ReadFile(filepath.Join(tmpDir, "package.json"))
				assert.NoError(t, err)
				assert.Contains(t, string(packageJSONContent), "is-odd", "is-odd should still be in package.json")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pm, origDir := tc.setupFunc(t)

			// Ensure we always restore the original directory
			defer func() {
				if origDir != "" {
					os.Chdir(origDir)
				}
			}()

			// Get the current working directory which should be tmpDir
			cwd, err := os.Getwd()
			assert.NoError(t, err)

			err = pm.Remove(tc.pkgToRemove, tc.removeFromPackageJson)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.validate != nil {
					tc.validate(t, pm, cwd)
				}
			}
		})
	}
}

func TestAdd(t *testing.T) {
	testCases := []struct {
		name        string
		setupFunc   func(t *testing.T) (*PackageManager, string)
		pkgName     string
		version     string
		isInstall   bool
		expectError bool
		validate    func(t *testing.T, pm *PackageManager, tmpDir string)
	}{
		{
			name: "successfully adds new package to empty package.json",
			setupFunc: func(t *testing.T) (*PackageManager, string) {
				t.Helper()
				pm, tmpDir, origDir := setupTestPackageManager(t)

				// Create empty package.json
				packageJSONContent := `{
  "name": "test-project",
  "version": "1.0.0",
  "dependencies": {}
}`
				packageJSONPath := filepath.Join(tmpDir, "package.json")
				err := os.WriteFile(packageJSONPath, []byte(packageJSONContent), 0644)
				assert.NoError(t, err)

				// Parse it
				_, err = pm.packageJsonParse.ParseDefault()
				assert.NoError(t, err)

				// Create empty lock file
				lockContent := `{
  "name": "test-project",
  "version": "1.0.0",
  "lockfileVersion": 3,
  "requires": true,
  "packages": {},
  "dependencies": {}
}`
				lockPath := filepath.Join(tmpDir, packagejson.LOCK_FILE_NAME_GO_NPM)
				err = os.WriteFile(lockPath, []byte(lockContent), 0644)
				assert.NoError(t, err)

				// Parse lock file
				_, err = pm.packageJsonParse.ParseLockFile()
				assert.NoError(t, err)

				return pm, origDir
			},
			pkgName:     "is-odd",
			version:     "3.0.1",
			isInstall:   false,
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager, tmpDir string) {
				// Verify package was added to cache
				cachedPath := filepath.Join(pm.packagesPath, "is-odd@3.0.1")
				assert.DirExists(t, cachedPath, "package should be cached")

				// Verify packageLock is updated
				assert.NotNil(t, pm.packageLock)
				assert.Contains(t, pm.packageLock.Dependencies, "is-odd")
				assert.Equal(t, "3.0.1", pm.packageLock.Dependencies["is-odd"])

				// Verify go-npm-lock.json exists
				lockPath := filepath.Join(tmpDir, packagejson.LOCK_FILE_NAME_GO_NPM)
				assert.FileExists(t, lockPath, "lock file should exist")
			},
		},
		{
			name: "adds package with dependencies",
			setupFunc: func(t *testing.T) (*PackageManager, string) {
				t.Helper()
				pm, tmpDir, origDir := setupTestPackageManager(t)

				// Create empty package.json
				packageJSONContent := `{
  "name": "test-project",
  "version": "1.0.0",
  "dependencies": {}
}`
				packageJSONPath := filepath.Join(tmpDir, "package.json")
				err := os.WriteFile(packageJSONPath, []byte(packageJSONContent), 0644)
				assert.NoError(t, err)

				_, err = pm.packageJsonParse.ParseDefault()
				assert.NoError(t, err)

				// Create empty lock file
				lockContent := `{
  "name": "test-project",
  "version": "1.0.0",
  "lockfileVersion": 3,
  "requires": true,
  "packages": {},
  "dependencies": {}
}`
				lockPath := filepath.Join(tmpDir, packagejson.LOCK_FILE_NAME_GO_NPM)
				err = os.WriteFile(lockPath, []byte(lockContent), 0644)
				assert.NoError(t, err)

				_, err = pm.packageJsonParse.ParseLockFile()
				assert.NoError(t, err)

				return pm, origDir
			},
			pkgName:     "is-even",
			version:     "1.0.0",
			isInstall:   false,
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager, tmpDir string) {
				// Verify main package was cached
				cachedPath := filepath.Join(pm.packagesPath, "is-even@1.0.0")
				assert.DirExists(t, cachedPath)

				// Verify dependency was also cached
				depCachedPath := filepath.Join(pm.packagesPath, "is-odd@3.0.1")
				assert.DirExists(t, depCachedPath)

				// Verify packageLock was updated
				assert.NotNil(t, pm.packageLock)
				assert.Contains(t, pm.packageLock.Dependencies, "is-even")

				// Verify transitive dependency is in Packages (not Dependencies)
				assert.NotEmpty(t, pm.packageLock.Packages)
			},
		},
		{
			name: "skips adding package that already exists with same version when isInstall is false",
			setupFunc: func(t *testing.T) (*PackageManager, string) {
				t.Helper()
				pm, tmpDir, origDir := setupTestPackageManager(t)

				// Create package.json with existing package
				packageJSONContent := `{
  "name": "test-project",
  "version": "1.0.0",
  "dependencies": {
    "is-odd": "3.0.1"
  }
}`
				packageJSONPath := filepath.Join(tmpDir, "package.json")
				err := os.WriteFile(packageJSONPath, []byte(packageJSONContent), 0644)
				assert.NoError(t, err)

				_, err = pm.packageJsonParse.ParseDefault()
				assert.NoError(t, err)

				return pm, origDir
			},
			pkgName:     "is-odd",
			version:     "3.0.1",
			isInstall:   false,
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager, tmpDir string) {
				// Should return early without error
				// Package.json should remain unchanged
				packageJSONContent, err := os.ReadFile(filepath.Join(tmpDir, "package.json"))
				assert.NoError(t, err)
				assert.Contains(t, string(packageJSONContent), "is-odd")
			},
		},
		{
			name: "updates existing package to new version when isInstall is false",
			setupFunc: func(t *testing.T) (*PackageManager, string) {
				t.Helper()
				pm, tmpDir, origDir := setupTestPackageManager(t)

				// Create package.json with existing package at old version
				packageJSONContent := `{
  "name": "test-project",
  "version": "1.0.0",
  "dependencies": {
    "is-odd": "3.0.0"
  }
}`
				packageJSONPath := filepath.Join(tmpDir, "package.json")
				err := os.WriteFile(packageJSONPath, []byte(packageJSONContent), 0644)
				assert.NoError(t, err)

				_, err = pm.packageJsonParse.ParseDefault()
				assert.NoError(t, err)

				// Create empty lock file
				lockContent := `{
  "name": "test-project",
  "version": "1.0.0",
  "lockfileVersion": 3,
  "requires": true,
  "packages": {},
  "dependencies": {}
}`
				lockPath := filepath.Join(tmpDir, packagejson.LOCK_FILE_NAME_GO_NPM)
				err = os.WriteFile(lockPath, []byte(lockContent), 0644)
				assert.NoError(t, err)

				_, err = pm.packageJsonParse.ParseLockFile()
				assert.NoError(t, err)

				return pm, origDir
			},
			pkgName:     "is-odd",
			version:     "3.0.1",
			isInstall:   false,
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager, tmpDir string) {
				// Verify new version was cached
				cachedPath := filepath.Join(pm.packagesPath, "is-odd@3.0.1")
				assert.DirExists(t, cachedPath)

				// Verify packageLock was updated to new version
				assert.NotNil(t, pm.packageLock)
				assert.Contains(t, pm.packageLock.Dependencies, "is-odd")
				assert.Equal(t, "3.0.1", pm.packageLock.Dependencies["is-odd"], "packageLock should have new version")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pm, origDir := tc.setupFunc(t)

			// Ensure we always restore the original directory
			defer func() {
				if origDir != "" {
					os.Chdir(origDir)
				}
			}()

			// Get the current working directory
			cwd, err := os.Getwd()
			assert.NoError(t, err)

			err = pm.Add(tc.pkgName, tc.version, tc.isInstall)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.validate != nil {
					tc.validate(t, pm, cwd)
				}
			}
		})
	}
}

func TestUninstallGlobal(t *testing.T) {
	testCases := []struct {
		name        string
		setupFunc   func(t *testing.T) (*PackageManager, string, string)
		pkgToRemove string
		expectError bool
		validate    func(t *testing.T, pm *PackageManager, tmpDir string)
	}{
		{
			name: "successfully uninstalls globally installed package",
			setupFunc: func(t *testing.T) (*PackageManager, string, string) {
				t.Helper()
				pm, tmpDir, origDir := setupTestPackageManager(t)

				// Override HOME environment variable to use temp directory
				originalHome := os.Getenv("HOME")
				err := os.Setenv("HOME", tmpDir)
				assert.NoError(t, err)

				t.Cleanup(func() {
					os.Setenv("HOME", originalHome)
				})

				// Override global paths to use temp directory
				pm.config.GlobalDir = filepath.Join(tmpDir, ".go-npm-global")
				pm.config.GlobalNodeModules = filepath.Join(pm.config.GlobalDir, "node_modules")
				pm.config.GlobalBinDir = filepath.Join(pm.config.GlobalDir, "bin")
				pm.config.GlobalLockFile = filepath.Join(pm.config.GlobalDir, packagejson.LOCK_FILE_NAME_GO_NPM)

				// Setup global mode
				err = pm.SetupGlobal()
				assert.NoError(t, err)

				// Install a package globally first
				err = pm.InstallGlobal("is-odd", "3.0.1")
				assert.NoError(t, err)

				// Verify it was installed
				pkgPath := filepath.Join(pm.extractedPath, "is-odd")
				assert.DirExists(t, pkgPath, "package should be installed before uninstall test")

				return pm, tmpDir, origDir
			},
			pkgToRemove: "is-odd",
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager, tmpDir string) {
				// Verify package was removed from global node_modules
				pkgPath := filepath.Join(pm.config.GlobalNodeModules, "is-odd")
				assert.NoDirExists(t, pkgPath, "package should be removed from global node_modules")

				// Verify global lock file was updated
				lockFilePath := pm.config.GlobalLockFile
				assert.FileExists(t, lockFilePath, "global lock file should exist")

				// Read lock file and verify package entry is removed
				lockFileContent, err := os.ReadFile(lockFilePath)
				assert.NoError(t, err)

				// Lock file should not contain the removed package's node_modules entry
				assert.NotContains(t, string(lockFileContent), "node_modules/is-odd",
					"lock file should not contain removed package")

				// Verify bin directory still exists (even if empty)
				assert.DirExists(t, pm.config.GlobalBinDir, "bin directory should still exist")
			},
		},
		{
			name: "handles uninstalling non-existent global package gracefully",
			setupFunc: func(t *testing.T) (*PackageManager, string, string) {
				t.Helper()
				pm, tmpDir, origDir := setupTestPackageManager(t)

				// Override HOME environment variable
				originalHome := os.Getenv("HOME")
				err := os.Setenv("HOME", tmpDir)
				assert.NoError(t, err)

				t.Cleanup(func() {
					os.Setenv("HOME", originalHome)
				})

				// Override global paths
				pm.config.GlobalDir = filepath.Join(tmpDir, ".go-npm-global")
				pm.config.GlobalNodeModules = filepath.Join(pm.config.GlobalDir, "node_modules")
				pm.config.GlobalBinDir = filepath.Join(pm.config.GlobalDir, "bin")
				pm.config.GlobalLockFile = filepath.Join(pm.config.GlobalDir, packagejson.LOCK_FILE_NAME_GO_NPM)

				err = pm.SetupGlobal()
				assert.NoError(t, err)

				return pm, tmpDir, origDir
			},
			pkgToRemove: "non-existent-package",
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager, tmpDir string) {
				pkgPath := filepath.Join(pm.config.GlobalNodeModules, "non-existent-package")
				assert.NoDirExists(t, pkgPath)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pm, tmpDir, origDir := tc.setupFunc(t)

			defer func() {
				if origDir != "" {
					os.Chdir(origDir)
				}
			}()

			err := pm.Remove(tc.pkgToRemove, false)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.validate != nil {
					tc.validate(t, pm, tmpDir)
				}
			}
		})
	}
}

func TestFetchToCacheWithOptionalDependencies(t *testing.T) {
	testCases := []struct {
		name        string
		setupFunc   func(t *testing.T) (*PackageManager, string)
		packageJSON packagejson.PackageJSON
		expectError bool
		validate    func(t *testing.T, pm *PackageManager)
	}{
		{
			name: "successfully handles optional dependency compatible with current platform",
			setupFunc: func(t *testing.T) (*PackageManager, string) {
				t.Helper()
				pm, _, origDir := setupTestPackageManager(t)
				return pm, origDir
			},
			packageJSON: packagejson.PackageJSON{
				OptionalDependencies: map[string]string{
					"is-odd": "3.0.1",
				},
			},
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager) {
				pkgPath := filepath.Join(pm.packagesPath, "is-odd@3.0.1")
				assert.DirExists(t, pkgPath, "optional dependency should be cached")

				assert.NotNil(t, pm.packageLock)
				assert.Contains(t, pm.packageLock.OptionalDependencies, "is-odd")
				assert.Equal(t, "3.0.1", pm.packageLock.OptionalDependencies["is-odd"])

				pkgItem, exists := pm.packageLock.Packages["node_modules/is-odd"]
				assert.True(t, exists, "optional dependency should exist in packageLock.Packages")
				assert.True(t, pkgItem.Optional, "package should be marked as optional")
			},
		},
		{
			name: "skips optional dependency incompatible with current platform (darwin-only package on linux)",
			setupFunc: func(t *testing.T) (*PackageManager, string) {
				t.Helper()
				pm, _, origDir := setupTestPackageManager(t)
				return pm, origDir
			},
			packageJSON: packagejson.PackageJSON{
				OptionalDependencies: map[string]string{
					// fsevents is darwin-only - will be skipped on non-darwin platforms
					"fsevents": "2.3.2",
				},
			},
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager) {
				assert.NotNil(t, pm.packageLock)

				currentOS := utils.GetCurrentOS()
				pkgPath := filepath.Join(pm.packagesPath, "fsevents@2.3.2")

				if currentOS == "darwin" {
					assert.DirExists(t, pkgPath, "fsevents should be installed on darwin")
					assert.Contains(t, pm.packageLock.OptionalDependencies, "fsevents")
				} else {
					assert.NoDirExists(t, pkgPath, "fsevents should not be installed on non-darwin platforms")

					pkgItem, exists := pm.packageLock.Packages["node_modules/fsevents"]
					if exists {
						assert.Equal(t, "", pkgItem.Resolved, "skipped optional dependency should have empty Resolved")
						assert.True(t, pkgItem.Optional, "should be marked as optional")
						assert.NotEmpty(t, pkgItem.OS, "should have OS constraints recorded")
					}
				}
			},
		},
		{
			name: "handles platform-specific optional dependencies with CPU constraints",
			setupFunc: func(t *testing.T) (*PackageManager, string) {
				t.Helper()
				pm, _, origDir := setupTestPackageManager(t)
				return pm, origDir
			},
			packageJSON: packagejson.PackageJSON{
				OptionalDependencies: map[string]string{
					"@esbuild/linux-x64": "0.19.0",
				},
			},
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager) {
				assert.NotNil(t, pm.packageLock)

				currentOS := utils.GetCurrentOS()
				currentCPU := utils.GetCurrentCPU()

				pkgPath := filepath.Join(pm.packagesPath, "@esbuild", "linux-x64@0.19.0")

				if currentOS == "linux" && currentCPU == "x64" {
					assert.DirExists(t, pkgPath, "platform-specific package should be installed on matching platform")
				} else {
					assert.NoDirExists(t, pkgPath, "platform-specific package should not be installed on non-matching platform")

					pkgItem, exists := pm.packageLock.Packages["node_modules/@esbuild/linux-x64"]
					if exists {
						assert.True(t, pkgItem.Optional, "should be marked as optional")
					}
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pm, origDir := tc.setupFunc(t)

			defer func() {
				if origDir != "" {
					os.Chdir(origDir)
				}
			}()

			err := pm.fetchToCache(tc.packageJSON, false)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.validate != nil {
					tc.validate(t, pm)
				}
			}
		})
	}
}
