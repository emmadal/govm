package pkg

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractArchive unpacks a Go distribution (.tar.gz or .zip) into dest,
// stripping the leading "go/" directory. It extracts into a sibling
// temporary directory first so a failed extraction never leaves a
// half-installed version behind. dest must not exist.
func ExtractArchive(archive, dest string) error {
	tmp, err := os.MkdirTemp(filepath.Dir(dest), ".extract-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	switch {
	case strings.HasSuffix(archive, ".tar.gz"):
		err = extractTarGz(archive, tmp)
	case strings.HasSuffix(archive, ".zip"):
		err = extractZip(archive, tmp)
	default:
		err = fmt.Errorf("unsupported archive format: %s", filepath.Base(archive))
	}
	if err != nil {
		return fmt.Errorf("failed to extract %s: %w", filepath.Base(archive), err)
	}
	return os.Rename(tmp, dest)
}

// targetPath maps an archive entry to a path under dest, dropping the first
// path component. It returns "" for the top-level directory itself and an
// error for entries that would escape dest.
func targetPath(dest, name string) (string, error) {
	name = strings.TrimPrefix(filepath.ToSlash(name), "./")
	_, rest, found := strings.Cut(name, "/")
	if !found || rest == "" {
		return "", nil
	}
	if !filepath.IsLocal(filepath.FromSlash(rest)) {
		return "", fmt.Errorf("illegal path in archive: %s", name)
	}
	return filepath.Join(dest, filepath.FromSlash(rest)), nil
}

func writeFile(path string, r io.Reader, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	perm := mode.Perm()
	if perm == 0 { // zip entries created on Windows carry no Unix mode
		perm = 0o644
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm|0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func extractTarGz(archive, dest string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		path, err := targetPath(dest, hdr.Name)
		if err != nil {
			return err
		}
		if path == "" {
			continue
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := writeFile(path, tr, hdr.FileInfo().Mode()); err != nil {
				return err
			}
		case tar.TypeSymlink:
			// Only allow links that stay inside the extracted tree.
			resolved := filepath.Join(filepath.Dir(path), hdr.Linkname)
			if filepath.IsAbs(hdr.Linkname) || !strings.HasPrefix(resolved, dest+string(filepath.Separator)) {
				return fmt.Errorf("illegal symlink in archive: %s -> %s", hdr.Name, hdr.Linkname)
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			if err := os.Symlink(hdr.Linkname, path); err != nil {
				return err
			}
		}
	}
}

func extractZip(archive, dest string) error {
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer func() { _ = zr.Close() }()

	for _, zf := range zr.File {
		path, err := targetPath(dest, zf.Name)
		if err != nil {
			return err
		}
		if path == "" {
			continue
		}
		if zf.FileInfo().IsDir() {
			if err := os.MkdirAll(path, 0o755); err != nil {
				return err
			}
			continue
		}
		rc, err := zf.Open()
		if err != nil {
			return err
		}
		err = writeFile(path, rc, zf.Mode())
		_ = rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
