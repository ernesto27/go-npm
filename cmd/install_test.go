package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildBinary builds the go-npm binary for testing and returns the path to it.
// The binary is built in a temp directory to avoid polluting the project.
func buildBinary(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "go-npm-test")

	// Get project root (parent of cmd directory)
	projectRoot, err := filepath.Abs("..")
	require.NoError(t, err, "failed to get project root")

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to build binary: %s", string(output))

	return binaryPath
}

// runWithIsolatedCache runs a command with GO_NPM_HOME set to a temp directory
// to avoid polluting the user's real cache at ~/.config/go-npm
func runWithIsolatedCache(t *testing.T, binaryPath string, workDir string, args ...string) ([]byte, error, string) {
	t.Helper()

	cacheDir := t.TempDir()

	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "GO_NPM_HOME="+cacheDir, "HOME="+cacheDir)

	output, err := cmd.CombinedOutput()
	return output, err, cacheDir
}

func TestInstallCLI(t *testing.T) {
	binaryPath := buildBinary(t)

	testCases := []struct {
		name        string
		setupFunc   func(t *testing.T, testDir string)
		args        []string
		expectError bool
		validate    func(t *testing.T, testDir string, cacheDir string, output string)
	}{
		{
			name: "successfully installs dependencies from package.json",
			setupFunc: func(t *testing.T, testDir string) {
				packageJSON := `{
					"name": "test-project",
					"version": "1.0.0",
					"dependencies": {
						"is-odd": "3.0.1"
					}
				}`
				err := os.WriteFile(filepath.Join(testDir, "package.json"), []byte(packageJSON), 0644)
				require.NoError(t, err)
			},
			args:        []string{"install"},
			expectError: false,
			validate: func(t *testing.T, testDir string, cacheDir string, output string) {
				assert.DirExists(t, filepath.Join(testDir, "node_modules", "is-odd"),
					"is-odd should be installed in node_modules")
				assert.FileExists(t, filepath.Join(testDir, "go-npm-lock.json"),
					"go-npm-lock.json should be created")
			},
		},
		{
			name: "installs only production dependencies with --production flag",
			setupFunc: func(t *testing.T, testDir string) {
				packageJSON := `{
					"name": "test-project",
					"version": "1.0.0",
					"dependencies": {
						"is-odd": "3.0.1"
					},
					"devDependencies": {
						"is-even": "1.0.0"
					}
				}`
				err := os.WriteFile(filepath.Join(testDir, "package.json"), []byte(packageJSON), 0644)
				require.NoError(t, err)
			},
			args:        []string{"install", "--production"},
			expectError: false,
			validate: func(t *testing.T, testDir string, cacheDir string, output string) {
				assert.DirExists(t, filepath.Join(testDir, "node_modules", "is-odd"),
					"is-odd (production dep) should be installed")
				assert.NoDirExists(t, filepath.Join(testDir, "node_modules", "is-even"),
					"is-even (dev dep) should NOT be installed with --production")
			},
		},
		{
			name:        "fails when package.json is missing",
			setupFunc:   func(t *testing.T, testDir string) {},
			args:        []string{"install"},
			expectError: true,
			validate: func(t *testing.T, testDir string, cacheDir string, output string) {
				assert.True(t,
					strings.Contains(output, "package.json") ||
						strings.Contains(strings.ToLower(output), "no such file"),
					"error should mention package.json, got: %s", output)
			},
		},
		{
			name: "shows verbose output with --verbose flag",
			setupFunc: func(t *testing.T, testDir string) {
				packageJSON := `{
					"name": "test-project",
					"version": "1.0.0",
					"dependencies": {
						"is-odd": "3.0.1"
					}
				}`
				err := os.WriteFile(filepath.Join(testDir, "package.json"), []byte(packageJSON), 0644)
				require.NoError(t, err)
			},
			args:        []string{"install", "--verbose"},
			expectError: false,
			validate: func(t *testing.T, testDir string, cacheDir string, output string) {
				assert.DirExists(t, filepath.Join(testDir, "node_modules", "is-odd"),
					"is-odd should be installed in node_modules")
				assert.Contains(t, output, "↓ is-odd@3.0.1",
					"verbose output should show package download status")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testDir := t.TempDir()

			tc.setupFunc(t, testDir)

			output, err, cacheDir := runWithIsolatedCache(t, binaryPath, testDir, tc.args...)

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

func TestInstallCLI_Global(t *testing.T) {
	binaryPath := buildBinary(t)

	testCases := []struct {
		name        string
		args        []string
		expectError bool
		validate    func(t *testing.T, cacheDir string, output string)
	}{
		{
			name:        "fails without package name",
			args:        []string{"install", "--global"},
			expectError: true,
			validate: func(t *testing.T, cacheDir string, output string) {
				assert.Contains(t, output, "package name is required",
					"error should say package name is required")
			},
		},
		{
			name:        "successfully installs package globally",
			args:        []string{"install", "--global", "is-odd"},
			expectError: false,
			validate: func(t *testing.T, cacheDir string, output string) {
				globalNodeModules := filepath.Join(cacheDir, "global", "node_modules", "is-odd")
				assert.DirExists(t, globalNodeModules,
					"is-odd should be installed in global node_modules")
				assert.FileExists(t, filepath.Join(globalNodeModules, "package.json"),
					"package.json should exist in globally installed package")
				assert.Contains(t, output, "Successfully installed is-odd globally",
					"output should confirm successful global installation")
			},
		},
		{
			name:        "successfully installs package with specific version globally",
			args:        []string{"install", "--global", "is-odd@3.0.1"},
			expectError: false,
			validate: func(t *testing.T, cacheDir string, output string) {
				globalNodeModules := filepath.Join(cacheDir, "global", "node_modules", "is-odd")
				assert.DirExists(t, globalNodeModules,
					"is-odd@3.0.1 should be installed in global node_modules")

				packageJSONPath := filepath.Join(globalNodeModules, "package.json")
				assert.FileExists(t, packageJSONPath)

				content, err := os.ReadFile(packageJSONPath)
				require.NoError(t, err)
				assert.Contains(t, string(content), `"version": "3.0.1"`,
					"installed package should be version 3.0.1")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testDir := t.TempDir()

			output, err, cacheDir := runWithIsolatedCache(t, binaryPath, testDir, tc.args...)

			t.Logf("CLI output:\n%s", string(output))

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err, "command failed with output: %s", string(output))
			}

			if tc.validate != nil {
				tc.validate(t, cacheDir, string(output))
			}
		})
	}
}
