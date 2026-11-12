package pkg

import (
	"archive/tar"
	"archive/zip"
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

// goTarGz builds a .tar.gz archive laid out like a Go release.
func goTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(content))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// goZip builds a .zip archive laid out like a Go release.
func goZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

var archiveFormats = []struct {
	ext   string
	build func(*testing.T, map[string]string) []byte
}{
	{"tar.gz", goTarGz},
	{"zip", goZip},
}

// Both formats are extracted on every platform: the code is the same
// everywhere, only the format a platform downloads differs.
func TestInstallVersion_CorruptArchiveIsCleanedUp(t *testing.T) {
	for _, format := range archiveFormats {
		t.Run(format.ext, func(t *testing.T) {
			cache, versions := t.TempDir(), t.TempDir()
			archive := filepath.Join(cache, "go1.25.0."+format.ext)
			if err := os.WriteFile(archive, []byte("this is not an archive"), 0o644); err != nil {
				t.Fatal(err)
			}

			if err := (&Tarball{}).InstallVersion(t.Context(), archive, "1.25.0", versions); err == nil {
				t.Fatal("expected an extraction error")
			}
			assertEmptyDir(t, versions)
			assertEmptyDir(t, cache)
		})
	}
}

// A truncated archive starts like a valid one and must still be rejected.
func TestInstallVersion_TruncatedArchiveIsCleanedUp(t *testing.T) {
	for _, format := range archiveFormats {
		t.Run(format.ext, func(t *testing.T) {
			cache, versions := t.TempDir(), t.TempDir()
			data := format.build(t, map[string]string{"go/bin/go": strings.Repeat("binary", 4096)})
			archive := filepath.Join(cache, "go1.25.0."+format.ext)
			if err := os.WriteFile(archive, data[:len(data)/2], 0o644); err != nil {
				t.Fatal(err)
			}

			if err := (&Tarball{}).InstallVersion(t.Context(), archive, "1.25.0", versions); err == nil {
				t.Fatal("expected an extraction error")
			}
			assertEmptyDir(t, versions)
			assertEmptyDir(t, cache)
		})
	}
}

func TestInstallVersion_ExtractsArchive(t *testing.T) {
	for _, format := range archiveFormats {
		t.Run(format.ext, func(t *testing.T) {
			// Paths with a space guard against any word-splitting.
			cache := filepath.Join(t.TempDir(), "my cache")
			versions := filepath.Join(t.TempDir(), "my versions")
			if err := os.MkdirAll(cache, 0o755); err != nil {
				t.Fatal(err)
			}
			archive := filepath.Join(cache, "go1.25.0."+format.ext)
			data := format.build(t, map[string]string{
				"go/VERSION":            "go1.25.0\n",
				"go/bin/go":             "#!/bin/sh\n",
				"go/src/fmt/doc.go":     "package fmt\n",
				"go/pkg/tool/x/compile": "tool",
			})
			if err := os.WriteFile(archive, data, 0o644); err != nil {
				t.Fatal(err)
			}

			if err := (&Tarball{}).InstallVersion(t.Context(), archive, "1.25.0", versions); err != nil {
				t.Fatalf("InstallVersion: %v", err)
			}
			// The top-level "go" directory of the archive is dropped.
			install := filepath.Join(versions, "go1.25.0")
			assertFile(t, filepath.Join(install, "VERSION"), []byte("go1.25.0\n"))
			assertFile(t, filepath.Join(install, "bin", "go"), []byte("#!/bin/sh\n"))
			assertFile(t, filepath.Join(install, "src", "fmt", "doc.go"), []byte("package fmt\n"))
			assertFile(t, filepath.Join(install, "pkg", "tool", "x", "compile"), []byte("tool"))
			if _, err := os.Stat(archive); err != nil {
				t.Errorf("archive must be kept after a successful install: %v", err)
			}
			if format.ext == "tar.gz" && runtime.GOOS != "windows" {
				info, err := os.Stat(filepath.Join(install, "bin", "go"))
				if err != nil {
					t.Fatal(err)
				}
				if info.Mode().Perm()&0o100 == 0 {
					t.Errorf("bin/go mode = %v, want executable", info.Mode())
				}
			}
		})
	}
}

// An archive must never write outside of the version folder.
func TestInstallVersion_EntriesCannotEscape(t *testing.T) {
	for _, format := range archiveFormats {
		t.Run(format.ext, func(t *testing.T) {
			base := t.TempDir()
			cache, versions := filepath.Join(base, "cache"), filepath.Join(base, "versions")
			if err := os.MkdirAll(cache, 0o755); err != nil {
				t.Fatal(err)
			}
			archive := filepath.Join(cache, "go1.25.0."+format.ext)
			data := format.build(t, map[string]string{"go/a/../../../../escaped": "evil"})
			if err := os.WriteFile(archive, data, 0o644); err != nil {
				t.Fatal(err)
			}

			if err := (&Tarball{}).InstallVersion(t.Context(), archive, "1.25.0", versions); err == nil {
				t.Error("expected the escaping entry to be rejected")
			}
			found := false
			_ = filepath.WalkDir(filepath.Dir(base), func(path string, _ os.DirEntry, _ error) error {
				if filepath.Base(path) == "escaped" {
					found = true
				}
				return nil
			})
			if found {
				t.Error("entry was written outside of the version folder")
			}
		})
	}
}
