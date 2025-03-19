package pkg

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
	REINSTALL      = "Reinstalling"
	INSTALL        = "Installing"
	GO_PATH        = "go"
)

type Tarball struct {
	Url  string
	Ext  string
	Arch string
}

// GetHomeDir returns the home directory.
func GetHomeDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("Unable to get user home directory")
	}
	return homeDir, nil
}

// GetDownloadUrl returns the download URL for a specific Go version.
func GetDownloadUrl(version string) *Tarball {
	switch runtime.GOOS {
	case "windows":
		return &Tarball{Url: fmt.Sprintf("%s/go%s.windows-%s.zip", GO_RELEASE_URL, version, runtime.GOARCH),
			Ext: "zip", Arch: fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)}
	case "linux":
		return &Tarball{Url: fmt.Sprintf("%s/go%s.linux-%s.tar.gz", GO_RELEASE_URL, version, runtime.GOARCH),
			Ext: "tar.gz", Arch: fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)}
	case "darwin":
		return &Tarball{Url: fmt.Sprintf("%s/go%s.darwin-%s.tar.gz", GO_RELEASE_URL, version, runtime.GOARCH),
			Ext: "tar.gz", Arch: fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)}
	}
	return nil
}

// CreateConfigDir creates the config directory.
func CreateConfigDir() error {
	homeDir, err := GetHomeDir()
	if err != nil {
		return err
	}
	configPath := filepath.Join(homeDir, config, goVersions, GO_PATH)

	if err := os.MkdirAll(configPath, 0755); err != nil {
		return fmt.Errorf("Unable to create config directory")
	}
	return nil
}

// CreateCacheDir creates the cache directory.
func CreateCacheDir() (string, error) {
	homeDir, err := GetHomeDir()
	if err != nil {
		return "", err
	}
	cachePath := filepath.Join(homeDir, config, goCache)

	if err := os.MkdirAll(cachePath, 0755); err != nil {
		return "", fmt.Errorf("Unable to create cache directory")
	}
	return cachePath, nil
}

// GetConfigDir returns the config directory.
func GetConfigDir() (string, error) {
	homeDir, err := GetHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, config, goVersions, GO_PATH), nil
}

// GetCacheDir returns the cache directory.
func GetCacheDir() (string, error) {
	homeDir, err := GetHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, config, goCache), nil
}

// CachedGoVersion returns the cached Go version.
func CachedGoVersion(version string) (string, error) {
	cachePath, err := GetCacheDir()
	if err != nil {
		return "", err
	}

	cachedFile := filepath.Join(cachePath, fmt.Sprintf("go%s.%s.%s", version, GetDownloadUrl(version).Arch, GetDownloadUrl(version).Ext))

	if _, err := os.Stat(cachedFile); err == nil {
		return cachedFile, nil
	}
	return "", fmt.Errorf("Unable to find cached file for version %s", version)
}

// DownloadGoVersion downloads a specific Go version.
func DownloadGoVersion(version string) (error, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()

	// Get download URL for the system
	goURL := GetDownloadUrl(version)
	if goURL == nil {
		return fmt.Errorf("Unable to get download URL for the system"), ""
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, goURL.Url, nil)
	if err != nil {
		return fmt.Errorf("Failed to create HTTP request: %s\n", err.Error()), ""
	}

	// Send HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Download failed: %s\n", err.Error()), ""
	}

	// Create cache directory
	cachePath, err := CreateCacheDir()
	if err != nil {
		return err, ""
	}

	// Create file
	file := filepath.Join(cachePath, fmt.Sprintf("go%s.%s.%s", version, goURL.Arch, goURL.Ext))
	f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("Failed to create file %s: %s\n", file, err.Error()), ""
	}

	defer f.Close()
	defer resp.Body.Close()

	// Copy the file to the cache directory
	if resp.ContentLength > 0 {
		// Create progress bar
		ansiColor := fmt.Sprintf("%s===>%s", Green_ANSI, Reset_ANSI)
		bar := progressbar.DefaultBytes(resp.ContentLength, fmt.Sprintf("%s Downloading go%s", ansiColor, version))
		if _, err := io.Copy(io.MultiWriter(bar, f), resp.Body); err != nil {
			return fmt.Errorf("Unable to copy %s: %s\n", file, err.Error()), ""
		}
	}
	return nil, file
}

// UnzipDependency unzips a dependency.
func UnzipDependency(text, file, version string) error {
	ansiColor := fmt.Sprintf("%s===>%s", Blue_ANSI, Reset_ANSI)
	fmt.Printf("%s %s %sgo%s%s\n", ansiColor, text, Green_ANSI, version, Reset_ANSI)

	versionDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	versionFolder := filepath.Join(versionDir, fmt.Sprintf("go%s", version))
	if err := os.MkdirAll(versionFolder, 0755); err != nil {
		return fmt.Errorf("Unable to create version folder %s: %s\n", versionFolder, err.Error())
	}

	switch runtime.GOOS {
	case "linux", "darwin":
		cmd := exec.Command("sh", "-c", fmt.Sprintf("tar -xzf %s --strip-components=1 -C %s", file, versionFolder))
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to unzip %s: %s\n", file, err.Error())
		}
		return nil

	case "windows":
		fmt.Printf("We can't unzip on Windows yet\n")
		return nil
	}
	return nil
}

// UseGoVersion changes the active Go version.
func UseGoVersion(version string) error {
	goVersionDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	goPath := fmt.Sprintf("%s/%s%s/bin", goVersionDir, GO_PATH, version)
	shellConfig, err := GetShellConfig()
	if err != nil {
		return err
	}

	// Check if it's already in use
	currentPath := os.Getenv("PATH")
	if strings.Contains(currentPath, goPath) {
		fmt.Printf("\n✅ go%s is already in use\n", version)
		return nil
	}

	// Set PATH for the current session
	newPath := fmt.Sprintf("%s%c%s", goPath, os.PathListSeparator, currentPath)
	os.Setenv("PATH", newPath)

	// Persist PATH update in shell profile
	err = UpdateShellProfile(goPath)
	if err != nil {
		return fmt.Errorf("Failed to update shell profile: %w", err)
	}

	fmt.Printf("✅ Switched to go%s. Run 'source ~/%s' or restart your terminal to apply permanently.\n", version, shellConfig)
	return nil
}

// UpdateShellProfile updates the shell profile.
func UpdateShellProfile(goPath string) error {
	shellConfig, err := GetShellConfig()
	if err != nil {
		return err
	}

	// Remove old Go path entries
	if err := RemoveOldGoPaths(shellConfig); err != nil {
		return err
	}

	// Append the new Go path
	cmd := exec.Command("sh", "-c", fmt.Sprintf("echo 'export PATH=\"%s:$PATH\"' >> ~/%s", goPath, shellConfig))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to update %s: %w", shellConfig, err)
	}

	return nil
}

// GetShellConfig returns the shell config file.
func GetShellConfig() (string, error) {
	shell := os.Getenv("SHELL")
	if strings.Contains(shell, "zsh") {
		return ".zshrc", nil
	} else if strings.Contains(shell, "bash") {
		return ".bashrc", nil
	}
	return "", fmt.Errorf("Unsupported shell: %s", shell)
}

// RemoveOldGoPaths removes old Go paths from the shell profile.
func RemoveOldGoPaths(shellConfig string) error {
	cmd := exec.Command("sh", "-c", fmt.Sprintf("sed -i '' '/export PATH=.*.govm\\/version/d' ~/%s", shellConfig))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to remove old Go paths from %s: %w", shellConfig, err)
	}
	return nil
}
