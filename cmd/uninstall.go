package cmd

import (
	"fmt"
	"github.com/emmadal/govm/internal"
	"strings"

	"github.com/spf13/cobra"
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:     "uninstall",
	Short:   "uninstall govm from the system",
	Example: strings.Join([]string{"$ govm uninstall"}, "\n"),
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("expect no arguments")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Ask for confirmation
		confirmed, err := internal.UninstallConfirm()
		if !confirmed || err != nil {
			return err
		}
		return internal.Uninstall()
	},
}
