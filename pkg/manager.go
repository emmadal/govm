package pkg

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/schollz/progressbar/v3"
)

// DefaultGoBaseURL is where Go archives are downloaded from.
const DefaultGoBaseURL = "https://golang.org/dl"

type Tarball struct {
	Url  string
	Arch string
	// BaseURL overrides DefaultGoBaseURL.
	BaseURL string
	// Downloader overrides the default download settings.
	Downloader *Downloader
}

// GetArchWithExt returns the architecture and extension for the current OS.
func (t *Tarball) GetArchWithExt() string {
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("%s-%s.%s", runtime.GOOS, runtime.GOARCH, ext)
}

// GetURL sets the download URL for the current OS.
func (t *Tarball) GetURL(version string) {
	t.Url = fmt.Sprintf("%s/go%s.%s", cmp.Or(t.BaseURL, DefaultGoBaseURL), version, t.GetArchWithExt())
}

// DownloadGoVersion downloads a specific Go version into cachePath and returns
// the path of the archive. A stalled or interrupted download is retried; if it
// still fails, no partial archive is left in the cache.
func (t *Tarball) DownloadGoVersion(ctx context.Context, version, cachePath string) (string, error) {
	t.GetURL(version)
	file := filepath.Join(cachePath, fmt.Sprintf("go%s.%s", version, t.GetArchWithExt()))

	var d Downloader
	if t.Downloader != nil {
		d = *t.Downloader
	}
	if d.Progress == nil {
		d.Progress = func(_ int, total int64) io.Writer {
			// A total of -1 renders a spinner instead of a percentage.
			return progressbar.DefaultBytes(total)
		}
	}
	if d.OnRetry == nil {
		d.OnRetry = func(attempt, maxAttempts int, err error) {
			RedPrintln(fmt.Sprintf("\n⚠️  Download interrupted: %v\n", err))
			BlackPrintln(fmt.Sprintf("🔁 Retrying (%d/%d)...\n\n", attempt, maxAttempts))
		}
	}

	BlackPrintln(fmt.Sprintf("⚡️Downloading go%s", version) + "\n")
	if err := d.Download(ctx, t.Url, file); err != nil {
		fmt.Println() // end the progress bar line before the error is printed
		if statusErr, ok := errors.AsType[*HTTPStatusError](err); ok && statusErr.StatusCode == http.StatusNotFound {
			return "", fmt.Errorf("no version go%s found. Please use a valid version number", version)
		}
		return "", fmt.Errorf("failed to download go%s: %w", version, err)
	}
	return file, nil
}

// InstallVersion installs a specific Go version from the archive at file. If
// the extraction fails, both the half-extracted version folder and the archive
// are removed so that neither is mistaken for a valid install.
func (t *Tarball) InstallVersion(ctx context.Context, file, version, versionDir string) error {
	BlackPrintln(fmt.Sprintf("⏳ Installing go%s", version) + "\n")

	versionFolder := filepath.Join(versionDir, fmt.Sprintf("go%s", version))
	if err := os.MkdirAll(versionFolder, 0755); err != nil {
		return fmt.Errorf("unable to create version folder %s", versionFolder)
	}

	if err := extract(ctx, file, versionFolder); err != nil {
		_ = os.RemoveAll(versionFolder)
		_ = os.Remove(file)
		return fmt.Errorf("failed to unzip %s: %w", file, err)
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
