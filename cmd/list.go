package cmd

import (
	"fmt"
	"strings"

	"github.com/emmadal/govm/pkg"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List installed Go versions",
	Example: strings.Join([]string{"$ govm list"}, "\n"),
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("expect no arguments")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// List installed Go versions
		err := pkg.ListGoVersions()
		if err != nil {
			return err
		}
		return nil
	},
}
