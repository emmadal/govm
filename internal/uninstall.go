package internal

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

// ANSI color codes
const (
	Green = "\033[32m"
	Blue  = "\033[34m"
	Red   = "\033[31m"
	Reset = "\033[0m"
	Bold  = "\033[1m"
)

func colorPrintln(color, text string) {
	_, _ = fmt.Fprint(os.Stdout, color+text+Reset+"\n")
}

// checkSudo checks if the user has sudo access
func checkSudo() bool {
	cmd := exec.Command("sudo", "-n", "true")
	return cmd.Run() == nil
}

// detectShellProfile finds the appropriate shell profile file
func detectShellProfile() string {
	shell := os.Getenv("SHELL")
	shellName := filepath.Base(shell)

	homedir, err := os.UserHomeDir()
	if err != nil {
		colorPrintln(Red, "Error getting home directory: "+err.Error())
		os.Exit(1)
	}

	switch shellName {
	case "bash":
		bashProfile := filepath.Join(homedir, ".bash_profile")
		if _, err := os.Stat(bashProfile); err == nil {
			return bashProfile
		}
		return filepath.Join(homedir, ".bashrc")
	case "zsh":
		return filepath.Join(homedir, ".zshrc")
	default:
		return filepath.Join(homedir, ".profile")
	}
}

// removeFile removes a file with optional sudo
func removeFile(path string, useSudo bool) error {
	if useSudo {
		cmd := exec.Command("sudo", "rm", "-f", path)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	return os.Remove(path)
}

// cleanShellProfile removes govm-related lines from shell profile
func cleanShellProfile(profilePath string) error {
	// Read the profile file
	file, err := os.Open(profilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a temporary file
	tempFile, err := os.CreateTemp(filepath.Dir(profilePath), "tempprofile")
	if err != nil {
		return err
	}
	tempFilePath := tempFile.Name()
	defer os.Remove(tempFilePath) // Clean up in case of failure

	// Filter out govm-related lines
	scanner := bufio.NewScanner(file)
	writer := bufio.NewWriter(tempFile)

	govmPatterns := []*regexp.Regexp{
		regexp.MustCompile(`# govm installation`),
		regexp.MustCompile(`export PATH="\${HOME}/\.local/bin:\${PATH}"`),
		regexp.MustCompile(`export PATH="\${GOVM_DIR}/versions/go`),
		regexp.MustCompile(`export GOROOT=`),
	}

	for scanner.Scan() {
		line := scanner.Text()
		keepLine := true

		for _, pattern := range govmPatterns {
			if pattern.MatchString(line) {
				keepLine = false
				break
			}
		}

		if keepLine {
			if _, err := writer.WriteString(line + "\n"); err != nil {
				_ = tempFile.Close()
				return err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		_ = tempFile.Close()
		return err
	}

	_ = writer.Flush()
	_ = tempFile.Close()

	// Replace the original file with the cleaned version
	return os.Rename(tempFilePath, profilePath)
}

func Uninstall() error {
	// Print header
	colorPrintln(Red+Bold, "Removing govm - Go Version Manager")

	// Define installation directories
	homedir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error getting home directory: %v", err)
	}

	govmDir := filepath.Join(homedir, ".govm")
	govmBinDir := "/usr/local/bin"

	// Check if the user has sudo access
	hasSudo := checkSudo()
	if !hasSudo {
		colorPrintln(Blue, "No sudo access detected. Assuming govm was installed locally.")
		govmBinDir = filepath.Join(homedir, ".local/bin")
	}

	// Detect shell profile
	shellProfile := detectShellProfile()

	// Remove govm binary
	colorPrintln(Blue, "Removing govm binary...")
	govmBinaryPath := filepath.Join(govmBinDir, "govm")
	if _, err := os.Stat(govmBinaryPath); err == nil {
		if hasSudo {
			if err := removeFile(govmBinaryPath, true); err != nil {
				return fmt.Errorf("failed to remove govm binary: %v", err)
			} else {
				colorPrintln(Green, "✓ Removed govm binary")
			}
		} else {
			if err := removeFile(govmBinaryPath, false); err != nil {
				return fmt.Errorf("failed to remove govm binary: %v", err)
			} else {
				colorPrintln(Green, "✓ Removed govm binary")
			}
		}
	} else {
		colorPrintln(Red, fmt.Sprintf("govm binary not found in %s", govmBinDir))
	}

	// Remove govm directories
	colorPrintln(Blue, "Removing govm directories...")
	if _, err := os.Stat(govmDir); err == nil {
		if err := os.RemoveAll(govmDir); err != nil {
			return fmt.Errorf("failed to remove govm directory: %v", err)
		} else {
			colorPrintln(Green, "✓ Removed govm directories")
		}
	} else {
		colorPrintln(Red, fmt.Sprintf("govm directory not found at %s", govmDir))
	}

	// Clean shell profile
	colorPrintln(Blue, fmt.Sprintf("Updating shell profile (%s)...", shellProfile))
	if _, err := os.Stat(shellProfile); err == nil {
		if err := cleanShellProfile(shellProfile); err != nil {
			return fmt.Errorf("failed to update shell profile: %v", err)
		} else {
			colorPrintln(Green, "✓ Updated shell profile")
		}
	} else {
		return fmt.Errorf("shell profile not found at %s", shellProfile)
	}

	// Final success message
	colorPrintln(Green+Bold, "✓ govm has been successfully removed from your system!")
	colorPrintln(Blue, "To ensure all changes take effect, please restart your terminal or run:")
	_, _ = fmt.Fprintf(os.Stdout, "    source %s\n\n", shellProfile)
	_, _ = fmt.Fprintln(os.Stdout, "Thank you for using govm!")

	return nil
}
