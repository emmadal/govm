package pkg

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
)

// quietTarball returns a Tarball pointed at baseURL, with fast retries and no
// progress bar.
func quietTarball(baseURL string) *Tarball {
	d := fastDownloader()
	d.Progress = func(int, int64) io.Writer { return nil }
	d.OnRetry = func(int, int, error) {}
	return &Tarball{BaseURL: baseURL, Downloader: d}
}

func TestDownloadGoVersion_SavesArchive(t *testing.T) {
	wantPath := "/go1.25.0." + (&Tarball{}).GetArchWithExt()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != wantPath {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	cache := t.TempDir()
	file, err := quietTarball(srv.URL).DownloadGoVersion(t.Context(), "1.25.0", cache)
	if err != nil {
		t.Fatalf("DownloadGoVersion: %v", err)
	}
	if want := filepath.Join(cache, strings.TrimPrefix(wantPath, "/")); file != want {
		t.Errorf("file = %s, want %s", file, want)
	}
	assertFile(t, file, payload)
}

// Regression: a response without Content-Length used to write nothing and
// crash the install command on a nil file.
func TestDownloadGoVersion_NoContentLength(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.(http.Flusher).Flush()
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	file, err := quietTarball(srv.URL).DownloadGoVersion(t.Context(), "1.25.0", t.TempDir())
	if err != nil {
		t.Fatalf("DownloadGoVersion: %v", err)
	}
	assertFile(t, file, payload)
}

func TestDownloadGoVersion_NotFound(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.NotFound(w, r)
	}))
	defer srv.Close()

	cache := t.TempDir()
	_, err := quietTarball(srv.URL).DownloadGoVersion(t.Context(), "9.99.9", cache)
	if err == nil || !strings.Contains(err.Error(), "no version go9.99.9 found") {
		t.Fatalf("err = %v, want \"no version go9.99.9 found\"", err)
	}
	if got := requests.Load(); got != 1 {
		t.Errorf("requests = %d, want 1", got)
	}
	assertEmptyDir(t, cache)
}

// Regression: every non-200 status used to be reported as an unknown version.
func TestDownloadGoVersion_ServerErrorIsNotUnknownVersion(t *testing.T) {
	for _, status := range []int{http.StatusInternalServerError, http.StatusTooManyRequests} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var requests atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.WriteHeader(status)
			}))
			defer srv.Close()

			cache := t.TempDir()
			_, err := quietTarball(srv.URL).DownloadGoVersion(t.Context(), "1.25.0", cache)
			if err == nil {
				t.Fatal("expected an error")
			}
			if strings.Contains(err.Error(), "no version") {
				t.Errorf("err = %q, must not claim the version is unknown", err)
			}
			if !strings.Contains(err.Error(), fmt.Sprint(status)) || !strings.Contains(err.Error(), "after 3 attempts") {
				t.Errorf("err = %q, want the HTTP status and \"after 3 attempts\"", err)
			}
			if got := requests.Load(); got != 3 {
				t.Errorf("requests = %d, want 3", got)
			}
			assertEmptyDir(t, cache)
		})
	}
}

func TestDownloadGoVersion_TimeoutLeavesCacheEmpty(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte("partial"))
		w.(http.Flusher).Flush()
		stall(r)
	}))
	defer srv.Close()

	cache := t.TempDir()
	_, err := quietTarball(srv.URL).DownloadGoVersion(t.Context(), "1.25.0", cache)
	if err == nil || !strings.Contains(err.Error(), "after 3 attempts") {
		t.Fatalf("err = %v, want \"after 3 attempts\"", err)
	}
	if got := requests.Load(); got != 3 {
		t.Errorf("requests = %d, want 3", got)
	}
	assertEmptyDir(t, cache)
}

func TestInstallVersion_CorruptArchiveIsCleanedUp(t *testing.T) {
	cache, versions := t.TempDir(), t.TempDir()
	archive := filepath.Join(cache, "go1.25.0."+(&Tarball{}).GetArchWithExt())
	if err := os.WriteFile(archive, []byte("this is not an archive"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := (&Tarball{}).InstallVersion(t.Context(), archive, "1.25.0", versions); err == nil {
		t.Fatal("expected an extraction error")
	}
	assertEmptyDir(t, versions)
	assertEmptyDir(t, cache)
}

func TestInstallVersion_ExtractsArchive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("builds a tar.gz; Windows installs from zip")
	}
	// A path with a space guards against shell word-splitting.
	cache := filepath.Join(t.TempDir(), "my cache")
	versions := filepath.Join(t.TempDir(), "my versions")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	content := []byte("#!/bin/sh\n")
	if err := tw.WriteHeader(&tar.Header{Name: "go/bin/go", Mode: 0o755, Size: int64(len(content))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(cache, "go1.25.0.tar.gz")
	if err := os.WriteFile(archive, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := (&Tarball{}).InstallVersion(t.Context(), archive, "1.25.0", versions); err != nil {
		t.Fatalf("InstallVersion: %v", err)
	}
	assertFile(t, filepath.Join(versions, "go1.25.0", "bin", "go"), content)
	if _, err := os.Stat(archive); err != nil {
		t.Errorf("archive must be kept after a successful install: %v", err)
	}
}
