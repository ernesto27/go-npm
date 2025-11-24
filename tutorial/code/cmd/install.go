package cmd

import (
	"fmt"
	"go-npm/config"
	"go-npm/extractor"
	"go-npm/manifest"
	"go-npm/tarball"
	"go-npm/version"
	"path/filepath"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:     "install [package[@version]]",
	Aliases: []string{"i"},
	Short:   "Install packages",
	Long:    `Install packages from package.json or install a specific package globally.`,
	RunE:    runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	fmt.Println("Starting installation process...")

	cfg, err := config.New()
	if err != nil {
		panic(err)
	}

	manifest, err := manifest.NewManifest(cfg.ManifestDir, cfg.NpmRegistryURL)
	if err != nil {
		panic(err)
	}

	npmPackage, err := manifest.Download("express")
	if err != nil {
		panic(err)
	}

	fmt.Println(npmPackage.Name)

	v := version.NewVersionInfo()
	resolvedVersion := v.GetVersion("^4.0.0", npmPackage)
	fmt.Println("Resolved version:", resolvedVersion)

	tarball := tarball.NewTarball()
	downloadedPath, err := tarball.Download(resolvedVersion, npmPackage)
	if err != nil {
		panic(err)
	}

	extractor := extractor.NewTGZExtractor()
	destPath := filepath.Join("node_modules", npmPackage.Name)
	if err := extractor.Extract(downloadedPath, destPath); err != nil {
		panic(err)
	}

	fmt.Printf("Package installed to %s\n", destPath)

	return nil
}
