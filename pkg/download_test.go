package pkg

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var payload = bytes.Repeat([]byte("govm"), 64*1024) // 256 KiB

// fastDownloader returns a Downloader whose waits are short enough for tests.
func fastDownloader() *Downloader {
	return &Downloader{IdleTimeout: 300 * time.Millisecond, RetryDelay: time.Millisecond}
}

// stall blocks the handler until the client gives up.
func stall(r *http.Request) { <-r.Context().Done() }

func assertEmptyDir(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		t.Errorf("leftover file in %s: %s", dir, entry.Name())
	}
}

func assertFile(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("%s: got %d bytes, want %d", path, len(got), len(want))
	}
}

func TestDownload_IdleDefaults(t *testing.T) {
	if DefaultIdleTimeout != 60*time.Second {
		t.Errorf("DefaultIdleTimeout = %s, want 60s", DefaultIdleTimeout)
	}
	if DefaultMaxAttempts != 3 {
		t.Errorf("DefaultMaxAttempts = %d, want 3", DefaultMaxAttempts)
	}
}

// A download that keeps progressing must survive for longer than the idle
// timeout: the timeout bounds silence, not total duration.
func TestDownload_IdleSlowButActiveSucceeds(t *testing.T) {
	var requests atomic.Int32
	const chunks = 30
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		for range chunks {
			_, _ = w.Write([]byte("x"))
			w.(http.Flusher).Flush()
			time.Sleep(30 * time.Millisecond)
		}
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "file")
	start := time.Now()
	if err := fastDownloader().Download(t.Context(), srv.URL, dest); err != nil {
		t.Fatalf("Download: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 600*time.Millisecond {
		t.Fatalf("download took %s, expected it to outlast the 300ms idle timeout", elapsed)
	}
	if got := requests.Load(); got != 1 {
		t.Errorf("requests = %d, want 1", got)
	}
	assertFile(t, dest, bytes.Repeat([]byte("x"), chunks))
}

func TestDownload_IdleSilentBodyTimesOut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("partial"))
		w.(http.Flusher).Flush()
		stall(r)
	}))
	defer srv.Close()

	dir := t.TempDir()
	d := fastDownloader()
	d.MaxAttempts = 1
	err := d.Download(t.Context(), srv.URL, filepath.Join(dir, "file"))
	if !errors.Is(err, ErrIdleTimeout) {
		t.Fatalf("err = %v, want ErrIdleTimeout", err)
	}
	assertEmptyDir(t, dir)
}

func TestDownload_IdleSilentHeadersTimesOut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { stall(r) }))
	defer srv.Close()

	dir := t.TempDir()
	d := fastDownloader()
	d.MaxAttempts = 1
	err := d.Download(t.Context(), srv.URL, filepath.Join(dir, "file"))
	if !errors.Is(err, ErrIdleTimeout) {
		t.Fatalf("err = %v, want ErrIdleTimeout", err)
	}
	assertEmptyDir(t, dir)
}

// The final name must not exist while the body is still arriving.
func TestDownload_AtomicNoFinalFileUntilComplete(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "file")
	sawFinal := make(chan bool, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload[:1024])
		w.(http.Flusher).Flush()
		time.Sleep(50 * time.Millisecond)
		_, err := os.Stat(dest)
		sawFinal <- err == nil
		_, _ = w.Write(payload[1024:])
	}))
	defer srv.Close()

	if err := fastDownloader().Download(t.Context(), srv.URL, dest); err != nil {
		t.Fatalf("Download: %v", err)
	}
	if <-sawFinal {
		t.Error("final file existed before the download completed")
	}
	assertFile(t, dest, payload)
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("want only the final file, got %d entries", len(entries))
	}
}

func TestDownload_NoContentLength(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.(http.Flusher).Flush() // forces chunked encoding: no Content-Length
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	var total int64
	d := fastDownloader()
	d.Progress = func(_ int, n int64) io.Writer { total = n; return nil }
	dest := filepath.Join(t.TempDir(), "file")
	if err := d.Download(t.Context(), srv.URL, dest); err != nil {
		t.Fatalf("Download: %v", err)
	}
	if total != -1 {
		t.Errorf("reported total = %d, want -1 (unknown)", total)
	}
	assertFile(t, dest, payload)
}

// A body shorter than Content-Length is a failed attempt, never a valid file.
func TestDownload_TruncatedBodyFails(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Length", "1000")
		_, _ = w.Write([]byte("short"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	err := fastDownloader().Download(t.Context(), srv.URL, filepath.Join(dir, "file"))
	if err == nil {
		t.Fatal("expected an error for a truncated body")
	}
	if got := requests.Load(); got != 3 {
		t.Errorf("requests = %d, want 3 (truncation is retried)", got)
	}
	assertEmptyDir(t, dir)
}

func TestDownload_ReplacesLargerExistingFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("new"))
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(dest, bytes.Repeat([]byte("old"), 1000), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := fastDownloader().Download(t.Context(), srv.URL, dest); err != nil {
		t.Fatalf("Download: %v", err)
	}
	assertFile(t, dest, []byte("new"))
}

// A failed download must leave a previously valid file untouched.
func TestDownload_FailureKeepsExistingFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(dest, []byte("valid"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := fastDownloader().Download(t.Context(), srv.URL, dest); err == nil {
		t.Fatal("expected an error")
	}
	assertFile(t, dest, []byte("valid"))
}

func TestDownload_RetrySucceedsOnThirdAttempt(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) < 3 {
			stall(r)
			return
		}
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	var retries []int
	d := fastDownloader()
	d.OnRetry = func(attempt, maxAttempts int, _ error) {
		if maxAttempts != 3 {
			t.Errorf("maxAttempts = %d, want 3", maxAttempts)
		}
		retries = append(retries, attempt)
	}
	dest := filepath.Join(t.TempDir(), "file")
	if err := d.Download(t.Context(), srv.URL, dest); err != nil {
		t.Fatalf("Download: %v", err)
	}
	if got := requests.Load(); got != 3 {
		t.Errorf("requests = %d, want 3", got)
	}
	if len(retries) != 2 || retries[0] != 2 || retries[1] != 3 {
		t.Errorf("OnRetry attempts = %v, want [2 3]", retries)
	}
	assertFile(t, dest, payload)
}

func TestDownload_RetryGivesUpAfterThreeTimeouts(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte("partial"))
		w.(http.Flusher).Flush()
		stall(r)
	}))
	defer srv.Close()

	dir := t.TempDir()
	err := fastDownloader().Download(t.Context(), srv.URL, filepath.Join(dir, "file"))
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "after 3 attempts") {
		t.Errorf("err = %q, want it to mention \"after 3 attempts\"", err)
	}
	if !errors.Is(err, ErrIdleTimeout) {
		t.Errorf("err = %v, want it to wrap ErrIdleTimeout", err)
	}
	if got := requests.Load(); got != 3 {
		t.Errorf("requests = %d, want 3", got)
	}
	assertEmptyDir(t, dir)
}

func TestDownload_RetryByStatus(t *testing.T) {
	tests := []struct {
		status       int
		wantRequests int32
	}{
		{http.StatusNotFound, 1},
		{http.StatusForbidden, 1},
		{http.StatusTooManyRequests, 3},
		{http.StatusInternalServerError, 3},
		{http.StatusBadGateway, 3},
	}
	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			var requests atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()

			dir := t.TempDir()
			err := fastDownloader().Download(t.Context(), srv.URL, filepath.Join(dir, "file"))
			statusErr, ok := errors.AsType[*HTTPStatusError](err)
			if !ok || statusErr.StatusCode != tt.status {
				t.Fatalf("err = %v, want HTTPStatusError %d", err, tt.status)
			}
			if got := requests.Load(); got != tt.wantRequests {
				t.Errorf("requests = %d, want %d", got, tt.wantRequests)
			}
			assertEmptyDir(t, dir)
		})
	}
}

// A local filesystem failure cannot be fixed by retrying.
func TestDownload_RetrySkippedOnLocalError(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "missing-dir", "file")
	if err := fastDownloader().Download(t.Context(), srv.URL, dest); err == nil {
		t.Fatal("expected an error")
	}
	if got := requests.Load(); got != 1 {
		t.Errorf("requests = %d, want 1", got)
	}
}

func TestDownload_CancelStopsWithoutRetry(t *testing.T) {
	var requests atomic.Int32
	started := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte("partial"))
		w.(http.Flusher).Flush()
		started <- struct{}{}
		stall(r)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		<-started
		cancel()
	}()

	dir := t.TempDir()
	d := fastDownloader()
	d.IdleTimeout = time.Minute // only the cancellation can end this attempt
	err := d.Download(ctx, srv.URL, filepath.Join(dir, "file"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if got := requests.Load(); got != 1 {
		t.Errorf("requests = %d, want 1 (no retry after cancellation)", got)
	}
	assertEmptyDir(t, dir)
}
