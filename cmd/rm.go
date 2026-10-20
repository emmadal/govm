package cmd

import (
	"github.com/emmadal/govm/pkg"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:     "rm <version>",
	Short:   "Remove an installed Go version",
	Example: "  govm rm 1.21.0\n  govm rm 1.21.0 --yes",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		yes, _ := cmd.Flags().GetBool("yes")
		return pkg.Remove(args[0], yes)
	},
}

func init() {
	rmCmd.Flags().BoolP("yes", "y", false, "do not ask for confirmation")
}
