package cmd

import (
	"fmt"
	"github.com/ernesto27/go-npm/manager"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <package[@version]>",
	Short: "Add a package to package.json and install it",
	Long:  `Add a package to package.json dependencies and install it.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	pkg, version := parsePackageArg(args[0])

	deps, err := manager.BuildDependencies(getVersion())
	if err != nil {
		return fmt.Errorf("error building dependencies: %w", err)
	}

	packageManager, err := manager.New(deps)
	if err != nil {
		return fmt.Errorf("error creating package manager: %w", err)
	}

	if err := packageManager.Add(pkg, version, false); err != nil {
		return fmt.Errorf("error adding package: %w", err)
	}

	return nil
}
