package cmd

import (
	"fmt"
	"github.com/emmadal/govm/pkg"
	"strings"

	"github.com/spf13/cobra"
)

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
		binary := pkg.Binary{}
		if err := binary.GetAllVersions(); err != nil {
			return err
		}
		return binary.GoVersionDetails()
	},
}
