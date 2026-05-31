package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type RemoteSyncResult struct {
	Name      string           `json:"name"`
	Spec      string           `json:"spec"`
	Mode      string           `json:"mode"`
	URL       string           `json:"url"`
	GitCommit string           `json:"git_commit,omitempty"`
	Export    *GitExportResult `json:"export"`
	Exported  string           `json:"exported"`
}

type exportPublication struct {
	ID      string
	Work    string
	Realm   string
	Mode    string
	Created string
}

func (s *Store) AddRemote(name, spec, mode string) error {
	if mode != "export-only" {
		return fmt.Errorf("only export-only remotes are supported in prototype 0")
	}
	_, err := s.DB.Exec(`INSERT OR REPLACE INTO remotes(name, spec, mode, created_at) VALUES(?, ?, ?, ?)`, name, spec, mode, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return err
	}
	return s.AppendAudit("remote_added", BootstrapUser, map[string]any{"name": name, "spec": spec, "mode": mode})
}

func (s *Store) Remotes() ([]map[string]string, error) {
	rows, err := s.DB.Query(`SELECT name, spec, mode FROM remotes ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var remotes []map[string]string
	for rows.Next() {
		var name, spec, mode string
		if err := rows.Scan(&name, &spec, &mode); err != nil {
			return nil, err
		}
		remotes = append(remotes, map[string]string{"name": name, "spec": spec, "mode": mode})
	}
	return remotes, rows.Err()
}

func (s *Store) Remote(name string) (map[string]string, error) {
	var spec, mode string
	if err := s.DB.QueryRow(`SELECT spec, mode FROM remotes WHERE name = ?`, name).Scan(&spec, &mode); err != nil {
		return nil, err
	}
	return map[string]string{"name": name, "spec": spec, "mode": mode}, nil
}

func (s *Store) ExportGit(realm, out string) error {
	_, err := s.ExportGitWithOptions(realm, out, DefaultGitExportOptions())
	return err
}

func (s *Store) ExportGitWithOptions(realm, out string, opts GitExportOptions) (*GitExportResult, error) {
	opts, err := opts.normalized()
	if err != nil {
		return nil, err
	}
	if err := ensureEmptyOrMissing(out); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return nil, err
	}
	if err := s.MaterializeRealm(realm, out); err != nil {
		return nil, err
	}
	result, err := s.GenerateGitCompatibilityFiles(out, opts)
	if err != nil {
		return nil, err
	}
	result.Realm = realm
	result.Out = out
	publication, err := s.latestPublicationForRealm(realm)
	if err != nil {
		return nil, err
	}
	exportedAt := time.Now().UTC().Format(time.RFC3339)
	subject, body := gitExportCommitMessage(realm, publication, exportedAt)
	commands := [][]string{
		{"git", "init", "-b", "main"},
		{"git", "add", "."},
		{"git", "-c", "user.name=Glyph", "-c", "user.email=glyph@example.local", "commit", "-m", subject, "-m", body},
	}
	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = out
		if outBytes, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("%s failed: %w\n%s", strings.Join(args, " "), err, string(outBytes))
		}
	}
	commit, err := gitHead(out)
	if err != nil {
		return nil, err
	}
	result.GitCommit = commit
	auditData := map[string]any{"realm": realm, "out": out, "git_commit": commit, "generated": result.Generated, "skipped": result.Skipped}
	addPublicationAuditData(auditData, publication)
	if err := s.AppendAudit("git_exported", BootstrapUser, auditData); err != nil {
		return nil, err
	}
	return result, nil
}

func ensureEmptyOrMissing(path string) error {
	entries, err := os.ReadDir(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("export destination %s is not empty", path)
	}
	return nil
}

func (s *Store) SyncRemote(name string) error {
	_, err := s.SyncRemoteWithOptions(name, DefaultGitExportOptions())
	return err
}

func (s *Store) SyncRemoteWithOptions(name string, opts GitExportOptions) (*RemoteSyncResult, error) {
	remote, err := s.Remote(name)
	if err != nil {
		return nil, err
	}
	if remote["mode"] != "export-only" {
		return nil, fmt.Errorf("remote %s is not export-only", name)
	}
	out := filepath.Join(s.Dir, "exports", name)
	if err := os.RemoveAll(out); err != nil {
		return nil, err
	}
	export, err := s.ExportGitWithOptions("public", out, opts)
	if err != nil {
		return nil, err
	}
	publication, err := s.latestPublicationForRealm("public")
	if err != nil {
		return nil, err
	}
	url := remoteURL(remote["spec"])
	if url == "" {
		return nil, fmt.Errorf("unsupported remote spec %q", remote["spec"])
	}
	commands := [][]string{
		{"git", "remote", "add", "origin", url},
	}
	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = out
		if outBytes, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("%s failed: %w\n%s", strings.Join(args, " "), err, string(outBytes))
		}
	}
	subject, body := gitExportCommitMessage("public", publication, time.Now().UTC().Format(time.RFC3339))
	commit, err := attachRemoteMainParent(out, subject, body)
	if err != nil {
		return nil, err
	}
	export.GitCommit = commit
	push := exec.Command("git", "push", "-u", "origin", "main")
	push.Dir = out
	if outBytes, err := push.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("git push -u origin main failed: %w\n%s", err, string(outBytes))
	}
	auditData := map[string]any{"remote": name, "url": url, "exported": out, "git_commit": commit}
	addPublicationAuditData(auditData, publication)
	if err := s.AppendAudit("remote_synced", BootstrapUser, auditData); err != nil {
		return nil, err
	}
	return &RemoteSyncResult{Name: name, Spec: remote["spec"], Mode: remote["mode"], URL: url, GitCommit: commit, Export: export, Exported: out}, nil
}

func attachRemoteMainParent(dir, subject, body string) (string, error) {
	fetch := exec.Command("git", "fetch", "origin", "main")
	fetch.Dir = dir
	if out, err := fetch.CombinedOutput(); err != nil {
		if strings.Contains(string(out), "couldn't find remote ref main") {
			return gitHead(dir)
		}
		return "", fmt.Errorf("git fetch origin main failed: %w\n%s", err, string(out))
	}
	tree := exec.Command("git", "rev-parse", "HEAD^{tree}")
	tree.Dir = dir
	treeOut, err := tree.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD^{tree}: %w", err)
	}
	commit := exec.Command("git", "commit-tree", strings.TrimSpace(string(treeOut)), "-p", "origin/main", "-m", subject, "-m", body)
	commit.Dir = dir
	commitOut, err := commit.Output()
	if err != nil {
		return "", fmt.Errorf("git commit-tree: %w", err)
	}
	newCommit := strings.TrimSpace(string(commitOut))
	reset := exec.Command("git", "reset", "--hard", newCommit)
	reset.Dir = dir
	if out, err := reset.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git reset --hard export commit failed: %w\n%s", err, string(out))
	}
	return newCommit, nil
}

func remoteURL(spec string) string {
	if strings.HasPrefix(spec, "github:") {
		return "git@github.com:" + strings.TrimPrefix(spec, "github:") + ".git"
	}
	if strings.HasPrefix(spec, "http://") || strings.HasPrefix(spec, "https://") || strings.HasPrefix(spec, "git@") {
		return spec
	}
	return ""
}

func (s *Store) latestPublicationForRealm(realm string) (*exportPublication, error) {
	row := s.DB.QueryRow(`SELECT id, work_name, dest_realm, mode, created_at FROM publications WHERE dest_realm = ? AND status = 'published' ORDER BY created_at DESC, id DESC LIMIT 1`, realm)
	var pub exportPublication
	if err := row.Scan(&pub.ID, &pub.Work, &pub.Realm, &pub.Mode, &pub.Created); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &pub, nil
}

func gitExportCommitMessage(realm string, pub *exportPublication, exportedAt string) (string, string) {
	if pub == nil {
		return fmt.Sprintf("Export Glyph %s projection", realm), strings.Join([]string{
			"Glyph export without a matching publication record.",
			"",
			"Glyph-Realm: " + realm,
			"Glyph-Exported-At: " + exportedAt,
		}, "\n")
	}
	return fmt.Sprintf("Publish %s to %s", pub.Work, pub.Realm), strings.Join([]string{
		"Export Glyph public projection.",
		"",
		"Glyph-Publication: " + pub.ID,
		"Glyph-Work: " + pub.Work,
		"Glyph-Realm: " + pub.Realm,
		"Glyph-Mode: " + pub.Mode,
		"Glyph-Published-At: " + pub.Created,
		"Glyph-Exported-At: " + exportedAt,
	}, "\n")
}

func addPublicationAuditData(data map[string]any, pub *exportPublication) {
	if pub == nil {
		return
	}
	data["publication"] = pub.ID
	data["work"] = pub.Work
	data["dest_realm"] = pub.Realm
	data["mode"] = pub.Mode
	data["published_at"] = pub.Created
}

func gitHead(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
