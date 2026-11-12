package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/emmadal/govm/pkg"
	"github.com/spf13/cobra"
)

const MinVersion = "1.21.0"

var installCmd = &cobra.Command{
	Use:     "install",
	Short:   "Install a specific go version",
	Example: strings.Join([]string{"$ govm install 1.21.0"}, "\n"),
	Args: func(cmd *cobra.Command, args []string) error {
		return validateVersionArg(args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Stop cleanly on Ctrl-C: the download is aborted, not retried, and
		// its partial file is removed.
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()

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
		archive, err := tarball.DownloadGoVersion(ctx, args[0], directory.CacheDir)
		if err != nil {
			return err
		}

		// Install the Go version
		if err := tarball.InstallVersion(ctx, archive, args[0], directory.ConfigDir); err != nil {
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
