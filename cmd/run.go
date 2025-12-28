package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/ernesto27/go-npm/config"
	"github.com/ernesto27/go-npm/packagejson"
	"github.com/ernesto27/go-npm/scripts"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <script>",
	Short: "Run a script defined in package.json",
	Long:  `Execute a script defined in the "scripts" section of package.json.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runScript,
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func runScript(cmd *cobra.Command, args []string) error {
	scriptName := args[0]

	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}
	parser := packagejson.NewPackageJSONParser(cfg, nil)

	pkgJSON, err := parser.ParseDefault()
	if err != nil {
		return fmt.Errorf("failed to parse package.json: %w", err)
	}

	if len(pkgJSON.Scripts) == 0 {
		return fmt.Errorf("no scripts defined in package.json")
	}

	script, exists := pkgJSON.Scripts[scriptName]
	if !exists {
		return fmt.Errorf("script %q not found in package.json\n\nAvailable scripts:\n%s",
			scriptName, formatAvailableScripts(pkgJSON.Scripts))
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	nodeModulesPath := cwd + "/node_modules"
	executor := scripts.NewScriptExecutor(nodeModulesPath)

	pkgName := pkgJSON.Name
	pkgVersion := ""
	if v, ok := pkgJSON.Version.(string); ok {
		pkgVersion = v
	}

	fmt.Printf("\n> %s@%s %s\n", pkgName, pkgVersion, scriptName)

	if err := executor.Execute(script, cwd, pkgName, pkgVersion, scriptName); err != nil {
		return err
	}

	return nil
}

func formatAvailableScripts(scripts map[string]string) string {
	names := make([]string, 0, len(scripts))
	for name := range scripts {
		names = append(names, name)
	}
	sort.Strings(names)

	result := ""
	for _, name := range names {
		result += fmt.Sprintf("  %s: %s\n", name, scripts[name])
	}
	return result
}
