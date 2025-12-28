package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ernesto27/go-npm/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCLI(t *testing.T) {
	projectRoot, err := filepath.Abs("..")
	require.NoError(t, err)
	binaryPath := utils.BuildTestBinary(t, projectRoot)

	testCases := []struct {
		name        string
		setupFunc   func(t *testing.T, testDir string)
		args        []string
		expectError bool
		validate    func(t *testing.T, output string)
	}{
		{
			name: "successfully runs script",
			setupFunc: func(t *testing.T, testDir string) {
				packageJSON := `{
					"name": "test-project",
					"version": "1.0.0",
					"scripts": {
						"greet": "echo 'Hello from test!'"
					}
				}`
				err := os.WriteFile(filepath.Join(testDir, "package.json"), []byte(packageJSON), 0644)
				require.NoError(t, err)
			},
			args:        []string{"run", "greet"},
			expectError: false,
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "Hello from test!")
				assert.Contains(t, output, "> test-project@1.0.0 greet")
			},
		},
		{
			name: "fails when script not found",
			setupFunc: func(t *testing.T, testDir string) {
				packageJSON := `{
					"name": "test-project",
					"version": "1.0.0",
					"scripts": {
						"build": "echo 'building'"
					}
				}`
				err := os.WriteFile(filepath.Join(testDir, "package.json"), []byte(packageJSON), 0644)
				require.NoError(t, err)
			},
			args:        []string{"run", "nonexistent"},
			expectError: true,
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "script \"nonexistent\" not found")
				assert.Contains(t, output, "build:")
			},
		},
		{
			name: "fails when no scripts defined",
			setupFunc: func(t *testing.T, testDir string) {
				packageJSON := `{
					"name": "test-project",
					"version": "1.0.0"
				}`
				err := os.WriteFile(filepath.Join(testDir, "package.json"), []byte(packageJSON), 0644)
				require.NoError(t, err)
			},
			args:        []string{"run", "test"},
			expectError: true,
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "no scripts defined")
			},
		},
		{
			name: "fails when no script name provided",
			setupFunc: func(t *testing.T, testDir string) {
				packageJSON := `{
					"name": "test-project",
					"version": "1.0.0",
					"scripts": {
						"test": "echo 'test'"
					}
				}`
				err := os.WriteFile(filepath.Join(testDir, "package.json"), []byte(packageJSON), 0644)
				require.NoError(t, err)
			},
			args:        []string{"run"},
			expectError: true,
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "accepts 1 arg")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testDir := t.TempDir()

			tc.setupFunc(t, testDir)

			output, err, _ := utils.RunWithIsolatedCache(t, binaryPath, testDir, tc.args...)

			t.Logf("CLI output:\n%s", string(output))

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err, "command failed with output: %s", string(output))
			}

			if tc.validate != nil {
				tc.validate(t, string(output))
			}
		})
	}
}
