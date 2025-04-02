package pkg

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

// UpdateShellProfile updates the shell profile.
func UpdateShellProfile(goPath string) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("Windows is not supported for updating shell profile")
	}
	// Get shell config
	shellConfig, err := GetShellConfig()
	if err != nil {
		return err
	}

	// Check if shell config is valid
	if shellConfig == "" {
		return fmt.Errorf("invalid shell configuration file name")
	}

	// Remove old Go path entries
	if err := RemoveOldGoPaths(shellConfig); err != nil {
		return err
	}

	// Append the new Go path
	cmd := exec.Command("sh", "-c", fmt.Sprintf("echo 'export PATH=\"%s:$PATH\"' >> ~/%s", goPath, shellConfig))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to append new Go path to %s: %s", shellConfig, err.Error())
	}
	return nil
}

// GetShellConfig returns the shell config file.
func GetShellConfig() (string, error) {
	if runtime.GOOS == "windows" {
		return "", fmt.Errorf("Windows is not supported for getting shell config")
	}

	// Get shell config
	shell := os.Getenv("SHELL")
	if shell == "" {
		return "", fmt.Errorf("could not determine shell, SHELL environment variable is empty")
	}

	if strings.Contains(shell, "zsh") {
		return ".zshrc", nil
	} else if strings.Contains(shell, "bash") {
		// On macOS, prefer `.bash_profile`
		if runtime.GOOS == "darwin" {
			return ".bash_profile", nil
		}
		return ".bashrc", nil
	} else if strings.Contains(shell, "fish") {
		return ".config/fish/config.fish", nil
	} else if strings.Contains(shell, "dash") {
		return ".profile", nil
	}

	return "", fmt.Errorf("unsupported shell: %s", shell)
}

// RemoveOldGoPaths removes old Go paths from the shell profile.
func RemoveOldGoPaths(shellConfig string) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("Windows is not supported for removing old Go paths")
	}

	// Check if shell config is valid
	if shellConfig == "" {
		return fmt.Errorf("invalid shell configuration file name")
	}

	// Portable `sed` command for macOS and Linux
	cmdStr := fmt.Sprintf(`[ "$(uname)" = "Darwin" ] && sed -i '' '/export PATH=.*.govm\\/version/d' ~/%s || sed -i '/export PATH=.*.govm\\/version/d' ~/%s`, shellConfig, shellConfig)
	cmd := exec.Command("sh", "-c", cmdStr)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove old Go paths from %s: %s", shellConfig, err.Error())
	}
	return nil
}

// GetActiveGoVersion returns the active Go version.
func GetActiveGoVersion() (string, error) {
	if runtime.GOOS == "windows" {
		return "", fmt.Errorf("Windows is not supported for getting active Go version")
	}

	// Check if go is installed
	if _, err := exec.LookPath("go"); err != nil {
		return "", fmt.Errorf("%s%s%s", Red_ANSI, "go is not installed. Please install go first with 'govm install'", Reset_ANSI)
	}

	// Get active Go version
	cmd := exec.Command("go", "version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s%s%s", Red_ANSI, "could not determine active Go version", Reset_ANSI)
	}
	// Extract version using regex
	re := regexp.MustCompile(`go\d+(\.\d+)+`)
	match := re.FindString(string(output))
	if match == "" {
		return "", fmt.Errorf("%s%s%s", Red_ANSI, "could not determine active Go version", Reset_ANSI)
	}
	return match, nil
}
