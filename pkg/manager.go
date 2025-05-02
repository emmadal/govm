package pkg

import (
	"fmt"
	"github.com/schollz/progressbar/v3"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

type Tarball struct {
	Url  string
	File *os.File
	Arch string
}

// GetArchWithExt returns the architecture and extension for the current OS.
func (t *Tarball) GetArchWithExt() string {
	var ext string
	os := runtime.GOOS
	if os == "windows" {
		ext = "zip"
	} else {
		ext = "tar.gz"
	}
	return fmt.Sprintf("%s-%s.%s", runtime.GOOS, runtime.GOARCH, ext)
}

// GetURL returns the download URL for the current OS.
func (t *Tarball) GetURL(version string) {
	var ext string
	if runtime.GOOS == "windows" {
		ext = "zip"
	} else {
		ext = "tar.gz"
	}
	arch := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	t.Url = fmt.Sprintf("https://golang.org/dl/go%s.%s.%s", version, arch, ext)
}

// DownloadGoVersion downloads a specific Go version.
func (t *Tarball) DownloadGoVersion(version, cachePath string) error {
	// Get download URL
	t.GetURL(version)

	// Create HTTP request
	req, err := http.NewRequest(http.MethodGet, t.Url, nil)
	if err != nil {
		return err
	}

	// Send HTTP request
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("no version go%s found. Please use a valid version number", version)
	}

	// Create a file
	fileName := fmt.Sprintf("go%s.%s", version, t.GetArchWithExt())
	file := filepath.Join(cachePath, fileName)
	f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file %s", f)
	}
	defer f.Close()

	// Copy the file to the cache directory
	if resp.ContentLength > 0 {
		// Create a progress bar
		BlackPrintln(fmt.Sprintf("⚡️Downloading go%s", version) + "\n")
		bar := progressbar.DefaultBytes(resp.ContentLength)
		if _, err := io.Copy(io.MultiWriter(bar, f), resp.Body); err != nil {
			return fmt.Errorf("failed to copy %s", file)
		}
		t.File = f
	}
	return nil
}

// InstallVersion installs a specific Go version.
func (t *Tarball) InstallVersion(file, version, versionDir string) error {

	BlackPrintln(fmt.Sprintf("⏳ Installing go%s", version) + "\n")

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
	case "windows":
		// For Windows 10 build 17,063 or later which has built-in tar
		cmd := exec.Command(
			"powershell", "-Command",
			fmt.Sprintf("tar -xzf '%s' --strip-components=1 -C '%s'", file, versionFolder),
		)
		if err := cmd.Run(); err != nil {
			// Try to fall back to 7-Zip if available (common on Windows)
			sevenZipCmd := exec.Command(
				"powershell", "-Command",
				fmt.Sprintf(
					"& { $env:tmp='%s'; 7z x '%s' -o\"$env:tmp\" -y; Get-ChildItem \"$env:tmp\" | Select-Object -First 1 | Get-ChildItem | Move-Item -Destination '%s'; }",
					os.TempDir(), file, versionFolder,
				),
			)
			if err := sevenZipCmd.Run(); err != nil {
				return fmt.Errorf("failed to unzip %s (tried both tar and 7z)", file)
			}
		}
		return nil
	}
	return nil
}

// UseGoVersion sets the current Go version.
func (t *Tarball) UseGoVersion(version, goVersionDir string) error {
	goPath := filepath.Join(goVersionDir, fmt.Sprintf("go%s", version), "bin")
	shellConfig, err := GetShellConfig()
	if err != nil {
		return err
	}

	// Set PATH for the current session
	newPath := fmt.Sprintf("%s%c%s", goPath, os.PathListSeparator, os.Getenv("PATH"))
	if err := os.Setenv("PATH", newPath); err != nil {
		return err
	}

	// Persist PATH update in shell profile
	if err = UpdateShellProfile(goPath); err != nil {
		return err
	}

	// Print a success message
	GreenPrintln(
		"✅ Switched to go" + version + ". " +
			"Run 'source ~/" + shellConfig + "' or restart your terminal to apply permanently." + "\n",
	)

	return nil
}
