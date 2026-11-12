package internal

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emmadal/govm/pkg"
)

var binaryPayload = bytes.Repeat([]byte("govm"), 1024)

// setup points the updater at srv with fast retries and an isolated temporary
// directory, which it returns.
func setup(t *testing.T, srv *httptest.Server) string {
	t.Helper()
	tmp := t.TempDir()
	for _, key := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(key, tmp)
	}

	oldURL, oldDownloader := releaseBaseURL, newDownloader
	releaseBaseURL = srv.URL
	newDownloader = func() *pkg.Downloader {
		return &pkg.Downloader{Mode: 0755, IdleTimeout: 300 * time.Millisecond, RetryDelay: time.Millisecond}
	}
	t.Cleanup(func() { releaseBaseURL, newDownloader = oldURL, oldDownloader })
	return tmp
}

func stall(r *http.Request) { <-r.Context().Done() }

func TestDownloadLatestRelease_RetriesThenSucceeds(t *testing.T) {
	var requests atomic.Int32
	var path atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path.Store(r.URL.Path)
		if requests.Add(1) < 3 {
			stall(r)
			return
		}
		_, _ = w.Write(binaryPayload)
	}))
	defer srv.Close()
	setup(t, srv)

	binaryPath, err := downloadLatestRelease(t.Context())
	if err != nil {
		t.Fatalf("downloadLatestRelease: %v", err)
	}
	if got := requests.Load(); got != 3 {
		t.Errorf("requests = %d, want 3", got)
	}
	if got := path.Load().(string); !strings.HasPrefix(got, "/govm_"+runtime.GOOS+"_"+runtime.GOARCH) {
		t.Errorf("requested path = %s", got)
	}
	got, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, binaryPayload) {
		t.Errorf("binary: got %d bytes, want %d", len(got), len(binaryPayload))
	}
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(binaryPath)
		if info.Mode().Perm()&0o100 == 0 {
			t.Errorf("binary mode = %v, want executable", info.Mode())
		}
	}
}

func TestDownloadLatestRelease_GivesUpAndCleansUp(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte("partial"))
		w.(http.Flusher).Flush()
		stall(r)
	}))
	defer srv.Close()
	tmp := setup(t, srv)

	_, err := downloadLatestRelease(t.Context())
	if err == nil || !strings.Contains(err.Error(), "after 3 attempts") {
		t.Fatalf("err = %v, want \"after 3 attempts\"", err)
	}
	if got := requests.Load(); got != 3 {
		t.Errorf("requests = %d, want 3", got)
	}
	entries, _ := os.ReadDir(tmp)
	for _, entry := range entries {
		t.Errorf("leftover in temporary directory: %s", entry.Name())
	}
}

func TestDownloadLatestRelease_NotFoundIsNotRetried(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.NotFound(w, r)
	}))
	defer srv.Close()
	tmp := setup(t, srv)

	if _, err := downloadLatestRelease(t.Context()); err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("err = %v, want it to mention 404", err)
	}
	if got := requests.Load(); got != 1 {
		t.Errorf("requests = %d, want 1", got)
	}
	if entries, _ := os.ReadDir(tmp); len(entries) != 0 {
		t.Errorf("temporary directory not cleaned: %d entries", len(entries))
	}
}

// Regression: copying over a larger file used to leave its tail in place.
func TestCopyFile_ReplacesLargerDestination(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "src"), filepath.Join(dir, "dst")
	if err := os.WriteFile(src, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, bytes.Repeat([]byte("old"), 1000), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile: %v", err)
	}
	got, _ := os.ReadFile(dst)
	if string(got) != "new" {
		t.Errorf("dst holds %d bytes, want exactly \"new\"", len(got))
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 2 {
		t.Errorf("want only src and dst, got %d entries", len(entries))
	}
}
