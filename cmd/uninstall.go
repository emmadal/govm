package cmd

import (
	"github.com/emmadal/govm/internal"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall govm and every Go version it installed",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		yes, _ := cmd.Flags().GetBool("yes")
		return internal.Uninstall(yes)
	},
}

func init() {
	uninstallCmd.Flags().BoolP("yes", "y", false, "do not ask for confirmation")
}
