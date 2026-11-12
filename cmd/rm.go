/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
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
		return validateVersionArg(args)
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
