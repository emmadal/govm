package cmd

import (
	"github.com/emmadal/govm/pkg"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List installed Go versions",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return pkg.List()
	},
}

var lsRemoteCmd = &cobra.Command{
	Use:     "ls-remote",
	Short:   "List Go versions available for download",
	Example: "  govm ls-remote\n  govm ls-remote --all",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		all, _ := cmd.Flags().GetBool("all")
		return pkg.ListRemote(all)
	},
}

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Print the active Go version",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return pkg.Current()
	},
}

func init() {
	lsRemoteCmd.Flags().BoolP("all", "a", false, "show every release, including older patches and pre-releases")
}
