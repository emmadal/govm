package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/emmadal/govm/cmd"
	"github.com/emmadal/govm/pkg"
	"github.com/spf13/cobra"
)

func getVersion() string {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return "dev" // Fallback if VERSION file is missing
	}
	versionFile := filepath.Join(homedir, ".local", "bin", "VERSION")
	data, err := os.ReadFile(versionFile)
	if err != nil {
		fmt.Println("Error reading VERSION file:", err)
		return "dev" // Fallback if VERSION file is missing
	}
	return strings.TrimSpace(string(data))
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "govm",
	Short: "Go version manager. Manage multiple Go versions easily",
	Version: strings.Join([]string{
		getVersion(),
		"https://github.com/emmadal/govm",
	}, "\n"),
	SilenceErrors: true,
	SilenceUsage:  true,
}

// init adds the commands before the main function is called
func init() {
	rootCmd.AddCommand(cmd.InstallCmd, cmd.UseCmd, cmd.ListCmd, cmd.RmCmd, cmd.UpdateCmd, cmd.RemoveCmd)
}

// main is the entry point of the application
func main() {
	// Block execution on Windows
	if runtime.GOOS == "windows" {
		fmt.Fprintln(os.Stderr, "Error: Windows is not supported for govm.\nAlternative: Use WSL (Windows Subsystem for Linux).")
		os.Exit(1)
	}

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s%s%s\n", pkg.Red_ANSI, err.Error(), pkg.Reset_ANSI)
		os.Exit(1)
	}
}
