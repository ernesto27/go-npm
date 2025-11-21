package cmd

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

//go:embed version.json
var versionFile []byte

type VersionInfo struct {
	Version string `json:"version"`
}

func getVersion() string {
	var versionInfo VersionInfo
	if err := json.Unmarshal(versionFile, &versionInfo); err != nil {
		return "unknown"
	}
	return versionInfo.Version
}

var rootCmd = &cobra.Command{
	Use:     "go-npm",
	Short:   "A Go implementation of npm package manager",
	Long:    `go-npm is a Go implementation of an npm package manager that downloads and installs npm packages and their dependencies.`,
	Version: getVersion(),
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}
