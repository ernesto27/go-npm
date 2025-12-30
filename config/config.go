package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const NPMRegistryURL = "https://registry.npmjs.org/"

type Config struct {
	// Base directories
	BaseDir     string
	ManifestDir string
	TarballDir  string
	PackagesDir string

	// Local installation paths
	LocalNodeModules string
	LocalBinDir      string

	// Global installation paths
	GlobalDir         string
	GlobalNodeModules string
	GlobalBinDir      string
	GlobalPackageJSON string
	GlobalLockFile    string
}

func New() (*Config, error) {
	// Allow overriding base directory via environment variable (useful for testing)
	baseDir := os.Getenv("GO_NPM_HOME")
	if baseDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		baseDir = filepath.Join(homeDir, ".config", "go-npm")
	}
	globalDir := filepath.Join(baseDir, "global")

	cfg := &Config{
		BaseDir:     baseDir,
		ManifestDir: filepath.Join(baseDir, "manifest"),
		TarballDir:  filepath.Join(baseDir, "tarball"),
		PackagesDir: filepath.Join(baseDir, "packages"),

		LocalNodeModules: "./node_modules",
		LocalBinDir:      "./node_modules/.bin",

		GlobalDir:         globalDir,
		GlobalNodeModules: filepath.Join(globalDir, "node_modules"),
		GlobalBinDir:      filepath.Join(globalDir, "bin"),
		GlobalPackageJSON: filepath.Join(globalDir, "package.json"),
		GlobalLockFile:    filepath.Join(globalDir, "go-package-lock.json"),
	}

	if err := cfg.EnsureDirectories(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) EnsureDirectories() error {
	dirs := []string{
		c.BaseDir,
		c.ManifestDir,
		c.TarballDir,
		c.PackagesDir,
		c.GlobalDir,

		filepath.Join(c.BaseDir, "etag"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

func (c *Config) ClearCache() error {
	cacheDirs := []string{
		c.ManifestDir,
		c.PackagesDir,
		c.TarballDir,
		filepath.Join(c.BaseDir, "etag"),
	}

	for _, dir := range cacheDirs {
		if err := os.RemoveAll(dir); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove %s: %w", dir, err)
			}
		}
	}

	return nil
}
