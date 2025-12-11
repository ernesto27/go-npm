package cmd

import (
	"fmt"
	"github.com/ernesto27/go-npm/config"

	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage package cache",
	Long:  `Manage the package cache directories.`,
}

var cacheRmCmd = &cobra.Command{
	Use:   "rm",
	Short: "Remove all cached packages and manifests",
	Long:  `Remove all cached packages, manifests, and ETags. Preserves global installations.`,
	RunE:  runCacheRm,
}

func init() {
	rootCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(cacheRmCmd)
}

func runCacheRm(cmd *cobra.Command, args []string) error {
	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	if err := cfg.ClearCache(); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	fmt.Println("Cache cleared successfully")
	return nil
}
