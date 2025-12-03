package workspace

import (
	"fmt"
	"path/filepath"

	"npm-packager/packagejson"
)

// Workspace represents a single workspace package in a monorepo
type Workspace struct {
	Name        string
	Version     string
	Path        string
	PackageJSON *packagejson.PackageJSON
}

// WorkspaceRegistry maintains a registry of all workspace packages in a monorepo
type WorkspaceRegistry struct {
	Packages map[string]*Workspace
	RootDir  string
	parser   *packagejson.PackageJSONParser
}

// NewWorkspaceRegistry creates a new workspace registry for the given root directory
func NewWorkspaceRegistry(rootDir string, parser *packagejson.PackageJSONParser) *WorkspaceRegistry {
	return &WorkspaceRegistry{
		Packages: make(map[string]*Workspace),
		RootDir:  rootDir,
		parser:   parser,
	}
}

// Discover discovers all workspace packages based on the root package.json
func (wr *WorkspaceRegistry) Discover(rootPackageJSON *packagejson.PackageJSON) error {
	patterns := rootPackageJSON.GetWorkspaces()
	if len(patterns) == 0 {
		return fmt.Errorf("no workspace patterns found in root package.json")
	}

	workspacePaths, err := wr.expandGlobPatterns(patterns)
	if err != nil {
		return fmt.Errorf("failed to expand workspace patterns: %w", err)
	}

	for _, wsPath := range workspacePaths {
		if err := wr.addWorkspacePackage(wsPath); err != nil {
			return err
		}
	}

	return nil
}

func (wr *WorkspaceRegistry) expandGlobPatterns(patterns []string) ([]string, error) {
	var workspacePaths []string

	for _, pattern := range patterns {
		fullPattern := filepath.Join(wr.RootDir, pattern)

		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern %s: %w", pattern, err)
		}

		for _, match := range matches {
			absMatch, err := filepath.Abs(match)
			if err != nil {
				continue
			}
			workspacePaths = append(workspacePaths, absMatch)
		}
	}

	return workspacePaths, nil
}

func (wr *WorkspaceRegistry) addWorkspacePackage(wsPath string) error {
	packageJSONPath := filepath.Join(wsPath, "package.json")

	pkgJSON, err := wr.parser.Parse(packageJSONPath)
	if err != nil {
		return fmt.Errorf("failed to parse package.json at %s: %w", packageJSONPath, err)
	}

	if pkgJSON.Name == "" {
		return fmt.Errorf("workspace package at %s has no name field", wsPath)
	}

	version := wr.extractVersion(pkgJSON.Version)

	absPath, err := filepath.Abs(wsPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", wsPath, err)
	}

	ws := &Workspace{
		Name:        pkgJSON.Name,
		Version:     version,
		Path:        absPath,
		PackageJSON: pkgJSON,
	}

	wr.Packages[pkgJSON.Name] = ws
	return nil
}

func (wr *WorkspaceRegistry) extractVersion(version interface{}) string {
	switch v := version.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.1f", v)
	case int:
		return fmt.Sprintf("%d.0.0", v)
	default:
		return "0.0.0"
	}
}

// IsWorkspacePackage checks if a package name belongs to the workspace
func (wr *WorkspaceRegistry) IsWorkspacePackage(name string) bool {
	_, exists := wr.Packages[name]
	return exists
}

// GetWorkspacePackage retrieves a workspace package by name
func (wr *WorkspaceRegistry) GetWorkspacePackage(name string) (*Workspace, bool) {
	pkg, exists := wr.Packages[name]
	return pkg, exists
}

// Validate performs validation checks on the workspace registry
func (wr *WorkspaceRegistry) Validate() []error {
	var errors []error

	pathsSeen := make(map[string]string)
	for name, pkg := range wr.Packages {
		if existingName, exists := pathsSeen[pkg.Path]; exists {
			errors = append(errors, fmt.Errorf(
				"duplicate path %s used by packages %s and %s", pkg.Path, name, existingName))
		}
		pathsSeen[pkg.Path] = name
	}

	for _, pkg := range wr.Packages {
		if pkg.Name == "" {
			errors = append(errors, fmt.Errorf(
				"workspace at %s has no name field in package.json", pkg.Path))
		}
	}

	return errors
}
