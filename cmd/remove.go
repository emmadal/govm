package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

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
		err := removeGovm()
		if err != nil {
			return err
		}
		return nil
	},
}

// removeGovm removes govm from the system
func removeGovm() error {
	// Ask for confirmation
	confirmed, err := pkg.ConfirmRemoval()
	if !confirmed || err != nil {
		return err
	}

	// Read the uninstall script from file
	content, err := os.ReadFile("uninstall.sh")
	if err != nil {
		return fmt.Errorf("failed to read uninstall script from 'uninstall.sh'")
	}

	// Ensure the script content isn't empty
	if len(content) == 0 {
		return fmt.Errorf("script 'uninstall.sh' is empty")
	}

	// Execute the uninstall script using the appropriate shell
	cmd := exec.Command("sh", "-c", string(content))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the uninstall script
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to uninstall govm")
	}

	return nil
}
