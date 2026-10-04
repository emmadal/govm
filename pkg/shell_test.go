package pkg

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRemoveGovmLines(t *testing.T) {
	content := strings.Join([]string{
		`alias ll="ls -l"`,
		`export PATH="$HOME/tools/.local/bin:$PATH"`,
		`export GOROOT=/usr/local/go`,
		`export PATH="/home/me/.govm/versions/go/go1.21.0/bin:$PATH"`,
		``,
		`# govm installation`,
		`export PATH="${HOME}/.local/bin:${PATH}"`,
		``,
		blockStart,
		`export PATH="$HOME/.govm/current/bin:$PATH"`,
		blockEnd,
		`echo done`,
		``,
	}, "\n")

	use := removeGovmLines(content, false)
	if strings.Contains(use, "versions/go/go1.21.0") {
		t.Error("legacy version PATH line kept")
	}
	for _, keep := range []string{"# govm installation", blockStart, "GOROOT", "tools/.local/bin"} {
		if !strings.Contains(use, keep) {
			t.Errorf("use removed %q", keep)
		}
	}

	uninstall := removeGovmLines(content, true)
	for _, gone := range []string{"govm", "${HOME}/.local/bin", "current/bin"} {
		if strings.Contains(uninstall, gone) {
			t.Errorf("uninstall kept %q:\n%s", gone, uninstall)
		}
	}
	for _, keep := range []string{`alias ll="ls -l"`, "GOROOT", "tools/.local/bin", "echo done"} {
		if !strings.Contains(uninstall, keep) {
			t.Errorf("uninstall removed user line %q", keep)
		}
	}
}

func TestShellSetupRoundTrip(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows stores PATH in the user environment, not a profile")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	p := Paths{Root: filepath.Join(home, ".govm"), Current: filepath.Join(home, ".govm", "current")}

	original := "export EDITOR=vim\n"
	rc := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rc, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}

	modified, err := EnsureShellSetup(p)
	if err != nil || modified != rc {
		t.Fatalf("EnsureShellSetup = %q, %v", modified, err)
	}
	data, _ := os.ReadFile(rc)
	if !strings.Contains(string(data), `export PATH="$HOME/.govm/current/bin:$PATH"`) {
		t.Fatalf("block missing:\n%s", data)
	}
	if info, _ := os.Stat(rc); info.Mode().Perm() != 0o600 {
		t.Errorf("permissions changed to %v", info.Mode().Perm())
	}

	// Idempotent.
	if modified, err := EnsureShellSetup(p); err != nil || modified != "" {
		t.Fatalf("second EnsureShellSetup = %q, %v", modified, err)
	}

	if _, err := CleanShellSetup(p.Root); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(rc)
	if strings.TrimSpace(string(data)) != strings.TrimSpace(original) {
		t.Fatalf("profile not restored:\n%s", data)
	}
}

func TestShellSetupFollowsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("profiles are not used on Windows")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/bash")
	dotfiles := filepath.Join(home, "dotfiles", "bashrc")
	if err := os.MkdirAll(filepath.Dir(dotfiles), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dotfiles, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(home, ".bashrc")
	if runtime.GOOS == "darwin" {
		link = filepath.Join(home, ".bash_profile")
	}
	if err := os.Symlink(dotfiles, link); err != nil {
		t.Fatal(err)
	}

	if _, err := EnsureShellSetup(Paths{Root: filepath.Join(home, ".govm"), Current: filepath.Join(home, ".govm", "current")}); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("profile symlink was replaced by a regular file")
	}
	if data, _ := os.ReadFile(dotfiles); !strings.Contains(string(data), blockStart) {
		t.Fatal("symlink target not updated")
	}
}

func TestShellBlockFish(t *testing.T) {
	block := shellBlock("/home/me/.config/fish/conf.d/govm.fish", "/opt/govm/current/bin")
	if !strings.Contains(block, `fish_add_path --global --move --path "/opt/govm/current/bin"`) {
		t.Fatalf("unexpected fish block:\n%s", block)
	}
}
