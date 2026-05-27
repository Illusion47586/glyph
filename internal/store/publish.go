package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Store) Publish(work, destRealm string) (string, error) {
	return s.PublishWithMode(work, destRealm, "squash")
}

func (s *Store) PublishWithMode(work, destRealm, mode string) (string, error) {
	if mode == "" {
		mode = "squash"
	}
	if mode != "squash" && mode != "preserve" {
		return "", fmt.Errorf("unsupported publication mode %q", mode)
	}
	wc, err := s.Work(work)
	if err != nil {
		return "", err
	}
	conflicts, err := s.WorkConflicts(work)
	if err != nil {
		return "", err
	}
	if len(conflicts) > 0 {
		if _, updateErr := s.DB.Exec(`UPDATE work_contexts SET status = 'conflicted' WHERE name = ?`, work); updateErr != nil {
			return "", updateErr
		}
		return "", fmt.Errorf("publication blocked by %d unresolved conflict(s)", len(conflicts))
	}
	if _, err := s.RunHook(HookContext{Event: "pre-publish", Work: work, DestRealm: destRealm, Mode: mode, Actor: BootstrapUser}); err != nil {
		return "", err
	}
	if err := s.Snapshot(work, "before publication"); err != nil {
		return "", err
	}
	if err := filepath.WalkDir(wc.WorkspacePath, func(path string, d os.DirEntry, walkErr error) error {
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
		if isDeniedPublic(rel) && destRealm == "public" {
			return fmt.Errorf("publication blocked by policy for %s", rel)
		}
		_, err = s.StoreFile(path, rel, destRealm)
		return err
	}); err != nil {
		return "", err
	}
	id := "publication:" + nowNano()
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := s.DB.Exec(`INSERT INTO publications(id, work_name, dest_realm, status, actor, mode, created_at) VALUES(?, ?, ?, 'published', ?, ?, ?)`, id, work, destRealm, BootstrapUser, mode, now); err != nil {
		return "", err
	}
	if _, err := s.DB.Exec(`UPDATE work_contexts SET status = 'published' WHERE name = ?`, work); err != nil {
		return "", err
	}
	if err := s.AppendAudit("publication_published", BootstrapUser, map[string]any{"publication": id, "work": work, "dest_realm": destRealm, "mode": mode}); err != nil {
		return "", err
	}
	_, _ = s.RunHook(HookContext{Event: "post-publish", Work: work, DestRealm: destRealm, Mode: mode, PublicationID: id, Actor: BootstrapUser})
	return id, nil
}

func isDeniedPublic(path string) bool {
	base := filepath.Base(path)
	return strings.HasPrefix(base, ".env")
}

func (s *Store) ListPublications() ([]map[string]string, error) {
	rows, err := s.DB.Query(`SELECT id, work_name, dest_realm, status, actor, mode, created_at FROM publications ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pubs []map[string]string
	for rows.Next() {
		var id, work, realm, status, actor, mode, created string
		if err := rows.Scan(&id, &work, &realm, &status, &actor, &mode, &created); err != nil {
			return nil, err
		}
		pubs = append(pubs, map[string]string{"id": id, "work": work, "realm": realm, "status": status, "actor": actor, "mode": mode, "created": created})
	}
	return pubs, rows.Err()
}
