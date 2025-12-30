package info

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ernesto27/go-npm/config"
	"github.com/ernesto27/go-npm/manifest"
	"github.com/ernesto27/go-npm/parsejson"
	"github.com/ernesto27/go-npm/version"
)

var (
	nameStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("cyan"))
	versionStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("green"))
	licenseStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("yellow"))
	headerStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("magenta"))
	keyStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	urlStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("blue")).Underline(true)
	keywordStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	maintainerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("251"))
	dateStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)

// Info handles fetching and displaying package information
type Info struct {
	manifest *manifest.Manifest
	parser   *parsejson.Parser
	version  *version.Info
}

// New creates a new Info instance
func New(cfg *config.Config) (*Info, error) {
	m, err := manifest.NewManifest(cfg.BaseDir, config.NPMRegistryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create manifest: %w", err)
	}

	return &Info{
		manifest: m,
		parser:   parsejson.New(),
		version:  version.New(),
	}, nil
}

// Show fetches package info and prints it to stdout
func (i *Info) Show(pkgName, requestedVersion string) error {
	manifestPath := filepath.Join(i.manifest.Path, pkgName+".json")

	// Check if manifest exists in cache, download if not
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		_, _, err := i.manifest.Download(pkgName, "")
		if err != nil {
			return fmt.Errorf("package '%s' not found on npm registry", pkgName)
		}
	}

	npmPkg, err := i.parser.Parse(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	resolvedVersion := i.version.GetVersion(requestedVersion, npmPkg)

	versionData, exists := npmPkg.Versions[resolvedVersion]
	if !exists {
		return fmt.Errorf("version '%s' not found for package '%s'", resolvedVersion, pkgName)
	}

	printPackageInfo(npmPkg, &versionData, resolvedVersion)
	return nil
}

func printPackageInfo(pkg *manifest.NPMPackage, ver *manifest.Version, resolvedVersion string) {
	license := extractLicense(pkg.License, ver.License)
	depsCount := len(ver.Dependencies)
	versionsCount := len(pkg.Versions)

	// Header: package@version | license | deps | versions
	fmt.Printf("%s@%s | %s | %s %d | %s %d\n",
		nameStyle.Render(pkg.Name),
		versionStyle.Render(resolvedVersion),
		licenseStyle.Render(license),
		keyStyle.Render("deps:"), depsCount,
		keyStyle.Render("versions:"), versionsCount)

	if pkg.Description != "" {
		fmt.Println(pkg.Description)
	}

	if homepage := extractString(pkg.Homepage); homepage != "" {
		fmt.Println(urlStyle.Render(homepage))
	}

	if keywords := extractKeywords(pkg.Keywords); len(keywords) > 0 {
		fmt.Printf("%s %s\n", keyStyle.Render("keywords:"), keywordStyle.Render(strings.Join(keywords, ", ")))
	}

	fmt.Println()

	// Dist section
	fmt.Println(headerStyle.Render("dist"))
	fmt.Printf(" %s %s\n", keyStyle.Render(".tarball:"), urlStyle.Render(ver.Dist.Tarball))
	fmt.Printf(" %s %s\n", keyStyle.Render(".shasum:"), ver.Dist.Shasum)
	if ver.Dist.Integrity != "" {
		fmt.Printf(" %s %s\n", keyStyle.Render(".integrity:"), ver.Dist.Integrity)
	}
	if ver.Dist.UnpackedSize > 0 {
		fmt.Printf(" %s %s\n", keyStyle.Render(".unpackedSize:"), versionStyle.Render(formatBytes(ver.Dist.UnpackedSize)))
	}

	fmt.Println()

	// Dist-tags section
	fmt.Println(headerStyle.Render("dist-tags:"))
	printDistTags(pkg.DistTags)

	fmt.Println()

	// Maintainers section
	if maintainers := extractMaintainers(pkg.Maintainers); len(maintainers) > 0 {
		fmt.Println(headerStyle.Render("maintainers:"))
		for _, m := range maintainers {
			if m.Email != "" {
				fmt.Printf("- %s %s\n", maintainerStyle.Render(m.Name), keyStyle.Render("<"+m.Email+">"))
			} else {
				fmt.Printf("- %s\n", maintainerStyle.Render(m.Name))
			}
		}
		fmt.Println()
	}

	// Published date
	if pubDate, ok := pkg.Time[resolvedVersion]; ok {
		fmt.Printf("%s %s\n", keyStyle.Render("Published:"), dateStyle.Render(pubDate))
	}
}

func extractLicense(pkgLicense, verLicense any) string {
	for _, lic := range []any{verLicense, pkgLicense} {
		switch v := lic.(type) {
		case string:
			if v != "" {
				return v
			}
		case map[string]interface{}:
			if t, ok := v["type"].(string); ok {
				return t
			}
		}
	}
	return "Unknown"
}

func extractString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func extractKeywords(v any) []string {
	switch kw := v.(type) {
	case []interface{}:
		result := make([]string, 0, len(kw))
		for _, k := range kw {
			if s, ok := k.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case []string:
		return kw
	}
	return nil
}

func extractMaintainers(v any) []manifest.Maintainer {
	switch m := v.(type) {
	case []interface{}:
		result := make([]manifest.Maintainer, 0, len(m))
		for _, item := range m {
			if obj, ok := item.(map[string]interface{}); ok {
				info := manifest.Maintainer{}
				if name, ok := obj["name"].(string); ok {
					info.Name = name
				}
				if email, ok := obj["email"].(string); ok {
					info.Email = email
				}
				if info.Name != "" {
					result = append(result, info)
				}
			}
		}
		return result
	}
	return nil
}

func printDistTags(tags manifest.DistTags) {
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		fmt.Printf("%s %s\n", keyStyle.Render(k+":"), versionStyle.Render(tags[k]))
	}
}

func formatBytes(bytes int) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
