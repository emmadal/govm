package cmd

import (
	"github.com/emmadal/govm/internal"
	"github.com/spf13/cobra"
	"strings"
)

// initCmd represents the base command when called without any subcommands
var initCmd = &cobra.Command{
	Use:   "govm",
	Short: "Go version manager. Manage multiple Go versions easily",
	Version: strings.Join(
		[]string{
			internal.GetVersion(),
			"https://github.com/emmadal/govm",
		}, "\n",
	),
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	initCmd.AddCommand(installCmd, useCmd, listCmd, rmCmd, updateCmd, removeCmd)
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return initCmd.Execute()
}
