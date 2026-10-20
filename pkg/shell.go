package pkg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// Markers around the lines govm owns in a shell profile. Everything between
// them is added and removed as a unit; nothing else in the file is touched.
const (
	blockStart = "# >>> govm >>>"
	blockEnd   = "# <<< govm <<<"
)

// Lines written by older govm releases, removed when found.
var (
	// One of these was appended on every `govm use`.
	legacyVersionPathRe = regexp.MustCompile(`^export PATH="[^"]*\.govm/versions/go/go[^/"]+/bin:\$PATH"$`)
	// The old install.sh wrote this comment followed by legacyInstallerPath.
	legacyInstallerComment = "# govm installation"
	legacyInstallerPath    = `export PATH="${HOME}/.local/bin:${PATH}"`
)

// ProfilePath returns the profile file govm writes to for the user's shell.
// It returns "" on Windows, where PATH lives in the user environment instead.
func ProfilePath() (string, error) {
	if runtime.GOOS == "windows" {
		return "", nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch filepath.Base(os.Getenv("SHELL")) {
	case "zsh":
		return filepath.Join(home, ".zshrc"), nil
	case "bash":
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, ".bash_profile"), nil
		}
		return filepath.Join(home, ".bashrc"), nil
	case "fish":
		return filepath.Join(home, ".config", "fish", "conf.d", "govm.fish"), nil
	default:
		return filepath.Join(home, ".profile"), nil
	}
}

// candidateProfiles lists every file govm may have written to.
func candidateProfiles() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".bash_profile"),
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".profile"),
		filepath.Join(home, ".config", "fish", "config.fish"),
		filepath.Join(home, ".config", "fish", "conf.d", "govm.fish"),
	}
}

// shellBlock returns the govm block that puts binDir first on PATH.
func shellBlock(profile, binDir string) string {
	if home, err := os.UserHomeDir(); err == nil {
		if rel, err := filepath.Rel(home, binDir); err == nil && filepath.IsLocal(rel) {
			binDir = "$HOME/" + filepath.ToSlash(rel)
		}
	}
	line := fmt.Sprintf(`export PATH="%s:$PATH"`, binDir)
	if strings.HasSuffix(profile, ".fish") {
		line = fmt.Sprintf(`fish_add_path --global --move --path "%s"`, binDir)
	}
	return blockStart + "\n" + line + "\n" + blockEnd + "\n"
}

// removeGovmLines strips the govm block and any lines written by older
// releases from a profile's content. The legacy installer lines are only
// removed when uninstalling, since they put govm itself on PATH.
func removeGovmLines(content string, uninstall bool) string {
	lines := strings.SplitAfter(content, "\n")
	out := make([]string, 0, len(lines))
	inBlock := false
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		switch {
		case uninstall && line == blockStart:
			inBlock = true
		case inBlock:
			if line == blockEnd {
				inBlock = false
			}
		case legacyVersionPathRe.MatchString(line):
		case uninstall && line == legacyInstallerComment:
			if i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == legacyInstallerPath {
				i++
			}
		default:
			out = append(out, lines[i])
		}
	}
	return strings.Join(out, "")
}

func hasBlock(content string) bool {
	return strings.Contains(content, blockStart)
}

// writeProfile atomically replaces a profile's content, following symlinks
// so dotfile managers keep working and keeping the file's permissions.
func writeProfile(path, content string) error {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".govm-profile-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), mode); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func readProfile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	return string(data), err
}

// EnsureShellSetup makes sure future shells have the active version's bin
// directory on PATH. It also removes PATH lines left by older govm releases,
// which would otherwise shadow it. It returns the file it modified, if any.
func EnsureShellSetup(p Paths) (string, error) {
	if runtime.GOOS == "windows" {
		changed, err := addWindowsUserPath(p.CurrentBin())
		if err != nil || !changed {
			return "", err
		}
		return "user PATH environment variable", nil
	}

	found := false
	modified := ""
	for _, profile := range candidateProfiles() {
		content, err := readProfile(profile)
		if err != nil {
			return "", err
		}
		found = found || hasBlock(content)
		if cleaned := removeGovmLines(content, false); cleaned != content {
			if err := writeProfile(profile, cleaned); err != nil {
				return "", fmt.Errorf("failed to update %s: %w", profile, err)
			}
			modified = profile
		}
	}
	if found {
		return modified, nil
	}

	profile, err := ProfilePath()
	if err != nil {
		return "", err
	}
	content, err := readProfile(profile)
	if err != nil {
		return "", err
	}
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += "\n" + shellBlock(profile, p.CurrentBin())
	if err := writeProfile(profile, content); err != nil {
		return "", fmt.Errorf("failed to update %s: %w", profile, err)
	}
	return profile, nil
}

// CleanShellSetup removes everything govm added to shell profiles (or, on
// Windows, the user PATH entries under root). It returns the files changed.
func CleanShellSetup(root string) ([]string, error) {
	if runtime.GOOS == "windows" {
		changed, err := removeWindowsUserPath(root)
		if err != nil || !changed {
			return nil, err
		}
		return []string{"user PATH environment variable"}, nil
	}

	var changed []string
	for _, profile := range candidateProfiles() {
		content, err := readProfile(profile)
		if err != nil {
			return changed, err
		}
		if cleaned := removeGovmLines(content, true); cleaned != content {
			if err := writeProfile(profile, cleaned); err != nil {
				return changed, fmt.Errorf("failed to update %s: %w", profile, err)
			}
			changed = append(changed, profile)
		}
	}
	return changed, nil
}

// runPowerShell runs a script with the given extra environment variables,
// which is how values are passed in without any quoting concerns.
func runPowerShell(script string, env ...string) (string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("powershell: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func addWindowsUserPath(dir string) (bool, error) {
	out, err := runPowerShell(`
$entry = $env:GOVM_PATH_ENTRY
$path = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($null -eq $path) { $path = '' }
$parts = @($path -split ';' | Where-Object { $_ -ne '' })
if ($parts -contains $entry) { 'unchanged' } else {
  [Environment]::SetEnvironmentVariable('Path', ((@($entry) + $parts) -join ';'), 'User')
  'changed'
}`, "GOVM_PATH_ENTRY="+dir)
	return out == "changed", err
}

func removeWindowsUserPath(root string) (bool, error) {
	out, err := runPowerShell(`
$root = $env:GOVM_ROOT.TrimEnd('\')
$path = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($null -eq $path) { $path = '' }
$parts = @($path -split ';' | Where-Object { $_ -ne '' })
$kept = @($parts | Where-Object { $_ -ne $root -and -not $_.StartsWith($root + '\', [StringComparison]::OrdinalIgnoreCase) })
if ($kept.Count -eq $parts.Count) { 'unchanged' } else {
  [Environment]::SetEnvironmentVariable('Path', ($kept -join ';'), 'User')
  'changed'
}`, "GOVM_ROOT="+root)
	return out == "changed", err
}
