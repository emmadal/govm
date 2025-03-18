package internal

import (
	"strings"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "govm",
	Short: "version manager for Go.",
	Long:  "Allows you to easily switch between different Go versions.",
	Version: strings.Join([]string{
		"v1.0.0",
		"https://github.com/emmadal/govm",
	}, "\n"),
	Example: strings.Join([]string{
		"$ govm install 1.21.3",
		"$ govm use 1.21.3",
	}, "\n"),
}

func Internal() int {
	rootCmd.AddCommand()
	err := rootCmd.Execute()
	if err != nil {
		return 1
	}
	return 0
}
