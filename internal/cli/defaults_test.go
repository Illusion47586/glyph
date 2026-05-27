package cli

import (
	"os"
	"path/filepath"
	"testing"

	"glyph/internal/store"
)

func TestCommandDefaultsLoadsGlyphYAML(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, filepath.Join(root, "glyph.yaml"), `project:
  name: defaults-test
  status: bootstrap
specs:
  directory: docs/specs
realms:
  default_public: public
  default_private: maintainers
identity:
  bootstrap_user: user:self:test
genesis_import:
  include:
    - glyph.yaml
  exclude:
    - .glyph/**
  transcripts:
    default: exclude
defaults:
  export:
    git:
      gitignore: generated
      gitinclude: overwrite
  viz:
    export:
      out: .glyph/visualizer
`)
	st, err := store.Init(root)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	t.Chdir(root)

	defaults, err := commandDefaults()
	if err != nil {
		t.Fatalf("command defaults: %v", err)
	}
	if defaults.Defaults.Export.Git.Gitignore != "generated" {
		t.Fatalf("gitignore default = %q", defaults.Defaults.Export.Git.Gitignore)
	}
	if defaults.Defaults.Export.Git.Gitinclude != "overwrite" {
		t.Fatalf("gitinclude default = %q", defaults.Defaults.Export.Git.Gitinclude)
	}
	if defaults.Defaults.Viz.Export.Out != ".glyph/visualizer" {
		t.Fatalf("viz out default = %q", defaults.Defaults.Viz.Export.Out)
	}
}

func writeCLIFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
