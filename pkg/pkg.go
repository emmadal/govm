package pkg

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
)

const (
	config         = ".govm"
	goVersions     = "versions"
	goCache        = ".cache"
	GO_RELEASE_URL = "https://golang.org/dl"
	Green_ANSI     = "\033[32m"
	Reset_ANSI     = "\033[0m"
	Blue_ANSI      = "\033[34m"
	Red_ANSI       = "\033[31m"
	REINSTALL      = "Reinstalling"
	INSTALL        = "Installing"
	GO_PATH        = "go"
)

type Tarball struct {
	Url  string
	Arch string
	Ext  string
}

// GetURLByOS returns the download URL for a specific Go version.
func (t *Tarball) GetURLByOS(version string) string {
	if runtime.GOOS == "windows" {
		t.Ext = ".zip"
	} else {
		t.Ext = ".tar.gz"
	}
	t.Arch = fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	t.Url = fmt.Sprintf("%s/go%s.%s%s", GO_RELEASE_URL, version, t.Arch, t.Ext)
	return t.Url
}

// GetArch returns the architecture.
func (t *Tarball) GetArch() string {
	return t.Arch
}

// GetUrl returns the download URL.
func (t *Tarball) GetUrl() string {
	return t.Url
}

// GetExt returns the extension.
func (t *Tarball) GetExt() string {
	return t.Ext
}

// CachedGoVersion returns the cached Go version.
func CachedGoVersion(version string) (string, error) {
	cachePath, err := GetCacheDir()
	if err != nil {
		return "", err
	}
	t := Tarball{}

	url := t.GetURLByOS(version)
	if url == "" {
		return "", fmt.Errorf("unable to get download URL for the system")
	}

	fileName := fmt.Sprintf("go%s.%s%s", version, t.GetArch(), t.GetExt())
	cachedFile := filepath.Join(cachePath, fileName)

	// Check if the cached file exists
	if _, err := os.Stat(cachedFile); err == nil {
		return cachedFile, nil
	} else if os.IsNotExist(err) {
		return "", fmt.Errorf("cached file for Go version %s not found", version)
	} else {
		return "", fmt.Errorf("error checking cached file: %v", err)
	}
}

// DownloadGoVersion downloads a specific Go version.
func DownloadGoVersion(version string) (string, error) {
	// Get download URL for the system
	t := Tarball{}
	goURLUrl := t.GetURLByOS(version)
	if goURLUrl == "" {
		return "", fmt.Errorf("unable to get download URL for the system")
	}

	// Create HTTP request
	req, err := http.NewRequest(http.MethodGet, goURLUrl, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %s", err.Error())
	}

	// Send HTTP request
	client := &http.Client{Timeout: 55 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send HTTP request: %s", err.Error())
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("no version go%s found. please check the version number", version)
	}

	// Create cache directory
	cachePath, err := CreateCacheDir()
	if err != nil {
		return "", err
	}

	// Create file
	file := filepath.Join(cachePath, fmt.Sprintf("go%s.%s%s", version, t.GetArch(), t.GetExt()))
	f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to create file %s", file)
	}
	defer f.Close()

	// Copy the file to the cache directory
	if resp.ContentLength > 0 {
		// Create progress bar
		ansiColor := fmt.Sprintf("%s===>%s", Green_ANSI, Reset_ANSI)
		bar := progressbar.DefaultBytes(resp.ContentLength, fmt.Sprintf("%s Downloading go%s", ansiColor, version))
		if _, err := io.Copy(io.MultiWriter(bar, f), resp.Body); err != nil {
			return "", fmt.Errorf("failed to copy %s", file)
		}
	}
	return file, nil
}

// UnzipDependency unzips a dependency.
func UnzipDependency(text, file, version string) error {
	ansiColor := fmt.Sprintf("%s===>%s", Blue_ANSI, Reset_ANSI)
	fmt.Fprintf(os.Stdout, "%s %s %sgo%s%s\n", ansiColor, text, Green_ANSI, version, Reset_ANSI)

	versionDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	versionFolder := filepath.Join(versionDir, fmt.Sprintf("go%s", version))
	if err := os.MkdirAll(versionFolder, 0755); err != nil {
		return fmt.Errorf("unable to create version folder %s", versionFolder)
	}

	switch runtime.GOOS {
	case "linux", "darwin":
		cmd := exec.Command("sh", "-c", fmt.Sprintf("tar -xzf %s --strip-components=1 -C %s", file, versionFolder))
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to unzip %s", file)
		}
		return nil
	default:
		return fmt.Errorf("Unsupported OS: %s", runtime.GOOS)
	}
}

// ListGoVersions lists installed Go versions.
func ListGoVersions() error {
	cachePath, err := GetCacheDir()
	if err != nil {
		return err
	}

	files, err := os.ReadDir(cachePath)
	if err != nil {
		return err
	}

	// Buffer writer
	bufWriter := bufio.NewWriter(os.Stdout)
	defer func() {
		if flushErr := bufWriter.Flush(); flushErr != nil {
			fmt.Fprintf(os.Stderr, "failed to flush buffer: %s", flushErr.Error())
		}
	}()

	if len(files) == 0 {
		return fmt.Errorf("No Go versions found. Install a version with 'govm install <version>'")
	}

	// Get architecture
	arch := fmt.Sprintf(".%s-%s", runtime.GOOS, runtime.GOARCH)

	activeGoVersion, err := GetActiveGoVersion()
	if err != nil {
		return err
	}

	// Loop through files
	for _, file := range files {
		if strings.HasPrefix(file.Name(), GO_PATH) && strings.Contains(file.Name(), arch) {
			version := strings.Split(file.Name(), arch)[0]

			color := Red_ANSI
			status := "N/A - Downloaded"

			if version == activeGoVersion {
				color = Green_ANSI
				status = "(Active)"
			}

			line := fmt.Sprintf("%s→ %s %s%s\n", color, version, status, Reset_ANSI)
			if _, err := bufWriter.WriteString(line); err != nil {
				return err
			}
		}
	}

	return nil
}

// RemoveGoVersion removes a specific Go version.
func RemoveGoVersion(version string) error {
	// Verify if the Go version is active
	activeGoVersion, err := GetActiveGoVersion()
	if err != nil {
		return err
	}
	if version == activeGoVersion {
		return fmt.Errorf("%s is currently active. Please switch to another version", version)
	}

	// Check if the version exists before proceeding
	cachedFile, err := CachedGoVersion(version)
	if err != nil {
		return fmt.Errorf("Go version %s is not installed", version)
	}

	// Verify the installation directory exists
	versionDir, err := GetConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %v", err)
	}

	versionFolder := filepath.Join(versionDir, fmt.Sprintf("go%s", version))
	if _, err := os.Stat(versionFolder); os.IsNotExist(err) {
		// Only the cached file exists, not the installation
		fmt.Fprintf(os.Stdout, "%sWarning: Installation directory for Go %s not found, but cached file exists%s\n",
			Red_ANSI, version, Reset_ANSI)
	}

	// Ask for confirmation
	response := ""
	fmt.Fprintf(os.Stdout, "Are you sure you want to remove Go version %s? (y/n): ", version)
	fmt.Scanln(&response)

	response = strings.ToLower(strings.TrimSpace(response))
	if response != "y" {
		fmt.Fprintf(os.Stdout, "%sCancelling removal of Go version %s%s\n", Red_ANSI, version, Reset_ANSI)
		return nil
	}

	fmt.Fprintf(os.Stdout, "Removing Go version %s...\n", version)

	// Remove cached file
	if err := os.Remove(cachedFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cached file %s: %v", cachedFile, err)
	}

	// Remove version folder
	if err := os.RemoveAll(versionFolder); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove version folder %s: %v", versionFolder, err)
	}

	fmt.Fprintf(os.Stdout, "%sGo version %s has been removed successfully.%s\n", Green_ANSI, version, Reset_ANSI)
	return nil
}
