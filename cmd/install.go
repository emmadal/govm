package cmd

import (
	"fmt"
	"strings"

	"github.com/emmadal/govm/pkg"
	"github.com/spf13/cobra"
)

// InstallCmd represents the install command
var InstallCmd = &cobra.Command{
	Use:     "install",
	Short:   "Install a specific Go version",
	Example: strings.Join([]string{"$ govm install 1.20"}, "\n"),
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 || len(args) == 0 {
			return fmt.Errorf("Expect one argument\n")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		err := installGoVersion(args[0])
		return err
	},
}

func installGoVersion(version string) error {
	if err := pkg.CreateConfigDir(); err != nil {
		return err
	}

	// Check if the version is already downloaded
	cachedFile, err := pkg.CachedGoVersion(version)
	if err == nil {
		fmt.Printf("go%s is already downloaded. Skipping download\n", version)
		if err := pkg.UnzipDependency(pkg.REINSTALL, cachedFile, version); err != nil {
			return err
		}
		return pkg.UseGoVersion(version)
	}
	// Download the Go version
	err, file := pkg.DownloadGoVersion(version)
	if err != nil {
		return err
	}
	// Unzip the downloaded file
	if err := pkg.UnzipDependency(pkg.INSTALL, file, version); err != nil {
		return err
	}
	// Export the Go version
	if err := pkg.UseGoVersion(version); err != nil {
		return err
	}
	return nil
}
