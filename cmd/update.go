/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"strings"

	"github.com/emmadal/govm/internal"
	"github.com/spf13/cobra"
)

// UpdateCmd represents the update command
var UpdateCmd = &cobra.Command{
	Use:     "update",
	Short:   "update govm to the latest version",
	Example: strings.Join([]string{"$ govm update"}, "\n"),
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("expect no arguments")
		}
		return nil
	},
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return updateGovm()
	},
}

// updateGovm updates govm to the latest version
func updateGovm() error {
	// Use the Go implementation to update govm
	err := internal.UpdateGovm()
	if err != nil {
		return fmt.Errorf("failed to update govm: %v", err)
	}

	return nil
}
