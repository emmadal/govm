package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/emmadal/govm/cmd"
	"github.com/emmadal/govm/pkg"
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
	SilenceErrors: true,
	SilenceUsage:  true,
}

// init adds the commands before the main function is called
func init() {
	rootCmd.AddCommand(cmd.InstallCmd, cmd.UseCmd, cmd.ListCmd)
}

// main is the entry point of the application
func main() {
	// Block execution on Windows
	if runtime.GOOS == "windows" {
		fmt.Fprintln(os.Stderr, "Error: Windows is not supported for govm")
		fmt.Fprintln(os.Stderr, "We will support Windows in the future")
		os.Exit(1)
	}

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s%s%s\n", pkg.Red_ANSI, err.Error(), pkg.Reset_ANSI)
		os.Exit(1)
	}
}
