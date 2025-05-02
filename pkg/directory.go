package pkg

import (
	"fmt"
	"os"
	"path/filepath"
)

type Directory struct {
	ConfigDir string
	CacheDir  string
}

// GetDirectories returns the config and cache directories.
func (d *Directory) GetDirectories() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to get home directory")
	}
	d.ConfigDir = filepath.Join(homeDir, ".govm", "versions", "go")
	d.CacheDir = filepath.Join(homeDir, ".govm", ".cache")
	return nil
}

// CreateInstallDir creates the config directory.
func (d *Directory) CreateInstallDir() error {
	if err := os.MkdirAll(d.ConfigDir, 0755); err != nil {
		return fmt.Errorf("unable to create config directory")
	}
	if err := os.MkdirAll(d.CacheDir, 0755); err != nil {
		return fmt.Errorf("unable to create cache directory")
	}
	return nil
}
