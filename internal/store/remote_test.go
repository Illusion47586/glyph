package store

import (
	"os/exec"
	"strings"
	"testing"
)

func TestRemoteURL(t *testing.T) {
	tests := map[string]string{
		"github:owner/repo":             "git@github.com:owner/repo.git",
		"https://github.com/owner/repo": "https://github.com/owner/repo",
		"git@github.com:owner/repo.git": "git@github.com:owner/repo.git",
		"not-a-remote":                  "",
	}
	for spec, want := range tests {
		if got := remoteURL(spec); got != want {
			t.Fatalf("remoteURL(%q) = %q, want %q", spec, got, want)
		}
	}
}

func TestEnsureGitAvailableReportsMissingGit(t *testing.T) {
	originalLookPath := lookPath
	t.Cleanup(func() { lookPath = originalLookPath })
	lookPath = func(file string) (string, error) {
		if file != "git" {
			t.Fatalf("lookPath called with %q", file)
		}
		return "", exec.ErrNotFound
	}

	err := ensureGitAvailable()
	if err == nil || !strings.Contains(err.Error(), "git is required") {
		t.Fatalf("ensureGitAvailable error = %v", err)
	}
}

func TestGitExportCommitMessageIncludesPublicationTrailers(t *testing.T) {
	pub := &exportPublication{
		ID:      "publication:123",
		Work:    "git-export-links",
		Realm:   "public",
		Mode:    "squash",
		Created: "2026-05-27T07:09:33Z",
	}
	subject, body := gitExportCommitMessage("public", pub, "2026-05-31T12:00:00Z")
	if subject != "Publish git-export-links to public" {
		t.Fatalf("subject = %q", subject)
	}
	for _, want := range []string{
		"Glyph-Publication: publication:123",
		"Glyph-Work: git-export-links",
		"Glyph-Realm: public",
		"Glyph-Mode: squash",
		"Glyph-Published-At: 2026-05-27T07:09:33Z",
		"Glyph-Exported-At: 2026-05-31T12:00:00Z",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q:\n%s", want, body)
		}
	}
}

func TestGitExportCommitMessageWithoutPublication(t *testing.T) {
	subject, body := gitExportCommitMessage("public", nil, "2026-05-31T12:00:00Z")
	if subject != "Export Glyph public projection" {
		t.Fatalf("subject = %q", subject)
	}
	for _, want := range []string{"Glyph-Realm: public", "Glyph-Exported-At: 2026-05-31T12:00:00Z"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q:\n%s", want, body)
		}
	}
}
