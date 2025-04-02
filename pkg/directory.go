package pkg

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetHomeDir returns the home directory.
func GetHomeDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to get user home directory")
	}
	if homeDir == "" {
		return "", fmt.Errorf("home directory cannot be empty")
	}
	return homeDir, nil
}

// CreateConfigDir creates the config directory.
func CreateConfigDir() error {
	homeDir, err := GetHomeDir()
	if err != nil {
		return err
	}
	configPath := filepath.Join(homeDir, config, goVersions, GO_PATH)

	if err := os.MkdirAll(configPath, 0755); err != nil {
		return fmt.Errorf("unable to create config directory")
	}
	return nil
}

// CreateCacheDir creates the cache directory.
func CreateCacheDir() (string, error) {
	homeDir, err := GetHomeDir()
	if err != nil {
		return "", err
	}
	cachePath := filepath.Join(homeDir, config, goCache)

	if err := os.MkdirAll(cachePath, 0755); err != nil {
		return "", fmt.Errorf("unable to create cache directory")
	}
	return cachePath, nil
}

// GetConfigDir returns the config directory.
func GetConfigDir() (string, error) {
	homeDir, err := GetHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, config, goVersions, GO_PATH), nil
}

// GetCacheDir returns the cache directory.
func GetCacheDir() (string, error) {
	homeDir, err := GetHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, config, goCache), nil
}
