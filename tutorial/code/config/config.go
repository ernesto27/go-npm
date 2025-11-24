package config

import (
	"go-npm/utils"
	"os"
	"path/filepath"
)

type Config struct {
	NpmRegistryURL string

	// Base directories
	BaseDir     string
	ManifestDir string
	TarballDir  string
	PackagesDir string

	// Local installation paths
	LocalNodeModules string
	LocalBinDir      string
}

func New() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	baseDir := filepath.Join(homeDir, ".config", "go-npm")

	if err := utils.CreateDir(baseDir); err != nil {
		return nil, err
	}

	manifestPath := filepath.Join(baseDir, "manifest")
	if err := utils.CreateDir(manifestPath); err != nil {
		return nil, err
	}

	tarballPath := filepath.Join(baseDir, "tarball")
	if err := utils.CreateDir(tarballPath); err != nil {
		return nil, err
	}

	packagesPath := filepath.Join(baseDir, "packages")
	if err := utils.CreateDir(packagesPath); err != nil {
		return nil, err
	}

	return &Config{
		NpmRegistryURL: "https://registry.npmjs.org/",
		BaseDir:        baseDir,
		ManifestDir:    filepath.Join(baseDir, "manifest"),
		TarballDir:     filepath.Join(baseDir, "tarball"),
		PackagesDir:    filepath.Join(baseDir, "packages"),

		LocalNodeModules: "./node_modules",
	}, nil
}
