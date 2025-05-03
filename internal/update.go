package internal

import (
	"fmt"
	"github.com/emmadal/govm/pkg"
	"io"
	"net/http"
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

// downloadLatestRelease downloads the latest govm binary for the current platform
func downloadLatestRelease() (string, error) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "govm-update-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %v", err)
	}

	// Detect a platform
	goos, arch, err := detectPlatform()
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", err
	}

	// Construct download URL and binary name based on a platform
	var downloadURL string
	if goos == "windows" {
		downloadURL = fmt.Sprintf("https://github.com/emmadal/govm/releases/latest/download/govm_%s_%s.exe", goos, arch)
	} else {
		downloadURL = fmt.Sprintf("https://github.com/emmadal/govm/releases/latest/download/govm_%s_%s", goos, arch)
	}

	pkg.BluePrintln(fmt.Sprintf("Downloading latest govm binary for %s_%s...\n", goos, arch))

	// Download the binary
	resp, err := http.Get(downloadURL)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to download govm binary: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			_ = os.RemoveAll(tmpDir)
			pkg.RedPrintln(fmt.Sprintf("Error closing response body: %v\n", err))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		_ = os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to download govm binary: HTTP status %d", resp.StatusCode)
	}

	// Save the binary to a temporary file
	binaryPath := filepath.Join(tmpDir, getBinaryName())
	outFile, err := os.OpenFile(binaryPath, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to create output file: %v", err)
	}

	_, err = io.Copy(outFile, resp.Body)
	err2 := outFile.Close()
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to write output file: %v", err)
	}
	if err2 != nil {
		_ = os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to close output file: %v", err2)
	}

	return binaryPath, nil
}

// copyFile copies a file from src to dst with appropriate permissions
func copyFile(src, dst string) error {
	// Open source file
	inFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer func() {
		_ = inFile.Close()
	}()

	// Create a destination file
	outFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %v", err)
	}
	defer func() {
		_ = outFile.Close()
	}()

	// Copy the content
	_, err = io.Copy(outFile, inFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %v", err)
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
func UpdateGovm() error {
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
	binaryPath, err := downloadLatestRelease()
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
	file, err := os.OpenFile(versionFilePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to create VERSION file: %v", err)
	}
	defer func() {
		_ = file.Close()
	}()

	sb := strings.Builder{}
	sb.WriteString(binary.LatestTag + "\n")
	sb.WriteString(strings.TrimSpace("time: " + time.Now().Format(time.RFC3339)))
	_, err = file.WriteString(sb.String())
	if err != nil {
		return fmt.Errorf("failed to write VERSION file: %v", err)
	}

	pkg.GreenPrintln("🎉 govm has been successfully updated!\n")
	pkg.BluePrintln("For more information, visit: https://github.com/emmadal/govm")

	return nil
}
