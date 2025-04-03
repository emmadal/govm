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
	// Check if shell config is valid
	if shellConfig == "" {
		return fmt.Errorf("invalid shell configuration file name")
	}

	// Portable `sed` command for macOS and Linux
	cmdStr := fmt.Sprintf("sed -i '' '/export PATH=.*.govm\\/version/d' ~/%s", shellConfig)
	cmd := exec.Command("sh", "-c", cmdStr)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove old Go paths from %s", shellConfig)
	}
	return nil
}

// GetActiveGoVersion returns the active Go version.
func GetActiveGoVersion() (string, error) {
	// Check if go is installed
	if _, err := exec.LookPath("go"); err != nil {
		return "", fmt.Errorf("go is not installed. Please install go first with 'govm install'")
	}

	// Get active Go version
	cmd := exec.Command("go", "version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("could not determine active Go version")
	}
	// Extract version using regex
	re := regexp.MustCompile(`go\d+(\.\d+)+`)
	match := re.FindString(string(output))
	if match == "" {
		return "", fmt.Errorf("could not determine active Go version")
	}
	return match, nil
}

// UseGoVersion changes the active Go version.
func UseGoVersion(version string) error {
	goVersionDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	goPath := fmt.Sprintf("%s/%s%s/bin", goVersionDir, GO_PATH, version)
	shellConfig, err := GetShellConfig()
	if err != nil {
		return err
	}

	// Set PATH for the current session
	newPath := fmt.Sprintf("%s%c%s", goPath, os.PathListSeparator, os.Getenv("PATH"))
	if err := os.Setenv("PATH", newPath); err != nil {
		return err
	}

	// Persist PATH update in shell profile
	err = UpdateShellProfile(goPath)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "✅ Switched to go%s. Run 'source ~/%s' and restart your terminal to apply permanently.\n", version, shellConfig)

	return nil
}
