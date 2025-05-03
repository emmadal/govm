package pkg

import (
	"encoding/json"
	"fmt"
	"golang.org/x/sync/errgroup"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type Binary struct {
	Versions   []string
	LatestTag  string
	CachedGo   string
	InstallDir string
}

// CachedGoVersion returns the cached Go version.
func (b *Binary) CachedGoVersion(version string) error {
	t := Tarball{}
	dir := Directory{}

	if err := dir.GetDirectories(); err != nil {
		return err
	}

	t.GetURL(version)
	fileName := fmt.Sprintf("go%s.%s", version, t.GetArchWithExt())
	cachedFile := filepath.Join(dir.CacheDir, fileName)

	// Check if the cached file exists
	if _, err := os.Stat(cachedFile); err == nil {
		b.CachedGo = cachedFile
		b.InstallDir = dir.ConfigDir
		return nil
	} else if os.IsNotExist(err) {
		return fmt.Errorf("go version %s not found", version)
	} else {
		return fmt.Errorf("error checking cached file: %v", err)
	}
}

// GoVersionDetails prints the Go version details.
func (b *Binary) GoVersionDetails() error {
	s := ShellConfig{}

	// Try getting the active Go version
	if err := s.GetActiveGoVersion(); err != nil {
		return err
	}
	sb := strings.Builder{}

	// Print the active Go version
	if slices.Contains(b.Versions, s.ActiveVersion) {
		version := TextGreen("→ (Active) - " + s.ActiveVersion + "\n")
		sb.WriteString(version)
	}

	// Print all versions and their status
	for _, name := range b.Versions {
		if name == s.ActiveVersion {
			continue
		}
		sb.WriteString(TextRed("→ (Inactive) - " + name + "\n"))
	}

	BlackPrintln(sb.String())
	return nil
}

// RemoveGoVersion removes a specific Go version.
func (b *Binary) RemoveGoVersion(version string) error {
	// Verify if the Go version is active
	s := ShellConfig{}
	goVersion := fmt.Sprintf("go%s", version)

	if err := s.GetActiveGoVersion(); err != nil {
		return err
	}
	if goVersion == s.ActiveVersion {
		return fmt.Errorf("cannot remove the active Go version")
	}

	// Check if the version exists before proceeding
	if err := b.CachedGoVersion(version); err != nil {
		return err
	}
	// Ask for confirmation
	response := ""
	fmt.Fprintf(os.Stdout, "Do you want to remove go%s? (y/n): ", version)
	if _, err := fmt.Scanln(&response); err != nil {
		return err
	}
	response = strings.ToLower(strings.TrimSpace(response))
	if response != "y" {
		// User cancelled
		RedPrintln("Removal cancelled")
		return nil
	}
	// Remove the Go version
	g := errgroup.Group{}
	g.SetLimit(2)
	fmt.Fprintf(os.Stdout, "Removing go%s...\n", version)

	g.Go(
		func() error {
			return os.Remove(b.CachedGo)
		},
	)
	g.Go(
		func() error {
			return os.RemoveAll(filepath.Join(b.InstallDir, goVersion))
		},
	)
	if err := g.Wait(); err != nil {
		return fmt.Errorf("failed to remove go%s", version)
	}
	GreenPrintln("Successfully removed Go version " + version + "\n")
	return nil
}

// GetLatestTag returns the latest version tag from GitHub
func (b *Binary) GetLatestTag() error {
	var tag struct {
		TagName string `json:"tag_name"`
	}
	// Get the latest version tag
	req, err := http.NewRequest("GET", "https://api.github.com/repos/emmadal/govm/releases/latest", nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request")
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get latest tag")
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if err := json.NewDecoder(resp.Body).Decode(&tag); err != nil {
		return fmt.Errorf("failed to decode latest tag")
	}
	b.LatestTag = tag.TagName
	return nil
}

// GetAllVersions returns all versions downloaded
func (b *Binary) GetAllVersions() error {
	var versionPath []string
	homeDir, _ := os.UserHomeDir()
	entries, err := os.ReadDir(filepath.Join(homeDir, ".govm", "versions", "go"))
	if err != nil {
		return fmt.Errorf("failed to read binaries directory")
	}
	if len(entries) == 0 {
		return fmt.Errorf("no versions found. Install a version with 'govm install <version>")
	}
	for _, entry := range entries {
		if entry.IsDir() {
			versionPath = append(versionPath, entry.Name())
		}
	}
	b.Versions = versionPath
	return nil
}
