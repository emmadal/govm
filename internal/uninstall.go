package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/emmadal/govm/pkg"
)

// isGovmRoot guards against deleting an unrelated directory when GOVM_DIR
// is misconfigured: the root must be named .govm or contain versions/go.
func isGovmRoot(p pkg.Paths) bool {
	if home, err := os.UserHomeDir(); err == nil && filepath.Clean(p.Root) == filepath.Clean(home) {
		return false
	}
	if filepath.Base(p.Root) == ".govm" {
		return true
	}
	info, err := os.Stat(p.Versions)
	return err == nil && info.IsDir()
}

// Uninstall removes govm, every Go version it installed, and its PATH setup.
func Uninstall(assumeYes bool) error {
	p, err := pkg.ResolvePaths()
	if err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to locate the govm binary: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	if !assumeYes {
		pkg.Warn("This will completely remove govm from your system, including:")
		pkg.Warn("  - the govm binary (%s)", exe)
		pkg.Warn("  - all Go versions installed with govm (%s)", p.Root)
		pkg.Warn("  - govm's PATH entries in your shell profile")
		ok, err := pkg.Confirm("Are you sure you want to proceed?")
		if err != nil {
			return err
		}
		if !ok {
			pkg.Println("Uninstall cancelled.")
			return nil
		}
	}

	changed, err := pkg.CleanShellSetup(p.Root)
	for _, c := range changed {
		pkg.Info("Updated %s", c)
	}
	if err != nil {
		return err
	}

	exeInRoot := strings.HasPrefix(exe, p.Root+string(filepath.Separator))
	if _, err := os.Stat(p.Root); err == nil {
		if !isGovmRoot(p) {
			return fmt.Errorf("refusing to delete %s: it does not look like a govm directory", p.Root)
		}
		// On Windows the running binary cannot be deleted; RemoveAll still
		// removes everything else before reporting that failure.
		lockedExe := runtime.GOOS == "windows" && exeInRoot
		if err := os.RemoveAll(p.Root); err != nil && !lockedExe {
			return fmt.Errorf("failed to remove %s: %w", p.Root, err)
		}
		pkg.Info("Removed %s", p.Root)
	}

	if runtime.GOOS == "windows" {
		pkg.Warn("Windows cannot delete a running program. Delete %s manually to finish.", exe)
	} else if !exeInRoot {
		if err := os.Remove(exe); err != nil && !os.IsNotExist(err) {
			return permissionHint(err, exe)
		}
		// VERSION file written next to the binary by older installers.
		versionFile := filepath.Join(filepath.Dir(exe), "VERSION")
		if data, err := os.ReadFile(versionFile); err == nil && strings.HasPrefix(string(data), "v") {
			_ = os.Remove(versionFile)
		}
		pkg.Info("Removed %s", exe)
	}

	pkg.Success("govm has been removed. Open a new terminal to refresh your PATH.")
	return nil
}
