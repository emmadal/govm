package internal

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/emmadal/govm/pkg"
)

func fakeGitHub(t *testing.T, handler http.HandlerFunc) *http.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	oldLatest, oldDownload := latestReleaseURL, releaseDownloadURL
	latestReleaseURL, releaseDownloadURL = srv.URL+"/latest", srv.URL+"/download"
	t.Cleanup(func() { latestReleaseURL, releaseDownloadURL = oldLatest, oldDownload })
	return srv.Client()
}

func TestLatestTag(t *testing.T) {
	status := http.StatusOK
	client := fakeGitHub(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"tag_name":"v1.2.3"}`))
	})

	if tag, err := latestTag(client); err != nil || tag != "v1.2.3" {
		t.Fatalf("latestTag = %q, %v", tag, err)
	}
	status = http.StatusForbidden // e.g. API rate limit
	if _, err := latestTag(client); err == nil {
		t.Fatal("expected error on HTTP 403")
	}
}

func TestExpectedChecksum(t *testing.T) {
	asset := assetName()
	client := fakeGitHub(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/download/v1.0.0/checksums.txt":
			_, _ = w.Write([]byte("aaaa  govm_plan9_arm\nbbbb *build/" + asset + "\n"))
		case "/download/v2.0.0/checksums.txt":
			_, _ = w.Write([]byte("aaaa  govm_plan9_arm\n"))
		default:
			http.NotFound(w, r)
		}
	})

	if sum, err := expectedChecksum(client, "v1.0.0", asset); err != nil || sum != "bbbb" {
		t.Errorf("v1.0.0: %q, %v", sum, err)
	}
	if _, err := expectedChecksum(client, "v2.0.0", asset); err == nil {
		t.Error("v2.0.0: expected error for missing entry")
	}
	if sum, err := expectedChecksum(client, "v0.9.0", asset); err != nil || sum != "" {
		t.Errorf("v0.9.0 (no checksums.txt): %q, %v", sum, err)
	}
}

func TestIsGovmRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	check := func(root string) bool {
		t.Setenv("GOVM_DIR", root)
		p, err := pkg.ResolvePaths()
		if err != nil {
			t.Fatal(err)
		}
		return isGovmRoot(p)
	}

	if check(home) {
		t.Error("home directory accepted as govm root")
	}
	if !check(filepath.Join(home, ".govm")) {
		t.Error(".govm rejected")
	}
	custom := filepath.Join(home, "tools")
	if check(custom) {
		t.Error("unrelated directory accepted")
	}
	if err := os.MkdirAll(filepath.Join(custom, "versions", "go"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !check(custom) {
		t.Error("custom root with versions/go rejected")
	}
}
