package internal

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/emmadal/govm/pkg"
)

// getInstallDir returns the appropriate installation directory based on sudo access
func getInstallDir() string {
	if checkSudo() {
		return "/usr/local/bin"
	}

	homedir, err := os.UserHomeDir()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "\033[31mError getting home directory: %v\033[0m\n", err)
		os.Exit(1)
	}

	localBin := filepath.Join(homedir, ".local", "bin")
	// Ensure a local bin directory exists
	if err := os.MkdirAll(localBin, 0755); err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "\033[31mError creating local bin directory: %v\033[0m\n", err)
	}

	return localBin
}

// detectPlatform returns the OS and architecture for download
func detectPlatform() (string, string, error) {
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

	// Construct download URL
	downloadURL := fmt.Sprintf("https://github.com/emmadal/govm/releases/latest/download/govm_%s_%s", goos, arch)

	_, _ = fmt.Fprintf(os.Stdout, "\033[34mDownloading latest govm binary for %s_%s...\033[0m\n", goos, arch)

	// Download the binary
	resp, err := http.Get(downloadURL)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to download govm binary: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			_ = os.RemoveAll(tmpDir)
			_, _ = fmt.Fprint(os.Stdout, fmt.Sprintf("\033[31mError closing response body: %v\033[0m\n", err))
		}
	}()

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
	defer func() {
		err := outFile.Close()
		if err != nil {
			_ = os.RemoveAll(tmpDir)
			_, _ = fmt.Fprint(os.Stdout, fmt.Sprintf("\033[31mError closing output file: %v\033[0m\n", err))
		}
	}()
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to write output file: %v", err)
	}

	return binaryPath, nil
}

// UpdateGovm updates govm to the latest version
func UpdateGovm() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	tag := make(chan string, 1)
	tagErr := make(chan error, 1)

	_, _ = fmt.Fprint(os.Stdout, "\033[34m\033[1mUpdating govm - Go Version Manager\033[0m\n")

	// Get install directory
	installDir := getInstallDir()
	hasSudo := checkSudo() && installDir == "/usr/local/bin"

	// Get the latest version tag
	go func() {
		latestTag, err := pkg.GetLatestTag()
		if err != nil {
			tagErr <- err
			close(tagErr)
			return
		}
		tag <- latestTag
		close(tag)
	}()

	// Download the latest release
	binaryPath, err := downloadLatestRelease()
	if err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(binaryPath); err != nil {
			_, _ = fmt.Fprint(os.Stdout, fmt.Sprintf("\033[31mError removing temporary file: %v\033[0m\n", err))
		}
	}()

	// Install the binary
	_, _ = fmt.Fprint(os.Stdout, "\033[34mInstalling govm binary...\033[0m\n")
	installPath := filepath.Join(installDir, "govm")

	if hasSudo {
		cmd := exec.Command("sh", "-c", fmt.Sprintf("sudo cp %s %s", binaryPath, installPath))
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install govm binary: %v", err)
		}

		chmodCmd := exec.Command("sh", "-c", fmt.Sprintf("sudo chmod +x %s", installPath))
		if err := chmodCmd.Run(); err != nil {
			return fmt.Errorf("failed to set execute permissions: %v", err)
		}
	} else {
		// Copy the binary to the installation directory
		inFile, err := os.Open(binaryPath)
		if err != nil {
			return fmt.Errorf("failed to open binary file: %v", err)
		}
		defer func() {
			err := inFile.Close()
			if err != nil {
				_, _ = fmt.Fprint(os.Stdout, fmt.Sprintf("\033[31mError closing input file: %v\033[0m\n", err))
			}
		}()

		outFile, err := os.OpenFile(installPath, os.O_CREATE|os.O_WRONLY, 0755)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		defer func() {
			err := outFile.Close()
			if err != nil {
				_, _ = fmt.Fprint(os.Stdout, fmt.Sprintf("\033[31mError closing output file: %v\033[0m\n", err))
			}
		}()

		_, err = io.Copy(outFile, inFile)
		if err != nil {
			return fmt.Errorf("failed to copy binary: %v", err)
		}
	}

	// Wait for the latest tag
	select {
	case err := <-tagErr:
		return err
	case <-ctx.Done():
		return fmt.Errorf("timeout waiting for latest tag")
	case latestTag := <-tag:
		file, err := os.OpenFile(filepath.Join(installDir, "VERSION"), os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to create VERSION file")
		}
		defer func() {
			err := file.Close()
			if err != nil {
				_, _ = fmt.Fprint(os.Stdout, fmt.Sprintf("\033[31mError closing VERSION file: %v\033[0m\n", err))
			}
		}()
		sb := strings.Builder{}
		sb.WriteString(latestTag + "\n")
		sb.WriteString("time: " + time.Now().Format(time.RFC3339))
		_, err = file.WriteString(sb.String())
		if err != nil {
			return fmt.Errorf("failed to write VERSION file")
		}
		_, _ = fmt.Fprint(os.Stdout, fmt.Sprintf("\033[32m\033[1m✓ govm has been successfully updated!\033[0m\n\n"))
		_, _ = fmt.Fprint(
			os.Stdout, fmt.Sprintf("\033[34mFor more information, visit: https://github.com/emmadal/govm\033[0m\n"),
		)
	}

	return nil
}
