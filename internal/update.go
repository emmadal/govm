package internal

import (
	"context"
	"fmt"
	"github.com/emmadal/govm/pkg"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// getBinaryName returns the appropriate binary name based on platform
func getBinaryName() string {
	if runtime.GOOS == "windows" {
		return "govm.exe"
	}
	return "govm"
}

// detectPlatform returns the OS and architecture for download
func detectPlatform() (string, string, error) {
	goos := runtime.GOOS

	// Map architecture
	arch := runtime.GOARCH
	switch arch {
	case "amd64", "arm64", "386", "x86_64":
		// These are already correctly named
		break
	default:
		return "", "", fmt.Errorf("unsupported architecture: %s", arch)
	}

	return goos, arch, nil
}

// releaseBaseURL is where the latest govm binaries are published.
var releaseBaseURL = "https://github.com/emmadal/govm/releases/latest/download"

// newDownloader builds the downloader used for the govm binary.
var newDownloader = func() *pkg.Downloader {
	return &pkg.Downloader{
		Mode: 0755,
		OnRetry: func(attempt, maxAttempts int, err error) {
			pkg.RedPrintln(fmt.Sprintf("Download interrupted: %v\n", err))
			pkg.BluePrintln(fmt.Sprintf("Retrying (%d/%d)...\n", attempt, maxAttempts))
		},
	}
}

// downloadLatestRelease downloads the latest govm binary for the current
// platform into a new temporary directory and returns the binary path. The
// directory is removed if the download fails.
func downloadLatestRelease(ctx context.Context) (string, error) {
	// Detect a platform
	goos, arch, err := detectPlatform()
	if err != nil {
		return "", err
	}

	// Construct download URL based on a platform
	downloadURL := fmt.Sprintf("%s/govm_%s_%s", releaseBaseURL, goos, arch)
	if goos == "windows" {
		downloadURL += ".exe"
	}

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "govm-update-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %v", err)
	}

	pkg.BluePrintln(fmt.Sprintf("Downloading latest govm binary for %s_%s...\n", goos, arch))

	// Download the binary
	binaryPath := filepath.Join(tmpDir, getBinaryName())
	if err := newDownloader().Download(ctx, downloadURL, binaryPath); err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to download govm binary: %w", err)
	}

	return binaryPath, nil
}

// copyFile copies src to dst. The content is written to a temporary file next
// to dst and renamed into place, so dst is never left half-written and a
// running binary is replaced rather than overwritten.
func copyFile(src, dst string) (err error) {
	inFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer func() {
		_ = inFile.Close()
	}()

	outFile, err := os.CreateTemp(filepath.Dir(dst), filepath.Base(dst)+".*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create destination file: %v", err)
	}
	defer func() {
		if err != nil {
			_ = outFile.Close()
			_ = os.Remove(outFile.Name())
		}
	}()

	if _, err = io.Copy(outFile, inFile); err != nil {
		return fmt.Errorf("failed to copy file: %v", err)
	}
	if err = outFile.Chmod(0755); err != nil {
		return fmt.Errorf("failed to set permissions: %v", err)
	}
	if err = outFile.Close(); err != nil {
		return fmt.Errorf("failed to close destination file: %v", err)
	}
	if err = os.Rename(outFile.Name(), dst); err != nil {
		return fmt.Errorf("failed to replace %s: %v", dst, err)
	}
	return nil
}

// getInstallDir returns the appropriate installation directory based on sudo access
func getInstallDir() string {
	homedir, err := os.UserHomeDir()
	if err != nil {
		pkg.RedPrintln(fmt.Sprintf("Error getting home directory: %v\n", err))
		os.Exit(1)
	}

	localBin := filepath.Join(homedir, ".local", "bin")
	// Ensure a local bin directory exists
	if err := os.MkdirAll(localBin, 0755); err != nil {
		pkg.RedPrintln(fmt.Sprintf("Error creating local bin directory: %v\n", err))
	}

	return localBin
}

// getBinaryVersion returns the current version of govm
func getBinaryVersion(latestVersion string) bool {
	versionFile := filepath.Join(getInstallDir(), "VERSION")
	data, err := os.ReadFile(versionFile)
	if err != nil {
		pkg.RedPrintln(fmt.Sprintf("Error reading version file: %v\n", err))
		return false
	}
	lines := strings.SplitN(string(data), "\n", 2)
	return strings.TrimSpace(lines[0]) == latestVersion
}

// UpdateGovm updates govm to the latest version
func UpdateGovm(ctx context.Context) error {
	pkg.BluePrintln("Updating govm - Go Version Manager\n")

	// Get the installation directory and check admin rights
	govmDir, err := getGovmExecDir()
	if err != nil {
		return fmt.Errorf("failed to determine installation directory: %v", err)
	}

	// Ensure the installation directory exists
	installDir := filepath.Dir(govmDir)
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("failed to create installation directory: %v", err)
	}

	// Get the latest version tag
	pkg.BluePrintln("Checking for updates...\n")
	binary := pkg.Binary{}
	if err := binary.GetLatestTag(); err != nil {
		return err
	}

	// Compare versions
	if getBinaryVersion(binary.LatestTag) {
		pkg.GreenPrintln("👍 govm is already up to date\n")
		return nil
	}

	// Download the release
	binaryPath, err := downloadLatestRelease(ctx)
	if err != nil {
		return err
	}
	// Clean up temp directory
	defer func() {
		_ = os.RemoveAll(filepath.Dir(binaryPath))
	}()

	// Install the binary
	pkg.BluePrintln("Installing govm binary...\n")
	installPath := filepath.Join(installDir, getBinaryName())

	// Use direct copy for user directories
	if err := copyFile(binaryPath, installPath); err != nil {
		return fmt.Errorf("failed to install govm binary: %v", err)
	}

	// Create a VERSION file
	versionFilePath := filepath.Join(installDir, "VERSION")
	sb := strings.Builder{}
	sb.WriteString(binary.LatestTag + "\n")
	sb.WriteString(strings.TrimSpace("time: " + time.Now().Format(time.RFC3339)))
	if err := os.WriteFile(versionFilePath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("failed to write VERSION file: %v", err)
	}

	pkg.GreenPrintln("🎉 govm has been successfully updated!\n")
	pkg.BluePrintln("For more information, visit: https://github.com/emmadal/govm")

	return nil
}
