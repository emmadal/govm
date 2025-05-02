package cmd

import (
	"fmt"
	"github.com/emmadal/govm/pkg"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// useCmd represents the use command
var useCmd = &cobra.Command{
	Use:     "use",
	Short:   "Use a specific Go version",
	Example: strings.Join([]string{"$ govm use 1.21.0"}, "\n"),
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
		// Use the Go version
		tarball := pkg.Tarball{}
		binary := pkg.Binary{}

		// Get the cached Go version
		if err := binary.CachedGoVersion(args[0]); err != nil {
			return err
		}
		
		// Check if the version exists before proceeding
		folder := filepath.Join(binary.InstallDir, fmt.Sprintf("go%s", args[0]))
		fileInfo, err := os.Stat(folder)
		if err != nil {
			return err
		}

		// use the Go version
		if fileInfo.IsDir() && fileInfo.Size() > 0 {
			directory := pkg.Directory{}
			if err := directory.GetDirectories(); err != nil {
				return err
			}
			// to Use the Go version
			if err := tarball.UseGoVersion(args[0], directory.ConfigDir); err != nil {
				return err
			}
		}
		return nil
	},
}
