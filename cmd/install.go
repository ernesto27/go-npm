package cmd

import (
	"fmt"
	"github.com/ernesto27/go-npm/manager"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	globalFlag     bool
	productionFlag bool
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
	installCmd.Flags().BoolVarP(&globalFlag, "global", "g", false, "Install package globally")
	installCmd.Flags().BoolVar(&productionFlag, "production", false, "Install only production dependencies")
}

func parsePackageArg(pkgArg string) (string, string) {
	parts := strings.Split(pkgArg, "@")
	pkg := parts[0]
	version := ""
	if len(parts) > 1 {
		version = parts[1]
	}
	return pkg, version
}

func runInstall(cmd *cobra.Command, args []string) error {
	startTime := time.Now()

	deps, err := manager.BuildDependencies()
	if err != nil {
		return fmt.Errorf("error building dependencies: %w", err)
	}

	packageManager, err := manager.New(deps)
	if err != nil {
		return fmt.Errorf("error creating package manager: %w", err)
	}

	if globalFlag {
		if len(args) < 1 {
			return fmt.Errorf("package name is required for global installation")
		}

		pkg, version := parsePackageArg(args[0])

		if err := packageManager.SetupGlobal(); err != nil {
			return fmt.Errorf("error setting up global installation: %w", err)
		}

		if err := packageManager.InstallGlobal(pkg, version); err != nil {
			return fmt.Errorf("error installing globally: %w", err)
		}

		fmt.Printf("\nExecution completed in: %v\n", time.Since(startTime))
		return nil
	}

	if err := packageManager.ParsePackageJSON(productionFlag); err != nil {
		return fmt.Errorf("error parsing package.json: %w", err)
	}

	if err := packageManager.InstallFromCache(); err != nil {
		return err
	}

	fmt.Printf("\nExecution completed in: %v\n", time.Since(startTime))
	return nil
}
