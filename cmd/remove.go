package cmd

import (
	"fmt"
	"strings"

	"github.com/emmadal/govm/internal"
	"github.com/emmadal/govm/pkg"
	"github.com/spf13/cobra"
)

// RemoveCmd represents the remove command
var RemoveCmd = &cobra.Command{
	Use:     "uninstall",
	Short:   "uninstall govm from the system",
	Example: strings.Join([]string{"$ govm uninstall"}, "\n"),
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("expect no arguments")
		}
		return nil
	},
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Ask for confirmation
		confirmed, err := pkg.ConfirmRemoval()
		if !confirmed || err != nil {
			return err
		}

		// Remove govm
		if err := internal.Uninstall(); err != nil {
			return err
		}
		return nil
	},
}
