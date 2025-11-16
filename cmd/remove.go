package cmd

import (
	"fmt"
	"npm-packager/manager"

	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove <package>",
	Aliases: []string{"rm"},
	Short:   "Remove a package from package.json and node_modules",
	Long:    `Remove a package from package.json dependencies and delete it from node_modules.`,
	Args:    cobra.ExactArgs(1),
	RunE:    runRemove,
}

func init() {
	rootCmd.AddCommand(removeCmd)
}

func runRemove(cmd *cobra.Command, args []string) error {
	deps, err := manager.BuildDependencies()
	if err != nil {
		return fmt.Errorf("error building dependencies: %w", err)
	}

	packageManager, err := manager.New(deps)
	if err != nil {
		return fmt.Errorf("error creating package manager: %w", err)
	}

	if err := packageManager.Remove(args[0], true); err != nil {
		return fmt.Errorf("error removing package: %w", err)
	}

	fmt.Println("Package removed successfully")
	return nil
}
