package cmd

import (
	"fmt"

	"github.com/ernesto27/go-npm/config"
	"github.com/ernesto27/go-npm/info"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info <package[@version]>",
	Short: "Show information about a package",
	Long:  `Display detailed metadata about an npm package including version, license, description, dist-tags, maintainers, and more.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runInfo,
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

func runInfo(cmd *cobra.Command, args []string) error {
	pkgName, version := parsePackageArg(args[0])

	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	infoService, err := info.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create info service: %w", err)
	}

	return infoService.Show(pkgName, version)
}
