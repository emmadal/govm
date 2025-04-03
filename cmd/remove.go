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
		return removeGovm()
	},
}

// removeGovm removes govm from the system
func removeGovm() error {
	// Ask for confirmation
	confirmed, err := pkg.ConfirmRemoval()
	if !confirmed || err != nil {
		return err
	}

	// Use the Go implementation to uninstall govm
	err = internal.Uninstall()
	if err != nil {
		return fmt.Errorf("failed to uninstall govm: %v", err)
	}

	return nil
}
