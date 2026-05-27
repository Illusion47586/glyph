package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateCompatFileModes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")

	result, err := generateCompatFile(path, GitCompatNone, "ignored\n")
	if err != nil {
		t.Fatalf("none: %v", err)
	}
	if result.Generated || result.Skipped {
		t.Fatalf("none result = %#v", result)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("none created file: %v", err)
	}

	result, err = generateCompatFile(path, GitCompatGenerated, "first\n")
	if err != nil {
		t.Fatalf("generated: %v", err)
	}
	if !result.Generated || result.Skipped {
		t.Fatalf("generated result = %#v", result)
	}
	if got := read(t, path); got != "first\n" {
		t.Fatalf("generated content = %q", got)
	}

	result, err = generateCompatFile(path, GitCompatGenerated, "second\n")
	if err != nil {
		t.Fatalf("generated skip: %v", err)
	}
	if result.Generated || !result.Skipped {
		t.Fatalf("generated skip result = %#v", result)
	}
	if got := read(t, path); got != "first\n" {
		t.Fatalf("generated overwrote content = %q", got)
	}

	result, err = generateCompatFile(path, GitCompatOverwrite, "third\n")
	if err != nil {
		t.Fatalf("overwrite: %v", err)
	}
	if !result.Generated || result.Skipped {
		t.Fatalf("overwrite result = %#v", result)
	}
	if got := read(t, path); got != "third\n" {
		t.Fatalf("overwrite content = %q", got)
	}
}

func TestGitExportOptionsNormalize(t *testing.T) {
	opts, err := (GitExportOptions{}).normalized()
	if err != nil {
		t.Fatalf("normalize empty: %v", err)
	}
	if opts.Gitignore != GitCompatNone || opts.Gitinclude != GitCompatNone {
		t.Fatalf("defaults = %#v", opts)
	}

	_, err = (GitExportOptions{Gitignore: "surprise"}).normalized()
	if err == nil || !strings.Contains(err.Error(), "unsupported gitignore mode") {
		t.Fatalf("invalid mode error = %v", err)
	}
}

func TestGeneratedPatternFilesDedupeAndTrim(t *testing.T) {
	content := generatedPatternFile("Test patterns.", []string{" one ", "two", "one", ""})
	if strings.Count(content, "one\n") != 1 {
		t.Fatalf("expected one deduped entry:\n%s", content)
	}
	if !strings.Contains(content, "# Test patterns.") {
		t.Fatalf("missing note:\n%s", content)
	}
}
