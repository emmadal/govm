package pkg

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var testReleases = []Release{
	{Version: "go1.23rc1", Stable: false},
	{Version: "go1.22.10", Stable: true},
	{Version: "go1.22.3", Stable: true},
	{Version: "go1.21.5", Stable: true},
	{Version: "go1.20", Stable: true},
}

func TestResolveRelease(t *testing.T) {
	tests := map[string]string{
		"latest":   "go1.22.10",
		"1.22":     "go1.22.10",
		"go1.22.3": "go1.22.3",
		"1.23rc1":  "go1.23rc1",
		"1.20":     "go1.20",
		"go1.21":   "go1.21.5",
	}
	for in, want := range tests {
		got, err := ResolveRelease(in, testReleases)
		if err != nil || got.Version != want {
			t.Errorf("ResolveRelease(%q) = %q, %v; want %q", in, got.Version, err, want)
		}
	}
	for _, in := range []string{"1.19", "1.22.99", "nope"} {
		if _, err := ResolveRelease(in, testReleases); err == nil {
			t.Errorf("ResolveRelease(%q) succeeded, want error", in)
		}
	}
}

func TestReleaseArchive(t *testing.T) {
	r := Release{Version: "go1.22.0", Files: []ReleaseFile{
		{Filename: "go1.22.0.src.tar.gz", Kind: "source"},
		{Filename: "go1.22.0.linux-amd64.tar.gz", OS: "linux", Arch: "amd64", Kind: "archive"},
		{Filename: "go1.22.0.windows-amd64.msi", OS: "windows", Arch: "amd64", Kind: "installer"},
		{Filename: "go1.22.0.windows-amd64.zip", OS: "windows", Arch: "amd64", Kind: "archive"},
	}}
	if f, err := r.Archive("windows", "amd64"); err != nil || f.Filename != "go1.22.0.windows-amd64.zip" {
		t.Errorf("Archive(windows) = %q, %v", f.Filename, err)
	}
	if _, err := r.Archive("plan9", "arm"); err == nil {
		t.Error("expected error for unsupported platform")
	}
}

func TestFetchReleasesSortsAndChecksStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("fail") != "" {
			http.Error(w, "nope", http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`[{"version":"go1.9.2","stable":true},{"version":"go1.22.0","stable":true},{"version":"go1.10","stable":true}]`))
	}))
	defer srv.Close()
	defer func(old string) { ReleaseIndexURL = old }(ReleaseIndexURL)

	ReleaseIndexURL = srv.URL
	releases, err := FetchReleases(srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, r := range releases {
		got = append(got, r.Version)
	}
	if strings.Join(got, ",") != "go1.22.0,go1.10,go1.9.2" {
		t.Errorf("order = %v", got)
	}

	ReleaseIndexURL = srv.URL + "?fail=1"
	if _, err := FetchReleases(srv.Client()); err == nil {
		t.Error("expected error on HTTP 429")
	}
}

func TestDownloadVerifiesChecksum(t *testing.T) {
	body := []byte("go distribution bytes")
	sum := sha256.Sum256(body)
	good := hex.EncodeToString(sum[:])
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "go.tar.gz")

	if err := Download(srv.Client(), srv.URL+"/f", dest, strings.Repeat("0", 64), "test"); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("bad checksum: got %v", err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatal("file kept after checksum mismatch")
	}

	if err := Download(srv.Client(), srv.URL+"/missing", dest, good, "test"); err == nil {
		t.Fatal("expected error on 404")
	}

	if err := Download(srv.Client(), srv.URL+"/f", dest, good, "test"); err != nil {
		t.Fatal(err)
	}
	if got, _ := FileSHA256(dest); got != good {
		t.Fatalf("downloaded checksum %s", got)
	}
	if matches, _ := filepath.Glob(filepath.Join(dir, "*.part")); len(matches) > 0 {
		t.Fatalf("partial files left: %v", matches)
	}
}
