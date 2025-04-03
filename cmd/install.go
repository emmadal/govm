package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/emmadal/govm/pkg"
	"github.com/spf13/cobra"
)

const MIN_VERSION = "1.21.0"

// InstallCmd represents the install command
var InstallCmd = &cobra.Command{
	Use:     "install",
	Short:   "Install a specific Go version",
	Example: strings.Join([]string{"$ govm install 1.21.0"}, "\n"),
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 || len(args) == 0 {
			return fmt.Errorf("expect one argument")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args[0]) < 6 {
			return fmt.Errorf("invalid version format. Please enter a valid version")
		}
		// Check if version is >= MIN_VERSION
		if !compareVersions(args[0], MIN_VERSION) {
			return fmt.Errorf("minimum supported version is %s. Please install a up-to-date version", MIN_VERSION)
		}
		// Install the Go version
		return installGoVersion(args[0])
	},
}

func installGoVersion(version string) error {
	if err := pkg.CreateConfigDir(); err != nil {
		return err
	}

	// Check if the version is already downloaded
	cachedFile, err := pkg.CachedGoVersion(version)
	if err == nil {
		fmt.Fprintf(os.Stdout, "%s%s%s is already downloaded. Skipping download\n", pkg.Blue_ANSI, version, pkg.Reset_ANSI)
		if err := pkg.UnzipDependency(pkg.REINSTALL, cachedFile, version); err != nil {
			return err
		}
		if err := pkg.UseGoVersion(version); err != nil {
			return err
		}
		return nil
	}
	// Download the Go version
	file, err := pkg.DownloadGoVersion(version)
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

func compareVersions(a, b string) bool {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	// Convert version parts to integers
	for i := range 3 {
		aNum, _ := strconv.Atoi(aParts[i])
		bNum, _ := strconv.Atoi(bParts[i])

		if aNum > bNum {
			return true // a is greater
		} else if aNum < bNum {
			return false // a is smaller
		}
	}
	return true // Versions are equal
}
