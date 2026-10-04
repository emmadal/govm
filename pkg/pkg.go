package pkg

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
)

// Endpoints of the official Go download server. Variables so tests can
// point them at a local server.
var (
	ReleaseIndexURL = "https://go.dev/dl/?mode=json&include=all"
	DownloadBaseURL = "https://go.dev/dl/"
)

// Release is one entry of the go.dev release index.
type Release struct {
	Version string        `json:"version"` // e.g. "go1.22.0"
	Stable  bool          `json:"stable"`
	Files   []ReleaseFile `json:"files"`
}

// ReleaseFile is one downloadable file of a release.
type ReleaseFile struct {
	Filename string `json:"filename"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	SHA256   string `json:"sha256"`
	Size     int64  `json:"size"`
	Kind     string `json:"kind"` // "archive", "installer" or "source"
}

// Number returns the version without the "go" prefix.
func (r Release) Number() string {
	return strings.TrimPrefix(r.Version, "go")
}

// Archive returns the archive for the given platform.
func (r Release) Archive(goos, goarch string) (ReleaseFile, error) {
	for _, f := range r.Files {
		if f.Kind == "archive" && f.OS == goos && f.Arch == goarch {
			return f, nil
		}
	}
	return ReleaseFile{}, fmt.Errorf("%s has no archive for %s/%s", r.Version, goos, goarch)
}

// NewHTTPClient returns a client suitable for large downloads: it bounds
// connection setup and time-to-first-byte, but not the whole transfer.
func NewHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 30 * time.Second
	return &http.Client{Transport: transport}
}

// FetchReleases downloads the list of all Go releases, newest first.
func FetchReleases(client *http.Client) ([]Release, error) {
	resp, err := client.Get(ReleaseIndexURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Go release list: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch Go release list: HTTP %s", resp.Status)
	}
	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to decode Go release list: %w", err)
	}
	slices.SortStableFunc(releases, func(a, b Release) int {
		return CompareVersions(b.Number(), a.Number())
	})
	return releases, nil
}

// ResolveRelease picks the release requested by input, which is "latest",
// an exact version ("1.22.3", "go1.22rc1") or a minor line ("1.22", which
// resolves to its newest stable patch release). releases must be sorted
// newest first.
func ResolveRelease(input string, releases []Release) (Release, error) {
	if strings.TrimSpace(input) == "latest" {
		for _, r := range releases {
			if r.Stable {
				return r, nil
			}
		}
		return Release{}, fmt.Errorf("no stable Go release found")
	}

	v, err := NormalizeVersion(input)
	if err != nil {
		return Release{}, err
	}
	for _, r := range releases {
		if r.Number() == v {
			return r, nil
		}
	}
	if isMinorLine(v) {
		for _, r := range releases {
			if inMinorLine(r.Number(), v) {
				return r, nil
			}
		}
	}
	return Release{}, fmt.Errorf("go%s not found. Run 'govm ls-remote' to see available versions", v)
}

// Download fetches url into dest, verifying the SHA-256 checksum when
// wantSHA256 is not empty. Data is streamed to dest+".part" and only
// renamed into place once complete and verified.
func Download(client *http.Client, url, dest, wantSHA256, label string) error {
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", label, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download %s: HTTP %s", label, resp.Status)
	}

	part := dest + ".part"
	f, err := os.OpenFile(part, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", part, err)
	}
	defer func() { _ = os.Remove(part) }() // no-op once renamed

	hash := sha256.New()
	bar := progressbar.DefaultBytes(resp.ContentLength, "Downloading "+label)
	_, copyErr := io.Copy(io.MultiWriter(f, hash, bar), resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		return fmt.Errorf("failed to download %s: %w", label, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("failed to write %s: %w", part, closeErr)
	}

	if wantSHA256 != "" {
		if got := hex.EncodeToString(hash.Sum(nil)); !strings.EqualFold(got, wantSHA256) {
			return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", label, wantSHA256, got)
		}
	}
	return os.Rename(part, dest)
}

// FileSHA256 returns the hex-encoded SHA-256 of a file.
func FileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
