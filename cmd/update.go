/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// UpdateCmd represents the update command
var UpdateCmd = &cobra.Command{
	Use:     "update",
	Short:   "update govm to the latest version	",
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
		// Update govm
		err := updateGovm()
		if err != nil {
			return err
		}
		return nil
	},
}

// updateGovm updates govm to the latest version
func updateGovm() error {
	// Read the update script from file
	content, err := os.ReadFile("update.sh")
	if err != nil {
		return fmt.Errorf("failed to read update script from 'update.sh'")
	}

	// Ensure the script content isn't empty
	if len(content) == 0 {
		return fmt.Errorf("update script 'update.sh' is empty")
	}

	// Execute the update script using the appropriate shell
	cmd := exec.Command("sh", "-c", string(content))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the update script
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update govm: %v", err)
	}

	return nil
}
