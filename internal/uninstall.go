package internal

import (
	"bufio"
	"fmt"
	"github.com/emmadal/govm/pkg"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// getGovmExecDir returns the directory where govm is installed
func getGovmExecDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(homeDir, ".govm"), nil
	}
	return filepath.Join(homeDir, ".local", "bin", "govm"), nil
}

// detectShellProfile finds the appropriate shell profile file
func detectShellProfile() string {
	// Windows doesn't traditionally use shell profiles like Unix systems
	if runtime.GOOS == "windows" {
		// For PowerShell users, find their profile
		cmd := exec.Command(
			"powershell", "-Command",
			"if (!(Test-Path -Path $PROFILE)) { Write-Output 'none' } else { Write-Output $PROFILE }",
		)
		output, err := cmd.Output()
		if err == nil {
			profile := strings.TrimSpace(string(output))
			if profile != "none" && profile != "" {
				return profile
			}
		}
		// Return an empty string if no PowerShell profile exists
		return ""
	}

	// Unix systems
	homedir, err := os.UserHomeDir()
	if err != nil {
		pkg.RedPrintln(err.Error())
	}

	shell := os.Getenv("SHELL")
	shellName := filepath.Base(shell)

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

// cleanShellProfile removes govm-related lines from the shell profile
func cleanShellProfile(profilePath string) error {
	// Skip for an empty profile path (Windows without PowerShell profile)
	if profilePath == "" {
		return nil
	}

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

	// Adjust patterns for the current platform
	govmPatterns := []*regexp.Regexp{
		regexp.MustCompile(`# govm installation`),
		regexp.MustCompile(`export PATH=.*\.local/bin.*`),
		regexp.MustCompile(`export PATH=.*GOVM_DIR.*versions.*go`),
		regexp.MustCompile(`export GOROOT=`),
	}

	// Add Windows-specific patterns
	if runtime.GOOS == "windows" {
		govmPatterns = append(
			govmPatterns,
			regexp.MustCompile(`\$env:Path.*govm.*`),
			regexp.MustCompile(`\$env:GOROOT.*`),
		)
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
	pkg.RedPrintln("Removing govm - Go Version Manager\n")

	// Get platform-specific installation directories
	govmDir, err := getGovmExecDir()
	if err != nil {
		return err
	}

	// Detect shell profile (maybe empty on Windows)
	shellProfile := detectShellProfile()

	// Remove govm
	pkg.BluePrintln("Removing govm...\n")
	if _, err := os.Stat(govmDir); err == nil {
		if err := os.RemoveAll(govmDir); err != nil {
			return fmt.Errorf("failed to remove govm directory: %v", err)
		} else {
			pkg.GreenPrintln("✓ Removed govm directories\n")
		}
	} else {
		pkg.RedPrintln(fmt.Sprintf("govm directory not found at %s", govmDir) + "\n")
	}

	// Clean shell profile (if it exists - may not exist on Windows)
	if shellProfile != "" {
		pkg.BluePrintln(fmt.Sprintf("Updating shell profile (%s)...", shellProfile) + "\n")
		if _, err := os.Stat(shellProfile); err == nil {
			if err := cleanShellProfile(shellProfile); err != nil {
				return fmt.Errorf("failed to update shell profile: %v", err)
			} else {
				pkg.GreenPrintln("✓ Updated shell profile\n")
			}
		} else {
			pkg.RedPrintln(fmt.Sprintf("Shell profile not found at %s", shellProfile) + "\n")
		}
	}

	// Final success message
	pkg.GreenPrintln("✓ govm has been successfully removed from your system!\n")

	// Platform-specific instructions for applying changes
	if runtime.GOOS == "windows" {
		if shellProfile != "" {
			pkg.BluePrintln("Please restart your PowerShell or run in PowerShell:" + "\n")
			pkg.BlackPrintln("    . " + shellProfile + "\n\n")
		} else {
			pkg.BluePrintln("Please restart your terminal.\n\n")
		}
	} else {
		pkg.BluePrintln("Please restart your terminal or run in your shell:" + "\n")
		pkg.BlackPrintln("    source " + shellProfile + "\n\n")
	}

	pkg.BlackPrintln("Thank you for using govm!")
	return nil
}

// UninstallConfirm prompts the user for confirmation to remove govm
func UninstallConfirm() (bool, error) {
	var sb strings.Builder

	// Build the confirmation message in memory using strings.Builder
	fmt.Fprintf(&sb, pkg.TextRed("This will completely remove govm from your system, including:"))
	fmt.Fprintln(&sb, "  - The govm binary")
	fmt.Fprintln(&sb, "  - All installed Go versions managed by govm")
	fmt.Fprintln(&sb, "  - All govm configuration files")
	fmt.Fprintln(os.Stdout, sb.String())

	// Ask for user confirmation
	var reply string

	// Add the prompt to the builder
	fmt.Fprint(os.Stdout, "Are you sure you want to proceed? (y/n): ")
	_, err := fmt.Scanln(&reply)
	if err != nil {
		return false, fmt.Errorf("failed to read input: %v", err)
	}

	reply = strings.ToLower(strings.TrimSpace(reply))

	// Handle valid responses
	if reply == "y" {
		return true, nil
	} else if reply == "n" {
		fmt.Fprintln(os.Stdout, "Removal cancelled.")
		return false, nil
	} else {
		// Invalid input, re-prompt
		fmt.Fprintln(os.Stdout, "Invalid input. Please enter 'y' to confirm or 'n' to cancel.")
		return false, nil
	}
}
