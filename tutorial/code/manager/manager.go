package manager

import (
	"fmt"
	"go-npm/config"
	"go-npm/extractor"
	"go-npm/manifest"
	"go-npm/packagejson"
	"go-npm/tarball"
	"go-npm/version"
	"path/filepath"
)

type Manager struct {
	Config      *config.Config
	Manifest    *manifest.Manifest
	Version     *version.VersionInfo
	Tarball     *tarball.Tarball
	Extractor   *extractor.TGZExtractor
	PackageJSON *packagejson.PackageJSON
}

type job struct {
	Name    string
	Version string
}

func New() (*Manager, error) {
	cfg, err := config.New()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	m, err := manifest.NewManifest(cfg.ManifestDir, cfg.NpmRegistryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to init manifest: %w", err)
	}

	parser := packagejson.NewPackageJSONParser(cfg)
	pkgJSON, err := parser.Parse("package.json")
	if err != nil {
		return nil, fmt.Errorf("failed to parse package.json: %w", err)
	}

	return &Manager{
		Config:      cfg,
		Manifest:    m,
		Version:     version.NewVersionInfo(),
		Tarball:     tarball.NewTarball(),
		Extractor:   extractor.NewTGZExtractor(),
		PackageJSON: pkgJSON,
	}, nil
}

func (m *Manager) Install() error {
	var queue []job
	for name, version := range m.PackageJSON.Dependencies {
		queue = append(queue, job{Name: name, Version: version})
	}

	installed := make(map[string]bool)
	parser := packagejson.NewPackageJSONParser(m.Config)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if installed[current.Name] {
			continue
		}

		npmPackage, err := m.Manifest.Download(current.Name)
		if err != nil {
			return err
		}

		fmt.Println("Installing:", npmPackage.Name)

		resolvedVersion := m.Version.GetVersion(current.Version, npmPackage)
		fmt.Println("Resolved version:", resolvedVersion)

		downloadedPath, err := m.Tarball.Download(resolvedVersion, npmPackage)
		if err != nil {
			return err
		}

		destPath := filepath.Join("node_modules", npmPackage.Name)
		if err := m.Extractor.Extract(downloadedPath, destPath); err != nil {
			return err
		}

		installedPkgJSONPath := filepath.Join(destPath, "package.json")
		installedPkgJSON, err := parser.Parse(installedPkgJSONPath)
		if err == nil && installedPkgJSON.Dependencies != nil {
			for name, version := range installedPkgJSON.Dependencies {
				if !installed[name] {
					queue = append(queue, job{Name: name, Version: version})
				}
			}
		}

		installed[current.Name] = true
	}

	return nil
}
