package main

import (
	"os"
	"strings"

	"github.com/emmadal/govm/cmd"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "govm",
	Short: "Go version manager. Manage multiple Go versions easily",
	Version: strings.Join([]string{
		"v1.0.0",
		"https://github.com/emmadal/govm",
	}, "\n"),
}

func init() {
	// Add all commands to the root command
	rootCmd.AddCommand(cmd.InstallCmd, cmd.UseCmd, cmd.ListCmd)
}

func main() {
	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		// If there's an error, exit with a non-zero status code
		os.Exit(1)
	}
}
