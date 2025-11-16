package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClearCache(t *testing.T) {
	testCases := []struct {
		name        string
		setupFunc   func(t *testing.T) *Config
		expectError bool
		validate    func(t *testing.T, cfg *Config)
	}{
		{
			name: "Successfully clear cache directories",
			setupFunc: func(t *testing.T) *Config {
				tmpDir := t.TempDir()
				cfg := &Config{
					BaseDir:     tmpDir,
					ManifestDir: filepath.Join(tmpDir, "manifest"),
					PackagesDir: filepath.Join(tmpDir, "packages"),
					GlobalDir:   filepath.Join(tmpDir, "global"),
				}

				// Create cache directories with some dummy files
				assert.NoError(t, os.MkdirAll(cfg.ManifestDir, 0755))
				assert.NoError(t, os.WriteFile(filepath.Join(cfg.ManifestDir, "test.json"), []byte("test"), 0644))

				assert.NoError(t, os.MkdirAll(cfg.PackagesDir, 0755))
				assert.NoError(t, os.WriteFile(filepath.Join(cfg.PackagesDir, "package.txt"), []byte("test"), 0644))

				etagDir := filepath.Join(cfg.BaseDir, "etag")
				assert.NoError(t, os.MkdirAll(etagDir, 0755))
				assert.NoError(t, os.WriteFile(filepath.Join(etagDir, "etag.json"), []byte("test"), 0644))

				// Create global directory (should be preserved)
				assert.NoError(t, os.MkdirAll(cfg.GlobalDir, 0755))
				assert.NoError(t, os.WriteFile(filepath.Join(cfg.GlobalDir, "package.json"), []byte("{}"), 0644))

				return cfg
			},
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				// Cache directories should be removed
				_, err := os.Stat(cfg.ManifestDir)
				assert.True(t, os.IsNotExist(err), "ManifestDir should be removed")

				_, err = os.Stat(cfg.PackagesDir)
				assert.True(t, os.IsNotExist(err), "PackagesDir should be removed")

				etagDir := filepath.Join(cfg.BaseDir, "etag")
				_, err = os.Stat(etagDir)
				assert.True(t, os.IsNotExist(err), "etag directory should be removed")

				// Global directory should be preserved
				globalPkg := filepath.Join(cfg.GlobalDir, "package.json")
				_, err = os.Stat(globalPkg)
				assert.NoError(t, err, "Global directory should be preserved")
			},
		},
		{
			name: "Clear cache when directories don't exist (should not error)",
			setupFunc: func(t *testing.T) *Config {
				tmpDir := t.TempDir()
				cfg := &Config{
					BaseDir:     tmpDir,
					ManifestDir: filepath.Join(tmpDir, "manifest"),
					PackagesDir: filepath.Join(tmpDir, "packages"),
					GlobalDir:   filepath.Join(tmpDir, "global"),
				}
				// Don't create any directories - they don't exist
				return cfg
			},
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				// Should complete successfully even when directories don't exist
				_, err := os.Stat(cfg.ManifestDir)
				assert.True(t, os.IsNotExist(err), "ManifestDir should not exist")
			},
		},
		{
			name: "Preserve global directory contents",
			setupFunc: func(t *testing.T) *Config {
				tmpDir := t.TempDir()
				cfg := &Config{
					BaseDir:     tmpDir,
					ManifestDir: filepath.Join(tmpDir, "manifest"),
					PackagesDir: filepath.Join(tmpDir, "packages"),
					GlobalDir:   filepath.Join(tmpDir, "global"),
				}

				// Create cache directories
				assert.NoError(t, os.MkdirAll(cfg.ManifestDir, 0755))
				assert.NoError(t, os.MkdirAll(cfg.PackagesDir, 0755))

				// Create global directory with nested structure
				globalNodeModules := filepath.Join(cfg.GlobalDir, "node_modules")
				assert.NoError(t, os.MkdirAll(globalNodeModules, 0755))
				assert.NoError(t, os.WriteFile(filepath.Join(cfg.GlobalDir, "package.json"), []byte("{}"), 0644))
				assert.NoError(t, os.WriteFile(filepath.Join(globalNodeModules, "test.js"), []byte("console.log('test')"), 0644))

				return cfg
			},
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				// Cache directories should be removed
				_, err := os.Stat(cfg.ManifestDir)
				assert.True(t, os.IsNotExist(err), "ManifestDir should be removed")

				// Global directory and its contents should be preserved
				globalPkg := filepath.Join(cfg.GlobalDir, "package.json")
				_, err = os.Stat(globalPkg)
				assert.NoError(t, err, "Global package.json should be preserved")

				globalNodeModules := filepath.Join(cfg.GlobalDir, "node_modules", "test.js")
				_, err = os.Stat(globalNodeModules)
				assert.NoError(t, err, "Global node_modules should be preserved")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := tc.setupFunc(t)
			err := cfg.ClearCache()

			if tc.expectError {
				assert.Error(t, err, "Expected an error")
			} else {
				assert.NoError(t, err, "Expected no error")
			}

			tc.validate(t, cfg)
		})
	}
}

func TestNew(t *testing.T) {
	cfg, err := New()
	assert.NoError(t, err, "New() should not return an error")
	assert.NotNil(t, cfg, "Config should not be nil")

	// Verify all paths are set
	assert.NotEmpty(t, cfg.BaseDir, "BaseDir should be set")
	assert.NotEmpty(t, cfg.ManifestDir, "ManifestDir should be set")
	assert.NotEmpty(t, cfg.PackagesDir, "PackagesDir should be set")
	assert.NotEmpty(t, cfg.GlobalDir, "GlobalDir should be set")
	assert.NotEmpty(t, cfg.LocalNodeModules, "LocalNodeModules should be set")

	// Verify paths contain expected values
	assert.Contains(t, cfg.BaseDir, ".config/go-npm", "BaseDir should contain .config/go-npm")
	assert.Contains(t, cfg.ManifestDir, "manifest", "ManifestDir should contain manifest")
	assert.Contains(t, cfg.PackagesDir, "packages", "PackagesDir should contain packages")
	assert.Contains(t, cfg.GlobalDir, "global", "GlobalDir should contain global")
}
