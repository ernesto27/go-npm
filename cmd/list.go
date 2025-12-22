package cmd

import (
	"fmt"

	"github.com/ernesto27/go-npm/config"
	"github.com/ernesto27/go-npm/list"
	"github.com/ernesto27/go-npm/packagejson"
	"github.com/ernesto27/go-npm/yarnlock"
	"github.com/spf13/cobra"
)

var listAll bool

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List installed packages",
	Long:    `Display a tree of installed packages and their dependencies.`,
	RunE:    runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVar(&listAll, "all", false, "Show all dependencies (full tree)")
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	parser := packagejson.NewPackageJSONParser(cfg, yarnlock.NewYarnLockParser())
	pkgJSON, err := parser.ParseDefault()
	if err != nil {
		return fmt.Errorf("failed to parse package.json: %w", err)
	}

	if parser.PackageLock == nil {
		return fmt.Errorf("no lock file found. Run 'go-npm install' first")
	}

	projectName := "project"
	projectVersion := ""
	if pkgJSON.Name != "" {
		projectName = pkgJSON.Name
		if v, ok := pkgJSON.Version.(string); ok {
			projectVersion = v
		}
	}

	lister := list.New(parser.PackageLock, projectName, projectVersion)
	lister.ShowAll = listAll
	lister.Print()

	return nil
}
