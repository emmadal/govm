package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/emmadal/govm/pkg"
	"github.com/spf13/cobra"
)

// useCmd represents the use command
var useCmd = &cobra.Command{
	Use:     "use",
	Short:   "Use a specific Go version",
	Example: strings.Join([]string{"$ govm use 1.21.0"}, "\n"),
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 || len(args) == 0 {
			return fmt.Errorf("expect one argument")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		err := verifyFileExists(args[0])
		if err != nil {
			return err
		}
		err = pkg.UseGoVersion(args[0])
		if err != nil {
			return err
		}
		return nil
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
	return nil
}
