package store

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func (s *Store) StartWork(name, realm string) (*WorkContext, error) {
	if name == "" {
		return nil, errors.New("work name is required")
	}
	path := filepath.Join(s.Dir, "workspaces", name)
	if _, err := os.Stat(path); err == nil {
		if _, workErr := s.Work(name); workErr == nil {
			return nil, fmt.Errorf("work context %q already exists", name)
		}
		if err := os.RemoveAll(path); err != nil {
			return nil, fmt.Errorf("removing stale workspace %q: %w", path, err)
		}
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return nil, err
	}
	if err := s.MaterializeRealm(realm, path); err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := s.DB.Exec(`INSERT INTO work_contexts(name, base_realm, workspace_path, status, created_at) VALUES(?, ?, ?, 'active', ?)`, name, realm, path, now); err != nil {
		return nil, err
	}
	if err := s.Snapshot(name, "work started"); err != nil {
		return nil, err
	}
	_ = s.AppendAudit("work_started", BootstrapUser, map[string]any{"work": name, "realm": realm})
	return &WorkContext{Name: name, BaseRealm: realm, WorkspacePath: path, Status: "active"}, nil
}

func (s *Store) Work(name string) (*WorkContext, error) {
	var wc WorkContext
	if err := s.DB.QueryRow(`SELECT name, base_realm, workspace_path, status FROM work_contexts WHERE name = ?`, name).Scan(&wc.Name, &wc.BaseRealm, &wc.WorkspacePath, &wc.Status); err != nil {
		return nil, err
	}
	return &wc, nil
}

func (s *Store) ListWork() ([]WorkContext, error) {
	rows, err := s.DB.Query(`SELECT name, base_realm, workspace_path, status FROM work_contexts ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WorkContext
	for rows.Next() {
		var wc WorkContext
		if err := rows.Scan(&wc.Name, &wc.BaseRealm, &wc.WorkspacePath, &wc.Status); err != nil {
			return nil, err
		}
		out = append(out, wc)
	}
	return out, rows.Err()
}

func (s *Store) Snapshot(work, reason string) error {
	wc, err := s.Work(work)
	if err != nil {
		return err
	}
	hash, err := treeHash(wc.WorkspacePath)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	id := "snapshot:" + nowNano() + ":" + hash[:12]
	if _, err := s.DB.Exec(`INSERT INTO snapshots(id, work_name, reason, hash, created_at) VALUES(?, ?, ?, ?, ?)`, id, work, reason, hash, now); err != nil {
		return err
	}
	return s.AppendAudit("snapshot_created", BootstrapUser, map[string]any{"snapshot": id, "work": work, "reason": reason, "hash": hash})
}

func treeHash(root string) (string, error) {
	var entries []string
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		entries = append(entries, filepath.ToSlash(rel)+":"+hex.EncodeToString(sum[:]))
		return nil
	}); err != nil {
		return "", err
	}
	sort.Strings(entries)
	sum := sha256.Sum256([]byte(strings.Join(entries, "\n")))
	return hex.EncodeToString(sum[:]), nil
}

func nowNano() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func (s *Store) ReadWorkFile(work, rel string) ([]byte, error) {
	wc, err := s.Work(work)
	if err != nil {
		return nil, err
	}
	path, err := safeJoin(wc.WorkspacePath, rel)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

func (s *Store) WriteWorkFile(work, rel string, data []byte, reason string) error {
	wc, err := s.Work(work)
	if err != nil {
		return err
	}
	if err := s.Snapshot(work, "before write: "+reason); err != nil {
		return err
	}
	path, err := safeJoin(wc.WorkspacePath, rel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	if err := s.Snapshot(work, "after write: "+reason); err != nil {
		return err
	}
	return s.AppendAudit("work_file_written", BootstrapUser, map[string]any{"work": work, "path": rel, "reason": reason})
}

func safeJoin(root, rel string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(rel))
	if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return "", fmt.Errorf("invalid path %q", rel)
	}
	return filepath.Join(root, clean), nil
}

func (s *Store) ProjectWork(work, dest string) error {
	wc, err := s.Work(work)
	if err != nil {
		return err
	}
	return copyDir(wc.WorkspacePath, dest)
}

func copyDir(src, dest string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		to := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(to, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(to, data, 0o644)
	})
}

func (s *Store) DiffWork(work string) ([]string, error) {
	changes, err := s.workChangeSet(work)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(changes))
	for path, change := range changes {
		out = append(out, change.Status+" "+path)
	}
	sort.Strings(out)
	return out, nil
}

type workChange struct {
	Path   string
	Status string
	Hash   string
}

func (s *Store) workChangeSet(work string) (map[string]workChange, error) {
	wc, err := s.Work(work)
	if err != nil {
		return nil, err
	}
	base, err := s.SourcesForRealm(wc.BaseRealm)
	if err != nil {
		return nil, err
	}
	baseHashes := map[string]string{}
	for _, src := range base {
		baseHashes[src.Path] = src.Hash
	}
	seen := map[string]bool{}
	changes := map[string]workChange{}
	err = filepath.WalkDir(wc.WorkspacePath, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(wc.WorkspacePath, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		seen[rel] = true
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		hash := hex.EncodeToString(sum[:])
		switch baseHash, ok := baseHashes[rel]; {
		case !ok:
			changes[rel] = workChange{Path: rel, Status: "A", Hash: hash}
		case baseHash != hash:
			changes[rel] = workChange{Path: rel, Status: "M", Hash: hash}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for path := range baseHashes {
		if !seen[path] {
			changes[path] = workChange{Path: path, Status: "D", Hash: ""}
		}
	}
	return changes, nil
}
