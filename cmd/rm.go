/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"github.com/emmadal/govm/pkg"
	"strings"

	"github.com/spf13/cobra"
)

// rmCmd represents the rm command
var rmCmd = &cobra.Command{
	Use:     "rm",
	Short:   "Remove a specific Go version",
	Example: strings.Join([]string{"$ govm rm 1.21.0"}, "\n"),
	Args: func(cmd *cobra.Command, args []string) error {
		if strings.Contains(args[0], "go") {
			return fmt.Errorf("invalid version format. Please enter a valid version")
		}
		if len(args) > 1 || len(args) == 0 {
			return fmt.Errorf("expect one argument")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Remove the Go version
		binary := pkg.Binary{}
		if err := binary.RemoveGoVersion(args[0]); err != nil {
			return err
		}
		return nil
	},
}
