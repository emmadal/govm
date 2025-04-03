package internal

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// CheckSudo checks if the user has sudo access
func CheckSudo() bool {
	cmd := exec.Command("sudo", "-n", "true")
	err := cmd.Run()
	return err == nil
}

// GetInstallDir returns the appropriate installation directory based on sudo access
func GetInstallDir() string {
	if CheckSudo() {
		return "/usr/local/bin"
	}

	homedir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stdout, "\033[31mError getting home directory: %v\033[0m\n", err)
		os.Exit(1)
	}

	localBin := filepath.Join(homedir, ".local", "bin")
	// Ensure local bin directory exists
	os.MkdirAll(localBin, 0755)

	return localBin
}

// DetectPlatform returns the OS and architecture for download
func DetectPlatform() (string, string, error) {
	goos := runtime.GOOS

	// Map architecture
	arch := runtime.GOARCH
	switch arch {
	case "amd64", "arm64", "386":
		// These are already correctly named
		break
	default:
		return "", "", fmt.Errorf("unsupported architecture: %s", arch)
	}

	return goos, arch, nil
}

// DownloadLatestRelease downloads the latest govm binary for the current platform
func DownloadLatestRelease() (string, error) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "govm-update-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %v", err)
	}

	// Detect platform
	goos, arch, err := DetectPlatform()
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", err
	}

	// Construct download URL
	downloadURL := fmt.Sprintf("https://github.com/emmadal/govm/releases/latest/download/govm_%s_%s", goos, arch)

	fmt.Fprintf(os.Stdout, "\033[34mDownloading latest govm binary for %s_%s...\033[0m\n", goos, arch)

	// Download the binary
	resp, err := http.Get(downloadURL)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to download govm binary: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_ = os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to download govm binary: HTTP status %d", resp.StatusCode)
	}

	// Save the binary to a temporary file
	binaryPath := filepath.Join(tmpDir, "govm")
	outFile, err := os.OpenFile(binaryPath, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to create output file: %v", err)
	}

	_, err = io.Copy(outFile, resp.Body)
	outFile.Close()
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to write output file: %v", err)
	}

	return binaryPath, nil
}

// UpdateGovm updates govm to the latest version
func UpdateGovm() error {
	fmt.Fprint(os.Stdout, "\033[34m\033[1mUpdating govm - Go Version Manager\033[0m\n")

	// Get install directory
	installDir := GetInstallDir()
	hasSudo := CheckSudo() && installDir == "/usr/local/bin"

	// Download latest release
	binaryPath, err := DownloadLatestRelease()
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(binaryPath))

	// Install the binary
	fmt.Fprint(os.Stdout, "\033[34mInstalling govm binary...\033[0m\n")
	installPath := filepath.Join(installDir, "govm")

	if hasSudo {
		cmd := exec.Command("sudo", "cp", binaryPath, installPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install govm binary: %v", err)
		}

		chmodCmd := exec.Command("sudo", "chmod", "+x", installPath)
		chmodCmd.Stdout = os.Stdout
		chmodCmd.Stderr = os.Stderr
		if err := chmodCmd.Run(); err != nil {
			return fmt.Errorf("failed to set execute permissions: %v", err)
		}
	} else {
		// Copy the binary to the install directory
		inFile, err := os.Open(binaryPath)
		if err != nil {
			return fmt.Errorf("failed to open binary file: %v", err)
		}
		defer inFile.Close()

		outFile, err := os.OpenFile(installPath, os.O_CREATE|os.O_WRONLY, 0755)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		defer outFile.Close()

		_, err = io.Copy(outFile, inFile)
		if err != nil {
			return fmt.Errorf("failed to copy binary: %v", err)
		}
	}

	fmt.Fprint(os.Stdout, "\033[32m\033[1m✓ govm has been successfully updated!\033[0m\n\n")
	fmt.Fprint(os.Stdout, "\033[34mFor more information, visit: https://github.com/emmadal/govm\033[0m\n")

	return nil
}
