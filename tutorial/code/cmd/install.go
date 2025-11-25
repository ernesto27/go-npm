package cmd

import (
	"fmt"
	"go-npm/manager"
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

	mgr, err := manager.New()
	if err != nil {
		return err
	}

	if err := mgr.Install(); err != nil {
		return err
	}

	fmt.Printf("Package installed to %s\n", filepath.Join("node_modules", "express"))

	return nil
}
