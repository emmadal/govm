package pkg

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type entry struct {
	name, body, link string
	mode             int64
	dir              bool
}

func writeTarGz(t *testing.T, path string, entries []entry) {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		hdr := &tar.Header{Name: e.name, Mode: e.mode, Size: int64(len(e.body)), Typeflag: tar.TypeReg}
		switch {
		case e.dir:
			hdr.Typeflag, hdr.Size = tar.TypeDir, 0
		case e.link != "":
			hdr.Typeflag, hdr.Linkname, hdr.Size = tar.TypeSymlink, e.link, 0
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if hdr.Typeflag == tar.TypeReg {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeZip(t *testing.T, path string, entries []entry) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		w, err := zw.Create(e.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(e.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func goDistribution() []entry {
	return []entry{
		{name: "go/", dir: true, mode: 0o755},
		{name: "go/VERSION", body: "go1.22.0", mode: 0o644},
		{name: "go/bin/go", body: "#!/bin/sh\n", mode: 0o755},
		{name: "go/src/fmt/print.go", body: "package fmt", mode: 0o644},
	}
}

func TestExtractTarGz(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "go.tar.gz")
	writeTarGz(t, archive, goDistribution())

	dest := filepath.Join(dir, "go1.22.0")
	if err := ExtractArchive(archive, dest); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dest, "src", "fmt", "print.go"))
	if err != nil || string(data) != "package fmt" {
		t.Fatalf("print.go = %q, %v", data, err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(dest, "bin", "go"))
		if err != nil || info.Mode().Perm()&0o100 == 0 {
			t.Fatalf("bin/go not executable: %v %v", info, err)
		}
	}
	assertNoTempDirs(t, dir)
}

func TestExtractZip(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "go.zip")
	writeZip(t, archive, goDistribution()[1:])

	dest := filepath.Join(dir, "go1.22.0")
	if err := ExtractArchive(archive, dest); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(dest, "VERSION")); err != nil || string(data) != "go1.22.0" {
		t.Fatalf("VERSION = %q, %v", data, err)
	}
}

func TestExtractRejectsTraversal(t *testing.T) {
	cases := map[string][]entry{
		"dotdot":      {{name: "go/../../evil", body: "x", mode: 0o644}},
		"symlink-out": {{name: "go/link", link: "../../outside", mode: 0o777}},
		"symlink-abs": {{name: "go/link", link: "/etc/passwd", mode: 0o777}},
	}
	for name, entries := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			archive := filepath.Join(dir, "evil.tar.gz")
			writeTarGz(t, archive, entries)
			dest := filepath.Join(dir, "out")
			err := ExtractArchive(archive, dest)
			if err == nil || !strings.Contains(err.Error(), "illegal") {
				t.Fatalf("got %v, want illegal path error", err)
			}
			if _, err := os.Stat(dest); !os.IsNotExist(err) {
				t.Fatal("destination exists after failed extraction")
			}
			if _, err := os.Stat(filepath.Join(dir, "..", "evil")); err == nil {
				t.Fatal("file written outside destination")
			}
			assertNoTempDirs(t, dir)
		})
	}
}

func TestExtractZipRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "evil.zip")
	writeZip(t, archive, []entry{{name: "go/../../evil", body: "x"}})
	if err := ExtractArchive(archive, filepath.Join(dir, "out")); err == nil {
		t.Fatal("expected error")
	}
}

func assertNoTempDirs(t *testing.T, dir string) {
	t.Helper()
	matches, _ := filepath.Glob(filepath.Join(dir, ".extract-*"))
	if len(matches) > 0 {
		t.Fatalf("temporary directories left behind: %v", matches)
	}
}
