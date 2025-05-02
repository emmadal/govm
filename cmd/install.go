package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/emmadal/govm/pkg"
	"github.com/spf13/cobra"
)

const MinVersion = "1.21.0"

var installCmd = &cobra.Command{
	Use:     "install",
	Short:   "Install a specific go version",
	Example: strings.Join([]string{"$ govm install 1.21.0"}, "\n"),
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
		if len(args[0]) < 6 {
			return fmt.Errorf("invalid version format. Please enter a valid version")
		}

		// Check if a version is >= MinVersion
		if !compareVersions(args[0], MinVersion) {
			return fmt.Errorf("minimum supported version is %s. Please install a newer version", MinVersion)
		}

		tarball := pkg.Tarball{}
		directory := pkg.Directory{}

		// Get the directories
		if err := directory.GetDirectories(); err != nil {
			return err
		}

		// Create the config directory
		if err := directory.CreateInstallDir(); err != nil {
			return err
		}

		// Download the Go version
		if err := tarball.DownloadGoVersion(args[0], directory.CacheDir); err != nil {
			return err
		}

		// Install the Go version
		if err := tarball.InstallVersion(tarball.File.Name(), args[0], directory.ConfigDir); err != nil {
			return err
		}

		// Export the Go version
		if err := tarball.UseGoVersion(args[0], directory.ConfigDir); err != nil {
			return err
		}

		return nil
	},
}

func compareVersions(a, b string) bool {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	// Convert version parts to integers
	for i := range 3 {
		aNum, _ := strconv.Atoi(aParts[i])
		bNum, _ := strconv.Atoi(bParts[i])

		if aNum > bNum {
			return true // it is greater
		} else if aNum < bNum {
			return false // a is smaller
		}
	}
	return true // Versions are equal
}
