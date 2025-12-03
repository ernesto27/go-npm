package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"npm-packager/packagejson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandGlobPatterns(t *testing.T) {
	testCases := []struct {
		name        string
		setupFunc   func(t *testing.T) (patterns []string, rootDir string)
		expectError bool
		validate    func(t *testing.T, paths []string, rootDir string)
	}{
		{
			name: "Simple glob packages/*",
			setupFunc: func(t *testing.T) ([]string, string) {
				tmpDir := t.TempDir()
				require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "packages", "pkg1"), 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "packages", "pkg2"), 0755))
				return []string{"packages/*"}, tmpDir
			},
			expectError: false,
			validate: func(t *testing.T, paths []string, rootDir string) {
				assert.Len(t, paths, 2)
				pkg1Found := false
				pkg2Found := false
				for _, p := range paths {
					if filepath.Base(p) == "pkg1" {
						pkg1Found = true
					}
					if filepath.Base(p) == "pkg2" {
						pkg2Found = true
					}
				}
				assert.True(t, pkg1Found, "pkg1 should be found")
				assert.True(t, pkg2Found, "pkg2 should be found")
			},
		},
		{
			name: "Multiple glob patterns",
			setupFunc: func(t *testing.T) ([]string, string) {
				tmpDir := t.TempDir()
				require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "packages", "pkg1"), 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "apps", "app1"), 0755))
				return []string{"packages/*", "apps/*"}, tmpDir
			},
			expectError: false,
			validate: func(t *testing.T, paths []string, rootDir string) {
				assert.Len(t, paths, 2)
			},
		},
		{
			name: "No matches",
			setupFunc: func(t *testing.T) ([]string, string) {
				tmpDir := t.TempDir()
				return []string{"packages/*"}, tmpDir
			},
			expectError: false,
			validate: func(t *testing.T, paths []string, rootDir string) {
				assert.Len(t, paths, 0)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patterns, rootDir := tc.setupFunc(t)
			parser := packagejson.NewPackageJSONParser(nil)
			registry := NewWorkspaceRegistry(rootDir, parser)

			paths, err := registry.expandGlobPatterns(patterns)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.validate != nil {
					tc.validate(t, paths, rootDir)
				}
			}
		})
	}
}

func TestDiscover(t *testing.T) {
	testCases := []struct {
		name        string
		setupFunc   func(t *testing.T) (rootPackageJSON *packagejson.PackageJSON, rootDir string)
		expectError bool
		validate    func(t *testing.T, registry *WorkspaceRegistry)
	}{
		{
			name: "Simple workspace structure",
			setupFunc: func(t *testing.T) (*packagejson.PackageJSON, string) {
				tmpDir := t.TempDir()

				rootPkg := &packagejson.PackageJSON{
					Name:       "root",
					Version:    "1.0.0",
					Workspaces: []any{"packages/*"},
				}

				createWorkspacePackage(t, tmpDir, "packages/utils", "@workspace/utils", "1.0.0")
				createWorkspacePackage(t, tmpDir, "packages/core", "@workspace/core", "2.0.0")

				return rootPkg, tmpDir
			},
			expectError: false,
			validate: func(t *testing.T, registry *WorkspaceRegistry) {
				assert.Len(t, registry.Packages, 2)
				assert.True(t, registry.IsWorkspacePackage("@workspace/utils"))
				assert.True(t, registry.IsWorkspacePackage("@workspace/core"))

				utils, ok := registry.GetWorkspacePackage("@workspace/utils")
				assert.True(t, ok)
				assert.Equal(t, "1.0.0", utils.Version)

				core, ok := registry.GetWorkspacePackage("@workspace/core")
				assert.True(t, ok)
				assert.Equal(t, "2.0.0", core.Version)
			},
		},
		{
			name: "Multiple glob patterns",
			setupFunc: func(t *testing.T) (*packagejson.PackageJSON, string) {
				tmpDir := t.TempDir()

				rootPkg := &packagejson.PackageJSON{
					Name:       "root",
					Version:    "1.0.0",
					Workspaces: []any{"packages/*", "apps/*"},
				}

				createWorkspacePackage(t, tmpDir, "packages/utils", "@workspace/utils", "1.0.0")
				createWorkspacePackage(t, tmpDir, "apps/web", "web-app", "1.0.0")

				return rootPkg, tmpDir
			},
			expectError: false,
			validate: func(t *testing.T, registry *WorkspaceRegistry) {
				assert.Len(t, registry.Packages, 2)
				assert.True(t, registry.IsWorkspacePackage("@workspace/utils"))
				assert.True(t, registry.IsWorkspacePackage("web-app"))
			},
		},
		{
			name: "Workspace with dependencies",
			setupFunc: func(t *testing.T) (*packagejson.PackageJSON, string) {
				tmpDir := t.TempDir()

				rootPkg := &packagejson.PackageJSON{
					Name:       "root",
					Version:    "1.0.0",
					Workspaces: []any{"packages/*"},
				}

				createWorkspacePackage(t, tmpDir, "packages/utils", "@workspace/utils", "1.0.0")

				corePath := filepath.Join(tmpDir, "packages", "core")
				require.NoError(t, os.MkdirAll(corePath, 0755))

				corePkg := map[string]interface{}{
					"name":    "@workspace/core",
					"version": "2.0.0",
					"dependencies": map[string]interface{}{
						"@workspace/utils": "^1.0.0",
					},
				}
				corePkgJSON, _ := json.MarshalIndent(corePkg, "", "  ")
				require.NoError(t, os.WriteFile(filepath.Join(corePath, "package.json"), corePkgJSON, 0644))

				return rootPkg, tmpDir
			},
			expectError: false,
			validate: func(t *testing.T, registry *WorkspaceRegistry) {
				assert.Len(t, registry.Packages, 2)

				core, ok := registry.GetWorkspacePackage("@workspace/core")
				assert.True(t, ok)
				deps := core.PackageJSON.GetDependencies()
				assert.Contains(t, deps, "@workspace/utils")
			},
		},
		{
			name: "Missing package.json name field",
			setupFunc: func(t *testing.T) (*packagejson.PackageJSON, string) {
				tmpDir := t.TempDir()

				rootPkg := &packagejson.PackageJSON{
					Name:       "root",
					Version:    "1.0.0",
					Workspaces: []any{"packages/*"},
				}

				pkgPath := filepath.Join(tmpDir, "packages", "invalid")
				require.NoError(t, os.MkdirAll(pkgPath, 0755))
				invalidPkg := map[string]interface{}{
					"version": "1.0.0",
				}
				pkgJSON, _ := json.MarshalIndent(invalidPkg, "", "  ")
				require.NoError(t, os.WriteFile(filepath.Join(pkgPath, "package.json"), pkgJSON, 0644))

				return rootPkg, tmpDir
			},
			expectError: true,
			validate:    nil,
		},
		{
			name: "No workspace patterns",
			setupFunc: func(t *testing.T) (*packagejson.PackageJSON, string) {
				tmpDir := t.TempDir()

				rootPkg := &packagejson.PackageJSON{
					Name:    "root",
					Version: "1.0.0",
				}

				return rootPkg, tmpDir
			},
			expectError: true,
			validate:    nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootPkg, rootDir := tc.setupFunc(t)
			parser := packagejson.NewPackageJSONParser(nil)
			registry := NewWorkspaceRegistry(rootDir, parser)

			err := registry.Discover(rootPkg)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.validate != nil {
					tc.validate(t, registry)
				}
			}
		})
	}
}

func TestValidate(t *testing.T) {
	testCases := []struct {
		name          string
		setupFunc     func(t *testing.T) *WorkspaceRegistry
		expectErrors  int
		errorContains string
	}{
		{
			name: "Valid registry",
			setupFunc: func(t *testing.T) *WorkspaceRegistry {
				parser := packagejson.NewPackageJSONParser(nil)
				registry := NewWorkspaceRegistry("/tmp", parser)
				registry.Packages["@workspace/utils"] = &Workspace{
					Name:    "@workspace/utils",
					Version: "1.0.0",
					Path:    "/tmp/packages/utils",
				}
				return registry
			},
			expectErrors: 0,
		},
		{
			name: "Missing name field",
			setupFunc: func(t *testing.T) *WorkspaceRegistry {
				parser := packagejson.NewPackageJSONParser(nil)
				registry := NewWorkspaceRegistry("/tmp", parser)
				registry.Packages[""] = &Workspace{
					Name:    "",
					Version: "1.0.0",
					Path:    "/tmp/packages/invalid",
				}
				return registry
			},
			expectErrors:  1,
			errorContains: "no name field",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := tc.setupFunc(t)
			errors := registry.Validate()

			assert.Len(t, errors, tc.expectErrors)
			if tc.expectErrors > 0 && tc.errorContains != "" {
				assert.Contains(t, errors[0].Error(), tc.errorContains)
			}
		})
	}
}

func TestCreateSymlink(t *testing.T) {
	testCases := []struct {
		name        string
		setupFunc   func(t *testing.T) (nodeModulesDir, packageName, workspacePath string, registry *WorkspaceRegistry)
		expectError bool
		validate    func(t *testing.T, nodeModulesDir, packageName, workspacePath string)
	}{
		{
			name: "Simple package symlink",
			setupFunc: func(t *testing.T) (string, string, string, *WorkspaceRegistry) {
				tmpDir := t.TempDir()
				nodeModulesDir := filepath.Join(tmpDir, "node_modules")
				require.NoError(t, os.MkdirAll(nodeModulesDir, 0755))

				workspacePath := filepath.Join(tmpDir, "packages", "utils")
				require.NoError(t, os.MkdirAll(workspacePath, 0755))

				parser := packagejson.NewPackageJSONParser(nil)
				registry := NewWorkspaceRegistry(tmpDir, parser)

				return nodeModulesDir, "utils", workspacePath, registry
			},
			expectError: false,
			validate: func(t *testing.T, nodeModulesDir, packageName, workspacePath string) {
				linkPath := filepath.Join(nodeModulesDir, packageName)
				info, err := os.Lstat(linkPath)
				require.NoError(t, err)
				assert.True(t, info.Mode()&os.ModeSymlink != 0, "should be a symlink")

				target, err := os.Readlink(linkPath)
				require.NoError(t, err)
				absTarget, _ := filepath.Abs(filepath.Join(filepath.Dir(linkPath), target))
				absWorkspace, _ := filepath.Abs(workspacePath)
				assert.Equal(t, absWorkspace, absTarget)
			},
		},
		{
			name: "Scoped package symlink",
			setupFunc: func(t *testing.T) (string, string, string, *WorkspaceRegistry) {
				tmpDir := t.TempDir()
				nodeModulesDir := filepath.Join(tmpDir, "node_modules")
				require.NoError(t, os.MkdirAll(nodeModulesDir, 0755))

				workspacePath := filepath.Join(tmpDir, "packages", "utils")
				require.NoError(t, os.MkdirAll(workspacePath, 0755))

				parser := packagejson.NewPackageJSONParser(nil)
				registry := NewWorkspaceRegistry(tmpDir, parser)

				return nodeModulesDir, "@workspace/utils", workspacePath, registry
			},
			expectError: false,
			validate: func(t *testing.T, nodeModulesDir, packageName, workspacePath string) {
				linkPath := filepath.Join(nodeModulesDir, packageName)
				info, err := os.Lstat(linkPath)
				require.NoError(t, err)
				assert.True(t, info.Mode()&os.ModeSymlink != 0)

				scopeDir := filepath.Join(nodeModulesDir, "@workspace")
				scopeInfo, err := os.Stat(scopeDir)
				require.NoError(t, err)
				assert.True(t, scopeInfo.IsDir(), "scope directory should exist")
			},
		},
		{
			name: "Symlink already exists and correct",
			setupFunc: func(t *testing.T) (string, string, string, *WorkspaceRegistry) {
				tmpDir := t.TempDir()
				nodeModulesDir := filepath.Join(tmpDir, "node_modules")
				require.NoError(t, os.MkdirAll(nodeModulesDir, 0755))

				workspacePath := filepath.Join(tmpDir, "packages", "utils")
				require.NoError(t, os.MkdirAll(workspacePath, 0755))

				parser := packagejson.NewPackageJSONParser(nil)
				registry := NewWorkspaceRegistry(tmpDir, parser)

				err := registry.CreateSymlink(nodeModulesDir, "utils", workspacePath)
				require.NoError(t, err)

				return nodeModulesDir, "utils", workspacePath, registry
			},
			expectError: false,
			validate: func(t *testing.T, nodeModulesDir, packageName, workspacePath string) {
				linkPath := filepath.Join(nodeModulesDir, packageName)
				info, err := os.Lstat(linkPath)
				require.NoError(t, err)
				assert.True(t, info.Mode()&os.ModeSymlink != 0)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nodeModulesDir, packageName, workspacePath, registry := tc.setupFunc(t)

			err := registry.CreateSymlink(nodeModulesDir, packageName, workspacePath)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.validate != nil {
					tc.validate(t, nodeModulesDir, packageName, workspacePath)
				}
			}
		})
	}
}

func createWorkspacePackage(t *testing.T, rootDir, relativePath, name, version string) {
	pkgPath := filepath.Join(rootDir, relativePath)
	require.NoError(t, os.MkdirAll(pkgPath, 0755))

	pkg := map[string]interface{}{
		"name":    name,
		"version": version,
	}
	pkgJSON, _ := json.MarshalIndent(pkg, "", "  ")
	require.NoError(t, os.WriteFile(filepath.Join(pkgPath, "package.json"), pkgJSON, 0644))
}
