package pkg

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// InstalledVersions returns the installed versions, newest first.
func InstalledVersions(p Paths) ([]string, error) {
	entries, err := os.ReadDir(p.Versions)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", p.Versions, err)
	}
	var versions []string
	for _, e := range entries {
		v, ok := strings.CutPrefix(e.Name(), "go")
		if !e.IsDir() || !ok || !versionRe.MatchString(v) {
			continue
		}
		versions = append(versions, v)
	}
	slices.SortFunc(versions, func(a, b string) int { return CompareVersions(b, a) })
	return versions, nil
}

// ActiveVersion returns the version the current link points to, or "" if
// none is active.
func ActiveVersion(p Paths) (string, error) {
	target, err := os.Readlink(p.Current)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", p.Current, err)
	}
	return strings.TrimPrefix(filepath.Base(filepath.Clean(target)), "go"), nil
}

// setCurrent points the current link at an installed version.
func setCurrent(p Paths, version string) error {
	target := p.VersionDir(version)
	if runtime.GOOS == "windows" {
		// Symlinks need Developer Mode or admin rights on Windows, so fall
		// back to a directory junction, which any user can create.
		if err := os.Remove(p.Current); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("failed to remove %s: %w", p.Current, err)
		}
		if err := os.Symlink(target, p.Current); err == nil {
			return nil
		}
		if out, err := exec.Command("cmd", "/c", "mklink", "/J", p.Current, target).CombinedOutput(); err != nil {
			return fmt.Errorf("failed to link %s: %v: %s", p.Current, err, strings.TrimSpace(string(out)))
		}
		return nil
	}

	// Relative link, so the whole govm directory can be moved.
	rel, err := filepath.Rel(p.Root, target)
	if err != nil {
		return err
	}
	// Create the new link beside the old one, then rename it into place,
	// so there is never a moment without an active version.
	tmp := p.Current + ".tmp"
	_ = os.Remove(tmp)
	if err := os.Symlink(rel, tmp); err != nil {
		return fmt.Errorf("failed to create link: %w", err)
	}
	if err := os.Rename(tmp, p.Current); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("failed to switch version: %w", err)
	}
	return nil
}

// resolveInstalled maps user input ("1.22.3", "go1.22", "latest") to an
// installed version.
func resolveInstalled(p Paths, input string) (string, error) {
	installed, err := InstalledVersions(p)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(input) == "latest" {
		if len(installed) == 0 {
			return "", fmt.Errorf("no Go versions installed. Run 'govm install latest'")
		}
		return installed[0], nil
	}
	v, err := NormalizeVersion(input)
	if err != nil {
		return "", err
	}
	if slices.Contains(installed, v) {
		return v, nil
	}
	if isMinorLine(v) {
		for _, iv := range installed {
			if inMinorLine(iv, v) {
				return iv, nil
			}
		}
	}
	return "", fmt.Errorf("go%s is not installed. Run 'govm install %s' first", v, v)
}

// onPath reports whether dir is an entry of the current PATH.
func onPath(dir string) bool {
	for _, entry := range filepath.SplitList(os.Getenv("PATH")) {
		entry = filepath.Clean(entry)
		if entry == filepath.Clean(dir) || (runtime.GOOS == "windows" && strings.EqualFold(entry, filepath.Clean(dir))) {
			return true
		}
	}
	return false
}

// Install downloads, verifies and installs a Go version, then switches to it.
func Install(input string) error {
	p, err := ResolvePaths()
	if err != nil {
		return err
	}
	if err := p.EnsureDirs(); err != nil {
		return err
	}

	client := NewHTTPClient()
	releases, err := FetchReleases(client)
	if err != nil {
		return err
	}
	rel, err := ResolveRelease(input, releases)
	if err != nil {
		return err
	}
	version := rel.Number()

	if p.IsInstalled(version) {
		Success("%s is already installed", rel.Version)
		return Use(version)
	}

	file, err := rel.Archive(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}
	archive := filepath.Join(p.Cache, file.Filename)
	if sum, err := FileSHA256(archive); err != nil || !strings.EqualFold(sum, file.SHA256) {
		if err := Download(client, DownloadBaseURL+file.Filename, archive, file.SHA256, rel.Version); err != nil {
			return err
		}
	}

	Info("Installing %s...", rel.Version)
	if err := ExtractArchive(archive, p.VersionDir(version)); err != nil {
		return err
	}
	Success("Installed %s", rel.Version)
	return Use(version)
}

// Use makes an installed version the active one.
func Use(input string) error {
	p, err := ResolvePaths()
	if err != nil {
		return err
	}
	version, err := resolveInstalled(p, input)
	if err != nil {
		return err
	}
	if err := setCurrent(p, version); err != nil {
		return err
	}
	modified, err := EnsureShellSetup(p)
	if err != nil {
		return err
	}
	if modified != "" {
		Info("Updated %s", modified)
	}
	Success("Now using go%s", version)

	if !onPath(p.CurrentBin()) {
		if runtime.GOOS == "windows" {
			Println("Open a new terminal to start using it.")
			return nil
		}
		profile, _ := ProfilePath()
		Println("Open a new terminal, or run: source %s", profile)
		return nil
	}
	if goBin, err := exec.LookPath("go"); err == nil && filepath.Clean(filepath.Dir(goBin)) != filepath.Clean(p.CurrentBin()) {
		Warn("Warning: %s comes before govm on your PATH and will take precedence.", goBin)
	}
	return nil
}

// Remove deletes an installed version and its cached archive.
func Remove(input string, assumeYes bool) error {
	p, err := ResolvePaths()
	if err != nil {
		return err
	}
	version, err := NormalizeVersion(input)
	if err != nil {
		return err
	}
	if !p.IsInstalled(version) {
		return fmt.Errorf("go%s is not installed", version)
	}
	active, err := ActiveVersion(p)
	if err != nil {
		return err
	}
	if active == version {
		return fmt.Errorf("go%s is the active version. Switch to another one with 'govm use' first", version)
	}

	if !assumeYes {
		ok, err := Confirm(fmt.Sprintf("Remove go%s?", version))
		if err != nil {
			return err
		}
		if !ok {
			Println("Removal cancelled.")
			return nil
		}
	}

	if err := os.RemoveAll(p.VersionDir(version)); err != nil {
		return fmt.Errorf("failed to remove go%s: %w", version, err)
	}
	archives, _ := filepath.Glob(filepath.Join(p.Cache, fmt.Sprintf("go%s.%s-*", version, runtime.GOOS)))
	for _, a := range archives {
		_ = os.Remove(a)
	}
	Success("Removed go%s", version)
	return nil
}

// List prints the installed versions, marking the active one.
func List() error {
	p, err := ResolvePaths()
	if err != nil {
		return err
	}
	versions, err := InstalledVersions(p)
	if err != nil {
		return err
	}
	if len(versions) == 0 {
		Println("No Go versions installed. Run 'govm install latest' to get started.")
		return nil
	}
	active, err := ActiveVersion(p)
	if err != nil {
		return err
	}
	for _, v := range versions {
		if v == active {
			Println("%s", Green("* go"+v+" (active)"))
		} else {
			Println("  go%s", v)
		}
	}
	return nil
}

// ListRemote prints the versions available for download. By default only
// the newest patch of each stable minor line is shown.
func ListRemote(all bool) error {
	p, err := ResolvePaths()
	if err != nil {
		return err
	}
	releases, err := FetchReleases(NewHTTPClient())
	if err != nil {
		return err
	}
	installed, _ := InstalledVersions(p)
	active, _ := ActiveVersion(p)

	seen := map[string]bool{}
	for _, r := range releases {
		v := r.Number()
		if !all {
			pv, ok := parseVersion(v)
			line := fmt.Sprintf("%d.%d", pv.major, pv.minor)
			if !ok || !r.Stable || seen[line] {
				continue
			}
			seen[line] = true
		}
		switch {
		case v == active:
			Println("%s", Green("* go"+v+" (active)"))
		case slices.Contains(installed, v):
			Println("  go%s (installed)", v)
		default:
			Println("  go%s", v)
		}
	}
	return nil
}

// Current prints the active version.
func Current() error {
	p, err := ResolvePaths()
	if err != nil {
		return err
	}
	active, err := ActiveVersion(p)
	if err != nil {
		return err
	}
	if active == "" {
		return fmt.Errorf("no active Go version. Run 'govm use <version>'")
	}
	Println("go%s", active)
	return nil
}
