package cmd

import (
	"github.com/emmadal/govm/internal"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update govm to the latest version",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		return internal.UpdateGovm(force)
	},
}

func init() {
	updateCmd.Flags().Bool("force", false, "reinstall even if already up to date")
}
