package pkg

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// extract unpacks the Go archive at file (.zip or .tar.gz) into dest, dropping
// the top-level directory of the archive. It needs no external tool, and every
// entry is confined to dest: an archive cannot write outside of it.
func extract(ctx context.Context, file, dest string) error {
	root, err := os.OpenRoot(dest)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()

	var entries int
	if strings.HasSuffix(file, ".zip") {
		entries, err = extractZip(ctx, file, root)
	} else {
		entries, err = extractTarGz(ctx, file, root)
	}
	if err != nil {
		return err
	}
	if entries == 0 {
		return errors.New("archive contains no files")
	}
	return nil
}

// stripTopDir removes the first component of an archive entry name, such as
// the "go/" every Go archive starts with. It reports false for the top-level
// directory itself.
func stripTopDir(name string) (string, bool) {
	_, rest, ok := strings.Cut(path.Clean(filepath.ToSlash(name)), "/")
	if !ok || rest == "" {
		return "", false
	}
	return filepath.FromSlash(rest), true
}

func extractTarGz(ctx context.Context, file string, root *os.Root) (int, error) {
	f, err := os.Open(file)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return 0, err
	}
	defer func() { _ = gz.Close() }()

	entries := 0
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return entries, nil
		}
		if err != nil {
			return entries, err
		}
		if err := ctx.Err(); err != nil {
			return entries, err
		}
		name, ok := stripTopDir(header.Name)
		if !ok {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			err = root.MkdirAll(name, 0o755)
		case tar.TypeReg:
			err = writeFile(root, name, header.FileInfo().Mode().Perm(), tr)
		case tar.TypeSymlink:
			if err = makeParent(root, name); err == nil {
				err = root.Symlink(header.Linkname, name)
			}
		default:
			continue // extended headers and special files
		}
		if err != nil {
			return entries, err
		}
		entries++
	}
}

func extractZip(ctx context.Context, file string, root *os.Root) (int, error) {
	zr, err := zip.OpenReader(file)
	if err != nil {
		return 0, err
	}
	defer func() { _ = zr.Close() }()

	entries := 0
	for _, entry := range zr.File {
		if err := ctx.Err(); err != nil {
			return entries, err
		}
		name, ok := stripTopDir(entry.Name)
		if !ok {
			continue
		}

		if entry.FileInfo().IsDir() {
			err = root.MkdirAll(name, 0o755)
		} else {
			err = writeZipEntry(root, name, entry)
		}
		if err != nil {
			return entries, err
		}
		entries++
	}
	return entries, nil
}

func writeZipEntry(root *os.Root, name string, entry *zip.File) error {
	src, err := entry.Open()
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()
	return writeFile(root, name, entry.Mode().Perm(), src)
}

// makeParent creates the directory that will hold name.
func makeParent(root *os.Root, name string) error {
	if dir := filepath.Dir(name); dir != "." {
		return root.MkdirAll(dir, 0o755)
	}
	return nil
}

// writeFile creates name inside root with the content of src.
func writeFile(root *os.Root, name string, perm os.FileMode, src io.Reader) (err error) {
	if err := makeParent(root, name); err != nil {
		return err
	}
	if perm == 0 {
		perm = 0o644
	}
	dst, err := root.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := dst.Close(); err == nil {
			err = closeErr
		}
	}()
	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to write %s: %w", name, err)
	}
	return nil
}
