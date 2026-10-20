package cmd

import (
	"github.com/emmadal/govm/pkg"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install <version>",
	Short: "Install a Go version and switch to it",
	Long: `Install a Go version and switch to it.

<version> is an exact release (1.22.3, go1.23rc1), a minor line (1.22,
which picks its newest patch release) or "latest". Downloads are verified
against the SHA-256 checksums published on go.dev.`,
	Example: "  govm install latest\n  govm install 1.22\n  govm install 1.21.5",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return pkg.Install(args[0])
	},
}
