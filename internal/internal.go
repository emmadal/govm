package internal

import (
	"strings"

	"github.com/emmadal/govm/cmd"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "govm",
	Short: "version manager for Go",
	Long:  "Allows you to install and switch between different Go versions",
	Version: strings.Join([]string{
		"v1.0.0",
		"https://github.com/emmadal/govm",
	}, "\n"),
}

func Internal() int {
	rootCmd.AddCommand(cmd.InstallCmd, cmd.UseCmd)
	err := rootCmd.Execute()
	if err != nil {
		return 1
	}
	return 0
}
