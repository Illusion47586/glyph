package store

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPrototypeSpine(t *testing.T) {
	_, st := newTestStore(t)
	defer st.Close()

	count, err := st.ImportWorkspace()
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if count != 3 {
		t.Fatalf("import count = %d, want 3", count)
	}

	public, err := st.SourcesForRealm("public")
	if err != nil {
		t.Fatalf("public sources: %v", err)
	}
	if len(public) != 3 {
		t.Fatalf("public sources = %d, want 3", len(public))
	}

	if _, err := st.StartWork("docs", "public"); err != nil {
		t.Fatalf("start work: %v", err)
	}
	if err := st.WriteWorkFile("docs", "docs/specs/README.md", []byte("# Specs\n\nChanged.\n"), "test edit"); err != nil {
		t.Fatalf("write work file: %v", err)
	}
	diff, err := st.DiffWork("docs")
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if len(diff) != 1 || diff[0] != "M docs/specs/README.md" {
		t.Fatalf("diff = %#v, want modified readme", diff)
	}
	pub, err := st.PublishWithMode("docs", "public", "squash")
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if !strings.HasPrefix(pub, "publication:") {
		t.Fatalf("publication id = %q", pub)
	}

	wc, err := st.Work("docs")
	if err != nil {
		t.Fatalf("work after publish: %v", err)
	}
	if err := st.PruneWork("docs"); err != nil {
		t.Fatalf("prune: %v", err)
	}
	if _, err := os.Stat(wc.WorkspacePath); !os.IsNotExist(err) {
		t.Fatalf("workspace path still exists after prune: %v", err)
	}
}

func TestConcurrencyClaimsDependenciesAndConflicts(t *testing.T) {
	_, st := newTestStore(t)
	defer st.Close()
	if _, err := st.ImportWorkspace(); err != nil {
		t.Fatalf("import: %v", err)
	}
	if _, err := st.StartWork("agent-a", "public"); err != nil {
		t.Fatalf("start agent-a: %v", err)
	}
	if _, err := st.StartWork("agent-b", "public"); err != nil {
		t.Fatalf("start agent-b: %v", err)
	}

	claim, err := st.ClaimWork("agent-a", "agent:codex:session-01", "exclusive", time.Minute)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if claim.Provider != "codex" || claim.SessionID != "session-01" {
		t.Fatalf("claim identity = %#v", claim)
	}
	if _, err := st.ClaimWork("agent-a", "agent:claude-code:session-02", "exclusive", time.Minute); err == nil {
		t.Fatalf("second exclusive claim succeeded, want conflict")
	}
	if _, err := st.HeartbeatWork("agent-a", "agent:codex:session-01", time.Minute); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	if err := st.ReleaseWork("agent-a", "agent:codex:session-01"); err != nil {
		t.Fatalf("release: %v", err)
	}
	if _, err := st.ClaimWork("agent-a", "agent:claude-code:session-02", "exclusive", time.Minute); err != nil {
		t.Fatalf("claim after release: %v", err)
	}

	dep, err := st.AddDependency("agent-b", "agent-a")
	if err != nil {
		t.Fatalf("dependency: %v", err)
	}
	if dep.WorkName != "agent-b" || dep.DependsOnWork != "agent-a" {
		t.Fatalf("dependency = %#v", dep)
	}

	if err := st.WriteWorkFile("agent-a", "docs/specs/README.md", []byte("# Specs\n\nA.\n"), "agent a edit"); err != nil {
		t.Fatalf("write agent-a: %v", err)
	}
	if err := st.WriteWorkFile("agent-b", "docs/specs/README.md", []byte("# Specs\n\nB.\n"), "agent b edit"); err != nil {
		t.Fatalf("write agent-b: %v", err)
	}
	conflicts, err := st.WorkConflicts("agent-a")
	if err != nil {
		t.Fatalf("conflicts: %v", err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("conflicts = %#v, want one conflict", conflicts)
	}
	if conflicts[0].Type != "content" || conflicts[0].Path != "docs/specs/README.md" {
		t.Fatalf("conflict = %#v", conflicts[0])
	}
	if _, err := st.PublishWithMode("agent-a", "public", "preserve"); err == nil {
		t.Fatalf("publish with unresolved conflict succeeded")
	}
}

func TestPublishHooks(t *testing.T) {
	_, st := newTestStore(t)
	defer st.Close()
	if _, err := st.ImportWorkspace(); err != nil {
		t.Fatalf("import: %v", err)
	}
	if _, err := st.StartWork("blocked", "public"); err != nil {
		t.Fatalf("start blocked: %v", err)
	}
	writeExecutable(t, filepath.Join(st.Dir, "hooks", "pre-publish"), "#!/bin/sh\necho blocked >&2\nexit 7\n")
	if _, err := st.PublishWithMode("blocked", "public", "squash"); err == nil {
		t.Fatalf("publish succeeded with failing pre-publish hook")
	}
	var blocked int
	if err := st.DB.QueryRow(`SELECT blocked FROM hook_runs WHERE event = 'pre-publish'`).Scan(&blocked); err != nil {
		t.Fatalf("hook run row: %v", err)
	}
	if blocked != 1 {
		t.Fatalf("blocked = %d, want 1", blocked)
	}

	if err := os.Remove(filepath.Join(st.Dir, "hooks", "pre-publish")); err != nil {
		t.Fatalf("remove pre-publish: %v", err)
	}
	writeExecutable(t, filepath.Join(st.Dir, "hooks", "post-publish"), "#!/bin/sh\n[ \"$GLYPH_PUBLICATION_ID\" != \"\" ] || exit 3\necho post ok\n")
	if _, err := st.StartWork("allowed", "public"); err != nil {
		t.Fatalf("start allowed: %v", err)
	}
	if _, err := st.PublishWithMode("allowed", "public", "preserve"); err != nil {
		t.Fatalf("publish with successful hooks: %v", err)
	}
	var postRuns int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM hook_runs WHERE event = 'post-publish' AND exit_code = 0`).Scan(&postRuns); err != nil {
		t.Fatalf("post-publish count: %v", err)
	}
	if postRuns != 1 {
		t.Fatalf("post-publish runs = %d, want 1", postRuns)
	}
}

func TestExportGitGeneratesCompatibilityFiles(t *testing.T) {
	root, st := newTestStore(t)
	defer st.Close()
	if _, err := st.ImportWorkspace(); err != nil {
		t.Fatalf("import: %v", err)
	}
	out := filepath.Join(root, "export")
	result, err := st.ExportGitWithOptions("public", out, GitExportOptions{Gitignore: GitCompatGenerated, Gitinclude: GitCompatGenerated})
	if err != nil {
		if _, gitErr := exec.LookPath("git"); gitErr != nil {
			t.Skip("git not installed")
		}
		t.Fatalf("export git: %v", err)
	}
	if len(result.Generated) != 2 {
		t.Fatalf("generated = %#v, want .gitignore and .gitinclude", result.Generated)
	}
	gitignore := read(t, filepath.Join(out, ".gitignore"))
	for _, want := range []string{".glyph/**", "node_modules/**"} {
		if !strings.Contains(gitignore, want) {
			t.Fatalf(".gitignore missing %q:\n%s", want, gitignore)
		}
	}
	if strings.Contains(gitignore, ".git/**") {
		t.Fatalf(".gitignore should not include Git's own metadata directory:\n%s", gitignore)
	}
	gitinclude := read(t, filepath.Join(out, ".gitinclude"))
	for _, want := range []string{"glyph.yaml", "docs/specs/**", "internal/**"} {
		if !strings.Contains(gitinclude, want) {
			t.Fatalf(".gitinclude missing %q:\n%s", want, gitinclude)
		}
	}
}

func newTestStore(t *testing.T) (string, *Store) {
	t.Helper()
	root := t.TempDir()
	write(t, filepath.Join(root, "glyph.yaml"), `project:
  name: glyph-test
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
    - docs/specs/**
    - internal/**
  exclude:
    - .git/**
    - .glyph/**
    - node_modules/**
    - .DS_Store
  transcripts:
    default: exclude
defaults:
  export:
    git:
      gitignore: generated
      gitinclude: generated
  viz:
    export:
      out: .glyph/visualizer
`)
	write(t, filepath.Join(root, "docs/specs/README.md"), "# Specs\n")
	write(t, filepath.Join(root, "internal/app.go"), "package internal\n")
	write(t, filepath.Join(root, ".env"), "SECRET=yes\n")

	st, err := Init(root)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	return root, st
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write executable %s: %v", path, err)
	}
}
