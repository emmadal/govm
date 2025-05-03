package cmd

import (
	"fmt"
	"github.com/emmadal/govm/internal"
	"strings"

	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:     "update",
	Short:   "update govm to the latest version",
	Example: strings.Join([]string{"$ govm update"}, "\n"),
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("expect no arguments")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return internal.UpdateGovm()
	},
}
