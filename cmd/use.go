package cmd

import (
	"fmt"
	"os"
	"runtime"

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
		if runtime.GOOS != "windows" {
			err := verifyFileExists(args[0])
			if err != nil {
				return err
			}
			return nil
		}
		return fmt.Errorf("%s-%s OS is not supported\n", runtime.GOOS, runtime.GOARCH)
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
