//go:build !windows

package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallCopiesBinaryAndUpdatesShellConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	t.Setenv("PATH", "/usr/bin:/bin")

	source := filepath.Join(t.TempDir(), "glyph-source")
	if err := os.WriteFile(source, []byte("glyph"), 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := Install(source, "")
	if err != nil {
		t.Fatal(err)
	}

	if !result.Installed {
		t.Fatalf("Installed = false, want true")
	}
	if result.BinDir != filepath.Join(home, ".local", "bin") {
		t.Fatalf("BinDir = %q", result.BinDir)
	}
	if got, err := os.ReadFile(result.InstalledPath); err != nil || string(got) != "glyph" {
		t.Fatalf("installed binary = %q, %v", got, err)
	}
	if !result.PathConfigured {
		t.Fatalf("PathConfigured = false, want true")
	}
	if result.PathAlreadyConfigured {
		t.Fatalf("PathAlreadyConfigured = true, want false")
	}
	zshrc := filepath.Join(home, ".zshrc")
	rc, err := os.ReadFile(zshrc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rc), "BEGIN GLYPH PATH") || !strings.Contains(string(rc), `$HOME/.local/bin`) {
		t.Fatalf(".zshrc missing glyph PATH block:\n%s", rc)
	}
}

func TestInstallLeavesPathWhenAlreadyConfigured(t *testing.T) {
	home := t.TempDir()
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	t.Setenv("PATH", binDir+":/usr/bin:/bin")

	source := filepath.Join(t.TempDir(), "glyph-source")
	if err := os.WriteFile(source, []byte("glyph"), 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := Install(source, binDir)
	if err != nil {
		t.Fatal(err)
	}

	if !result.PathAlreadyConfigured {
		t.Fatalf("PathAlreadyConfigured = false, want true")
	}
	if result.PathConfigured {
		t.Fatalf("PathConfigured = true, want false")
	}
	if len(result.ModifiedFiles) != 0 {
		t.Fatalf("ModifiedFiles = %#v, want none", result.ModifiedFiles)
	}
}
