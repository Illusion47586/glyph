package store

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type RemoteSyncResult struct {
	Name     string           `json:"name"`
	Spec     string           `json:"spec"`
	Mode     string           `json:"mode"`
	URL      string           `json:"url"`
	Export   *GitExportResult `json:"export"`
	Exported string           `json:"exported"`
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
	commands := [][]string{
		{"git", "init", "-b", "main"},
		{"git", "add", "."},
		{"git", "-c", "user.name=Glyph", "-c", "user.email=glyph@example.local", "commit", "-m", "Export Glyph public projection"},
	}
	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = out
		if outBytes, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("%s failed: %w\n%s", strings.Join(args, " "), err, string(outBytes))
		}
	}
	if err := s.AppendAudit("git_exported", BootstrapUser, map[string]any{"realm": realm, "out": out, "generated": result.Generated, "skipped": result.Skipped}); err != nil {
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
	url := remoteURL(remote["spec"])
	if url == "" {
		return nil, fmt.Errorf("unsupported remote spec %q", remote["spec"])
	}
	commands := [][]string{
		{"git", "remote", "add", "origin", url},
		{"git", "push", "-u", "origin", "main"},
	}
	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = out
		if outBytes, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("%s failed: %w\n%s", strings.Join(args, " "), err, string(outBytes))
		}
	}
	if err := s.AppendAudit("remote_synced", BootstrapUser, map[string]any{"remote": name, "url": url, "exported": out}); err != nil {
		return nil, err
	}
	return &RemoteSyncResult{Name: name, Spec: remote["spec"], Mode: remote["mode"], URL: url, Export: export, Exported: out}, nil
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
