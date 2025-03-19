package cmd

import (
	"fmt"
	"os"

	"github.com/emmadal/govm/pkg"
	"github.com/spf13/cobra"
)

// UseCmd represents the use command
var UseCmd = &cobra.Command{
	Use:   "use",
	Short: "Use Go version",
	Long:  "Use a specific Go version",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 || len(args) == 0 {
			return fmt.Errorf("Expect one argument\n")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		err := verifyFileExists(args[0])
		return err
	},
}

// verifyFileExists checks if a file exists
func verifyFileExists(version string) error {
	file, err := pkg.CachedGoVersion(version)
	if err != nil {
		message := fmt.Sprintf("go%s is not installed. Please do 'govm install %s' first.", version, version)
		return fmt.Errorf("%s%s%s", pkg.Red_ANSI, message, pkg.Reset_ANSI)
	}
	if _, err := os.Stat(file); err != nil {
		return err
	}
	return pkg.UseGoVersion(version)
}
