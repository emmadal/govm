package pkg

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

	if len(files) == 0 {
		return fmt.Errorf("No Go versions found. Install a version with 'govm install <version>'")
	}

	arch := fmt.Sprintf(".%s-%s", runtime.GOOS, runtime.GOARCH)

	// Try getting the active Go version, but don't exit if it fails
	activeGoVersion, err := GetActiveGoVersion()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Warning: Could not determine active Go version")
		activeGoVersion = ""
	}

	// Match Go version format
	re := regexp.MustCompile(`go(\d+\.\d+\.\d+([a-z]+\d+)?)`)
	sb := strings.Builder{}
	found := false

	// Loop through files
	for _, file := range files {
		name := file.Name()
		if strings.HasPrefix(name, GO_PATH) && strings.Contains(name, arch) {
			matches := re.FindStringSubmatch(name)
			if len(matches) > 1 {
				version := matches[1]
				color := Red_ANSI
				status := "N/A - Downloaded"

				if version == activeGoVersion {
					color = Green_ANSI
					status = "(Active)"
				}
				sb.WriteString(fmt.Sprintf("%s→ %s %s%s\n", color, version, status, Reset_ANSI))
				found = true
			}
		}
	}

	// Print the list of Go versions
	if !found {
		return fmt.Errorf("No matching Go versions found for architecture %s", arch)
	}
	fmt.Fprint(os.Stdout, sb.String())
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
	_, err = fmt.Scanln(&response)
	if err != nil {
		return fmt.Errorf("failed to read input: %v", err)
	}

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

// ConfirmRemoval asks for user confirmation to remove govm
func ConfirmRemoval() (bool, error) {
	var sb strings.Builder

	// Build the confirmation message in memory using strings.Builder
	fmt.Fprintf(&sb, "%s%s%s\n", Red_ANSI, "This will completely remove govm from your system, including:", Reset_ANSI)
	fmt.Fprintln(&sb, "  - The govm binary")
	fmt.Fprintln(&sb, "  - All installed Go versions managed by govm")
	fmt.Fprintln(&sb, "  - All govm configuration files")
	fmt.Fprintln(os.Stdout, sb.String())

	// Ask for user confirmation
	var reply string

	// Add the prompt to the builder
	fmt.Fprint(os.Stdout, "Are you sure you want to proceed? (y/n): ")
	_, err := fmt.Scanln(&reply)
	if err != nil {
		return false, fmt.Errorf("failed to read input: %v", err)
	}

	reply = strings.ToLower(strings.TrimSpace(reply))

	// Handle valid responses
	if reply == "y" {
		return true, nil
	} else if reply == "n" {
		fmt.Fprintln(os.Stdout, "Removal cancelled.")
		return false, nil
	} else {
		// Invalid input, re-prompt
		fmt.Fprintln(os.Stdout, "Invalid input. Please enter 'y' to confirm or 'n' to cancel.")
		return false, nil
	}
}

// GetLatestTag returns the latest version tag from GitHub
func GetLatestTag() (string, error) {
	var latestTag struct {
		TagName string `json:"tag_name"`
	}
	// Get the latest version tag
	req, err := http.NewRequest("GET", "https://api.github.com/repos/emmadal/govm/releases/latest", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get latest tag")
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&latestTag); err != nil {
		return "", fmt.Errorf("failed to decode latest tag")
	}

	return strings.TrimSpace(latestTag.TagName), nil
}
