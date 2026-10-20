package internal

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/emmadal/govm/pkg"
)

// GitHub endpoints, variables so tests can override them.
var (
	latestReleaseURL   = "https://api.github.com/repos/emmadal/govm/releases/latest"
	releaseDownloadURL = "https://github.com/emmadal/govm/releases/download"
)

// assetName returns the release asset for the current platform.
func assetName() string {
	name := fmt.Sprintf("govm_%s_%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// latestTag returns the tag of the latest govm release.
func latestTag(client *http.Client) (string, error) {
	resp, err := client.Get(latestReleaseURL)
	if err != nil {
		return "", fmt.Errorf("failed to check for updates: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to check for updates: HTTP %s", resp.Status)
	}
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to decode latest release: %w", err)
	}
	if release.TagName == "" {
		return "", errors.New("latest release has no tag")
	}
	return release.TagName, nil
}

// expectedChecksum reads the SHA-256 of asset from the release's
// checksums.txt. It returns "" when the release publishes no checksums.
func expectedChecksum(client *http.Client, tag, asset string) (string, error) {
	resp, err := client.Get(fmt.Sprintf("%s/%s/checksums.txt", releaseDownloadURL, tag))
	if err != nil {
		return "", fmt.Errorf("failed to download checksums: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download checksums: HTTP %s", resp.Status)
	}
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		// sha256sum format: "<hex>  <name>" or "<hex> *<name>"
		fields := strings.Fields(scanner.Text())
		if len(fields) == 2 && filepath.Base(strings.TrimPrefix(fields[1], "*")) == asset {
			return fields[0], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("checksums.txt for %s has no entry for %s", tag, asset)
}

// UpdateGovm replaces the running govm binary with the latest release.
func UpdateGovm(force bool) error {
	client := pkg.NewHTTPClient()
	current := GetVersion()

	pkg.Info("Checking for updates...")
	tag, err := latestTag(client)
	if err != nil {
		return err
	}
	if tag == current && !force {
		pkg.Success("govm %s is already up to date", current)
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to locate the govm binary: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	asset := assetName()
	sum, err := expectedChecksum(client, tag, asset)
	if err != nil {
		return err
	}
	if sum == "" {
		pkg.Warn("Release %s publishes no checksums; skipping verification.", tag)
	}

	// Download next to the binary so the final rename stays on one filesystem.
	tmp := filepath.Join(filepath.Dir(exe), "."+filepath.Base(exe)+".new")
	url := fmt.Sprintf("%s/%s/%s", releaseDownloadURL, tag, asset)
	if err := pkg.Download(client, url, tmp, sum, "govm "+tag); err != nil {
		return permissionHint(err, exe)
	}
	defer func() { _ = os.Remove(tmp) }()
	if err := os.Chmod(tmp, 0o755); err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		// A running executable cannot be replaced on Windows, but it can be
		// renamed out of the way.
		old := exe + ".old"
		_ = os.Remove(old)
		if err := os.Rename(exe, old); err != nil {
			return permissionHint(err, exe)
		}
		if err := os.Rename(tmp, exe); err != nil {
			_ = os.Rename(old, exe)
			return permissionHint(err, exe)
		}
	} else if err := os.Rename(tmp, exe); err != nil {
		return permissionHint(err, exe)
	}

	pkg.Success("govm updated from %s to %s", current, tag)
	return nil
}

func permissionHint(err error, exe string) error {
	if errors.Is(err, fs.ErrPermission) {
		return fmt.Errorf("%w\ngovm is installed at %s, which you cannot write to. Re-run with elevated privileges (e.g. sudo)", err, exe)
	}
	return err
}
