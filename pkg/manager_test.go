package pkg

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeGoDev serves a release index and archives for the given versions,
// mimicking go.dev/dl.
func fakeGoDev(t *testing.T, versions ...string) *httptest.Server {
	t.Helper()
	archives := map[string][]byte{}
	var releases []Release
	for _, v := range versions {
		ext := "tar.gz"
		if runtime.GOOS == "windows" {
			ext = "zip"
		}
		name := "go" + v + "." + runtime.GOOS + "-" + runtime.GOARCH + "." + ext
		path := filepath.Join(t.TempDir(), name)
		entries := []entry{{name: "go/VERSION", body: "go" + v, mode: 0o644}, {name: "go/bin/go", body: "#!/bin/sh\n", mode: 0o755}}
		if ext == "zip" {
			writeZip(t, path, entries)
		} else {
			writeTarGz(t, path, entries)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		archives[name] = data
		sum := sha256.Sum256(data)
		releases = append(releases, Release{Version: "go" + v, Stable: !strings.Contains(v, "rc"), Files: []ReleaseFile{
			{Filename: name, OS: runtime.GOOS, Arch: runtime.GOARCH, Kind: "archive", SHA256: hex.EncodeToString(sum[:])},
		}})
	}
	index, _ := json.Marshal(releases)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/index" {
			_, _ = w.Write(index)
			return
		}
		data, ok := archives[strings.TrimPrefix(r.URL.Path, "/dl/")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)

	oldIndex, oldBase := ReleaseIndexURL, DownloadBaseURL
	ReleaseIndexURL, DownloadBaseURL = srv.URL+"/index", srv.URL+"/dl/"
	t.Cleanup(func() { ReleaseIndexURL, DownloadBaseURL = oldIndex, oldBase })
	return srv
}

// sandbox points HOME and GOVM_DIR at temporary directories and captures output.
func sandbox(t *testing.T) (Paths, *bytes.Buffer) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("would modify the real user PATH on Windows")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/bash")
	t.Setenv("GOVM_DIR", filepath.Join(home, ".govm"))
	t.Setenv("NO_COLOR", "1")

	var out bytes.Buffer
	oldOut, oldErr, oldIn := Stdout, Stderr, Stdin
	Stdout, Stderr = &out, io.Discard
	t.Cleanup(func() { Stdout, Stderr, Stdin = oldOut, oldErr, oldIn })

	p, err := ResolvePaths()
	if err != nil {
		t.Fatal(err)
	}
	return p, &out
}

func TestInstallUseListRemove(t *testing.T) {
	fakeGoDev(t, "1.23rc1", "1.22.3", "1.21.5")
	p, out := sandbox(t)

	if err := Install("1.21"); err != nil {
		t.Fatal(err)
	}
	if err := Install("latest"); err != nil {
		t.Fatal(err)
	}
	if active, _ := ActiveVersion(p); active != "1.22.3" {
		t.Fatalf("active = %q after installing latest", active)
	}
	data, err := os.ReadFile(filepath.Join(p.CurrentBin(), "..", "VERSION"))
	if err != nil || string(data) != "go1.22.3" {
		t.Fatalf("current/VERSION = %q, %v", data, err)
	}

	// Reinstalling an installed version only switches to it.
	if err := Install("1.21.5"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "go1.21.5 is already installed") {
		t.Errorf("output missing 'already installed':\n%s", out)
	}

	if err := Use("go1.22"); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := List(); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "* go1.22.3 (active)\n  go1.21.5\n" {
		t.Errorf("List() =\n%s", got)
	}

	if err := Remove("1.22.3", true); err == nil {
		t.Error("removing the active version succeeded")
	}
	Stdin = strings.NewReader("n\n")
	if err := Remove("1.21.5", false); err != nil || !p.IsInstalled("1.21.5") {
		t.Fatalf("declined removal: err=%v installed=%v", err, p.IsInstalled("1.21.5"))
	}
	Stdin = strings.NewReader("y\n")
	if err := Remove("1.21.5", false); err != nil || p.IsInstalled("1.21.5") {
		t.Fatalf("confirmed removal: err=%v installed=%v", err, p.IsInstalled("1.21.5"))
	}
	if cached, _ := filepath.Glob(filepath.Join(p.Cache, "go1.21.5.*")); len(cached) != 0 {
		t.Errorf("cached archive not removed: %v", cached)
	}

	rc, _ := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".bashrc"))
	if strings.Count(string(rc), blockStart) != 1 {
		t.Errorf("expected exactly one govm block in .bashrc:\n%s", rc)
	}
}

func TestInstallRejectsCorruptCache(t *testing.T) {
	fakeGoDev(t, "1.22.3")
	p, _ := sandbox(t)
	if err := p.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	// A tampered archive in the cache must be replaced, not installed.
	name := "go1.22.3." + runtime.GOOS + "-" + runtime.GOARCH + ".tar.gz"
	writeTarGz(t, filepath.Join(p.Cache, name), []entry{{name: "go/VERSION", body: "evil", mode: 0o644}})

	if err := Install("1.22.3"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(p.VersionDir("1.22.3"), "VERSION"))
	if string(data) != "go1.22.3" {
		t.Fatalf("installed tampered archive: VERSION = %q", data)
	}
}

func TestInstallUnknownVersion(t *testing.T) {
	fakeGoDev(t, "1.22.3")
	p, _ := sandbox(t)
	if err := Install("1.19"); err == nil {
		t.Fatal("expected error")
	}
	if versions, _ := InstalledVersions(p); len(versions) != 0 {
		t.Fatalf("installed %v", versions)
	}
}
