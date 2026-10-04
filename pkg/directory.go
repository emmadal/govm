package pkg

import (
	"fmt"
	"os"
	"path/filepath"
)

// Paths holds every location govm reads from or writes to.
// It is the single source of truth for directory layout.
type Paths struct {
	Root     string // ~/.govm, or $GOVM_DIR when set
	Versions string // Root/versions/go, one go<version> directory per install
	Cache    string // Root/.cache, downloaded archives
	Current  string // Root/current, link to the active version directory
}

// ResolvePaths returns govm's directory layout, honouring $GOVM_DIR.
func ResolvePaths() (Paths, error) {
	root := os.Getenv("GOVM_DIR")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Paths{}, fmt.Errorf("unable to determine home directory: %w", err)
		}
		root = filepath.Join(home, ".govm")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return Paths{}, err
	}
	return Paths{
		Root:     root,
		Versions: filepath.Join(root, "versions", "go"),
		Cache:    filepath.Join(root, ".cache"),
		Current:  filepath.Join(root, "current"),
	}, nil
}

// EnsureDirs creates the versions and cache directories.
func (p Paths) EnsureDirs() error {
	for _, dir := range []string{p.Versions, p.Cache} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("unable to create %s: %w", dir, err)
		}
	}
	return nil
}

// VersionDir returns the install directory of a version (e.g. "1.22.0").
func (p Paths) VersionDir(version string) string {
	return filepath.Join(p.Versions, "go"+version)
}

// CurrentBin returns the bin directory of the active version, which is
// the one entry govm adds to PATH.
func (p Paths) CurrentBin() string {
	return filepath.Join(p.Current, "bin")
}

// IsInstalled reports whether a version directory exists.
func (p Paths) IsInstalled(version string) bool {
	info, err := os.Stat(p.VersionDir(version))
	return err == nil && info.IsDir()
}
