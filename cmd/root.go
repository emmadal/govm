package cmd

import (
	"github.com/emmadal/govm/internal"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:           "govm",
	Short:         "Go version manager. Manage multiple Go versions easily",
	Version:       internal.GetVersion() + "\nhttps://github.com/emmadal/govm",
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	rootCmd.AddCommand(installCmd, useCmd, listCmd, lsRemoteCmd, currentCmd, rmCmd, updateCmd, uninstallCmd)
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}
