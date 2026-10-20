package cmd

import (
	"github.com/emmadal/govm/pkg"
	"github.com/spf13/cobra"
)

var useCmd = &cobra.Command{
	Use:     "use <version>",
	Short:   "Switch to an installed Go version",
	Example: "  govm use 1.22.3\n  govm use 1.22\n  govm use latest",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return pkg.Use(args[0])
	},
}
