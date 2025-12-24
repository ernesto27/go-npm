package manager

import (
	"github.com/ernesto27/go-npm/workspace"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFetchToCacheWithWorkspaces(t *testing.T) {
	testCases := []struct {
		name        string
		setupFunc   func(t *testing.T) (*PackageManager, string)
		expectError bool
		validate    func(t *testing.T, pm *PackageManager, tmpDir string)
	}{
		{
			name: "resolves workspace packages instead of downloading from npm",
			setupFunc: func(t *testing.T) (*PackageManager, string) {
				t.Helper()
				pm, tmpDir, origDir := setupTestPackageManager(t)

				// Create workspace structure
				packagesDir := filepath.Join(tmpDir, "packages")
				err := os.MkdirAll(packagesDir, 0755)
				assert.NoError(t, err)

				uiDir := filepath.Join(packagesDir, "ui")
				err = os.MkdirAll(uiDir, 0755)
				assert.NoError(t, err)

				uiPackageJSON := `{
  "name": "@myorg/ui",
  "version": "1.5.0",
  "dependencies": {
    "is-odd": "3.0.1"
  }
}`
				err = os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(uiPackageJSON), 0644)
				assert.NoError(t, err)

				// Root package.json depends on workspace package
				rootPackageJSON := `{
  "name": "test-app",
  "version": "1.0.0",
  "workspaces": ["packages/*"],
  "dependencies": {
    "@myorg/ui": "*"
  }
}`
				err = os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(rootPackageJSON), 0644)
				assert.NoError(t, err)

				// Parse to setup
				data, err := pm.packageJsonParse.ParseDefault()
				assert.NoError(t, err)

				// Discover workspaces before fetchToCache
				pm.workspaceRegistry = workspace.NewWorkspaceRegistry(tmpDir, pm.packageJsonParse)
				err = pm.workspaceRegistry.Discover(data)
				assert.NoError(t, err)

				return pm, origDir
			},
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager, tmpDir string) {
				// Verify workspace package was NOT downloaded to cache
				cachedPath := filepath.Join(pm.packagesPath, "@myorg", "ui@1.5.0")
				assert.NoDirExists(t, cachedPath, "workspace package should NOT be cached like npm packages")

				// Verify workspace dependency WAS downloaded
				depCachedPath := filepath.Join(pm.packagesPath, "is-odd@3.0.1")
				assert.DirExists(t, depCachedPath, "workspace package's dependency should be cached")

				// Verify packageLock has workspace entry with link: true
				assert.NotNil(t, pm.packageLock)
				pkgItem, exists := pm.packageLock.Packages["node_modules/@myorg/ui"]
				assert.True(t, exists, "workspace package should be in lock file")
				assert.True(t, pkgItem.Link, "workspace package should have link: true")
				assert.Contains(t, pkgItem.Resolved, "file:", "resolved should be file: URL")

				// Verify workspaces section in lock file
				assert.NotNil(t, pm.packageLock.Workspaces)
				assert.Contains(t, pm.packageLock.Workspaces, "@myorg/ui")
				assert.Equal(t, "1.5.0", pm.packageLock.Workspaces["@myorg/ui"])

				// Verify workspace's dependencies are tracked
				assert.NotNil(t, pkgItem.Dependencies)
				assert.Contains(t, pkgItem.Dependencies, "is-odd")
			},
		},
		{
			name: "installs dependencies for workspace packages recursively",
			setupFunc: func(t *testing.T) (*PackageManager, string) {
				t.Helper()
				pm, tmpDir, origDir := setupTestPackageManager(t)

				packagesDir := filepath.Join(tmpDir, "packages")
				err := os.MkdirAll(packagesDir, 0755)
				assert.NoError(t, err)

				uiDir := filepath.Join(packagesDir, "ui")
				err = os.MkdirAll(uiDir, 0755)
				assert.NoError(t, err)

				uiPackageJSON := `{
  "name": "workspace-ui",
  "version": "1.0.0",
  "dependencies": {
    "is-even": "1.0.0"
  }
}`
				err = os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(uiPackageJSON), 0644)
				assert.NoError(t, err)

				rootPackageJSON := `{
  "name": "test-app",
  "version": "1.0.0",
  "workspaces": ["packages/*"],
  "dependencies": {
    "workspace-ui": "*"
  }
}`
				err = os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(rootPackageJSON), 0644)
				assert.NoError(t, err)

				data, err := pm.packageJsonParse.ParseDefault()
				assert.NoError(t, err)

				pm.workspaceRegistry = workspace.NewWorkspaceRegistry(tmpDir, pm.packageJsonParse)
				err = pm.workspaceRegistry.Discover(data)
				assert.NoError(t, err)

				return pm, origDir
			},
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager, tmpDir string) {
				// Verify workspace's dependency was cached
				depPath := filepath.Join(pm.packagesPath, "is-even@1.0.0")
				assert.DirExists(t, depPath, "workspace dependency should be cached")

				// Verify transitive dependency (is-even depends on is-odd@^0.1.2)
				transitivePath := filepath.Join(pm.packagesPath, "is-odd@0.1.2")
				assert.DirExists(t, transitivePath, "transitive dependency should be cached")

				// Verify lock file tracks all dependencies
				assert.NotNil(t, pm.packageLock.Packages["node_modules/is-even"])
				assert.NotNil(t, pm.packageLock.Packages["node_modules/is-odd"])
			},
		},
		{
			name: "handles multiple workspace packages with shared dependencies",
			setupFunc: func(t *testing.T) (*PackageManager, string) {
				t.Helper()
				pm, tmpDir, origDir := setupTestPackageManager(t)

				packagesDir := filepath.Join(tmpDir, "packages")
				err := os.MkdirAll(packagesDir, 0755)
				assert.NoError(t, err)

				// Workspace 1
				pkg1Dir := filepath.Join(packagesDir, "pkg1")
				err = os.MkdirAll(pkg1Dir, 0755)
				assert.NoError(t, err)
				pkg1JSON := `{
  "name": "ws-pkg1",
  "version": "1.0.0",
  "dependencies": {
    "is-odd": "3.0.1"
  }
}`
				err = os.WriteFile(filepath.Join(pkg1Dir, "package.json"), []byte(pkg1JSON), 0644)
				assert.NoError(t, err)

				// Workspace 2 (shares is-odd dependency)
				pkg2Dir := filepath.Join(packagesDir, "pkg2")
				err = os.MkdirAll(pkg2Dir, 0755)
				assert.NoError(t, err)
				pkg2JSON := `{
  "name": "ws-pkg2",
  "version": "2.0.0",
  "dependencies": {
    "is-odd": "3.0.1"
  }
}`
				err = os.WriteFile(filepath.Join(pkg2Dir, "package.json"), []byte(pkg2JSON), 0644)
				assert.NoError(t, err)

				rootPackageJSON := `{
  "name": "test-app",
  "version": "1.0.0",
  "workspaces": ["packages/*"],
  "dependencies": {
    "ws-pkg1": "*",
    "ws-pkg2": "*"
  }
}`
				err = os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(rootPackageJSON), 0644)
				assert.NoError(t, err)

				data, err := pm.packageJsonParse.ParseDefault()
				assert.NoError(t, err)

				pm.workspaceRegistry = workspace.NewWorkspaceRegistry(tmpDir, pm.packageJsonParse)
				err = pm.workspaceRegistry.Discover(data)
				assert.NoError(t, err)

				return pm, origDir
			},
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager, tmpDir string) {
				// Verify both workspaces are in lock file
				assert.Contains(t, pm.packageLock.Workspaces, "ws-pkg1")
				assert.Contains(t, pm.packageLock.Workspaces, "ws-pkg2")

				// Verify shared dependency is cached only once
				sharedDepPath := filepath.Join(pm.packagesPath, "is-odd@3.0.1")
				assert.DirExists(t, sharedDepPath, "shared dependency should be cached")

				// Verify both workspace packages reference the dependency
				pkg1Item := pm.packageLock.Packages["node_modules/ws-pkg1"]
				assert.Contains(t, pkg1Item.Dependencies, "is-odd")

				pkg2Item := pm.packageLock.Packages["node_modules/ws-pkg2"]
				assert.Contains(t, pkg2Item.Dependencies, "is-odd")
			},
		},
		{
			name: "handles workspace package with no dependencies",
			setupFunc: func(t *testing.T) (*PackageManager, string) {
				t.Helper()
				pm, tmpDir, origDir := setupTestPackageManager(t)

				packagesDir := filepath.Join(tmpDir, "packages")
				utilsDir := filepath.Join(packagesDir, "utils")
				err := os.MkdirAll(utilsDir, 0755)
				assert.NoError(t, err)

				utilsPackageJSON := `{
  "name": "my-utils",
  "version": "2.0.0"
}`
				err = os.WriteFile(filepath.Join(utilsDir, "package.json"), []byte(utilsPackageJSON), 0644)
				assert.NoError(t, err)

				rootPackageJSON := `{
  "name": "test-app",
  "version": "1.0.0",
  "workspaces": ["packages/*"],
  "dependencies": {
    "my-utils": "*"
  }
}`
				err = os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(rootPackageJSON), 0644)
				assert.NoError(t, err)

				data, err := pm.packageJsonParse.ParseDefault()
				assert.NoError(t, err)

				pm.workspaceRegistry = workspace.NewWorkspaceRegistry(tmpDir, pm.packageJsonParse)
				err = pm.workspaceRegistry.Discover(data)
				assert.NoError(t, err)

				return pm, origDir
			},
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager, tmpDir string) {
				// Verify workspace is in lock file
				assert.NotNil(t, pm.packageLock)
				pkgItem, exists := pm.packageLock.Packages["node_modules/my-utils"]
				assert.True(t, exists)
				assert.True(t, pkgItem.Link)
				assert.Equal(t, "2.0.0", pkgItem.Version)

				// Verify no spurious dependencies were added
				assert.Nil(t, pkgItem.Dependencies)

				// Verify workspace is tracked
				assert.Contains(t, pm.packageLock.Workspaces, "my-utils")
			},
		},
		{
			name: "handles scoped workspace packages",
			setupFunc: func(t *testing.T) (*PackageManager, string) {
				t.Helper()
				pm, tmpDir, origDir := setupTestPackageManager(t)

				packagesDir := filepath.Join(tmpDir, "packages")
				uiDir := filepath.Join(packagesDir, "ui")
				err := os.MkdirAll(uiDir, 0755)
				assert.NoError(t, err)

				uiPackageJSON := `{
  "name": "@company/ui-lib",
  "version": "3.2.1",
  "dependencies": {}
}`
				err = os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(uiPackageJSON), 0644)
				assert.NoError(t, err)

				rootPackageJSON := `{
  "name": "test-app",
  "version": "1.0.0",
  "workspaces": ["packages/*"],
  "dependencies": {
    "@company/ui-lib": "*"
  }
}`
				err = os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(rootPackageJSON), 0644)
				assert.NoError(t, err)

				data, err := pm.packageJsonParse.ParseDefault()
				assert.NoError(t, err)

				pm.workspaceRegistry = workspace.NewWorkspaceRegistry(tmpDir, pm.packageJsonParse)
				err = pm.workspaceRegistry.Discover(data)
				assert.NoError(t, err)

				return pm, origDir
			},
			expectError: false,
			validate: func(t *testing.T, pm *PackageManager, tmpDir string) {
				// Verify scoped workspace package is tracked correctly
				pkgItem, exists := pm.packageLock.Packages["node_modules/@company/ui-lib"]
				assert.True(t, exists, "scoped workspace package should be in lock file")
				assert.True(t, pkgItem.Link)
				assert.Equal(t, "@company/ui-lib", pkgItem.Name)
				assert.Equal(t, "3.2.1", pkgItem.Version)

				// Verify workspaces section
				assert.Contains(t, pm.packageLock.Workspaces, "@company/ui-lib")
				assert.Equal(t, "3.2.1", pm.packageLock.Workspaces["@company/ui-lib"])

				// Verify NOT cached in packages directory
				cachedPath := filepath.Join(pm.packagesPath, "@company", "ui-lib@3.2.1")
				assert.NoDirExists(t, cachedPath, "workspace should not be cached")
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

			cwd, err := os.Getwd()
			assert.NoError(t, err)

			// Parse and fetch
			data, err := pm.packageJsonParse.ParseDefault()
			assert.NoError(t, err)

			err = pm.fetchToCache(*data, false)

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
