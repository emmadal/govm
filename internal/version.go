package internal

import (
	"os"
	"path/filepath"
	"strings"
)

// GetVersion returns the version of govm
func GetVersion() string {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return "dev" // Fallback if VERSION file is missing
	}
	versionFile := filepath.Join(homedir, ".local", "bin", "VERSION")
	data, err := os.ReadFile(versionFile)
	if err != nil {
		return "dev" // Fallback if a VERSION file is missing
	}
	return strings.TrimSpace(string(data))
}
