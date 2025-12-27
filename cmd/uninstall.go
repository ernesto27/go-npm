package cmd

import (
	"fmt"

	"github.com/ernesto27/go-npm/manager"
	"github.com/ernesto27/go-npm/types"
	"github.com/spf13/cobra"
)

var uninstallGlobalFlag bool

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <package>",
	Short: "Uninstall a package",
	Long:  `Uninstall a package from node_modules or from global installation.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runUninstall,
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
	uninstallCmd.Flags().BoolVarP(&uninstallGlobalFlag, "global", "g", false, "Uninstall package globally")
}

func runUninstall(cmd *cobra.Command, args []string) error {
	opts := types.BuildOptions{
		Version: getVersion(),
	}
	deps, err := manager.BuildDependencies(opts)
	if err != nil {
		return fmt.Errorf("error building dependencies: %w", err)
	}

	packageManager, err := manager.New(deps)
	if err != nil {
		return fmt.Errorf("error creating package manager: %w", err)
	}

	if uninstallGlobalFlag {
		if err := packageManager.SetupGlobal(); err != nil {
			return fmt.Errorf("error setting up global installation: %w", err)
		}

		if err := packageManager.Remove(args[0], false); err != nil {
			return fmt.Errorf("error removing package: %w", err)
		}
	} else {
		if err := packageManager.Remove(args[0], true); err != nil {
			return fmt.Errorf("error removing package: %w", err)
		}
	}

	fmt.Println("Package removed successfully")
	return nil
}
