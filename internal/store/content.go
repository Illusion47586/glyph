package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"
)

func (s *Store) StoreFile(absPath, relPath, labels string) (*Source, error) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	hashBytes := sha256.Sum256(data)
	hash := hex.EncodeToString(hashBytes[:])
	id := "content:sha256:" + hash
	blob := filepath.Join(s.Dir, "content", hash[:2], hash)
	if err := os.MkdirAll(filepath.Dir(blob), 0o755); err != nil {
		return nil, err
	}
	if _, err := os.Stat(blob); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(blob, data, 0o644); err != nil {
			return nil, err
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := s.DB.Exec(`INSERT OR IGNORE INTO content(id, hash, size, path, created_at) VALUES(?, ?, ?, ?, ?)`, id, hash, len(data), relPath, now); err != nil {
		return nil, err
	}
	if _, err := s.DB.Exec(`INSERT OR REPLACE INTO sources(path, content_id, labels, updated_at) VALUES(?, ?, ?, ?)`, relPath, id, labels, now); err != nil {
		return nil, err
	}
	return &Source{Path: relPath, ContentID: id, Hash: hash, Labels: labels, Size: int64(len(data))}, nil
}

func (s *Store) SourcesForRealm(realm string) ([]Source, error) {
	query := `SELECT s.path, s.content_id, c.hash, s.labels, c.size FROM sources s JOIN content c ON c.id = s.content_id`
	var rows *sql.Rows
	var err error
	if realm == "maintainers" {
		rows, err = s.DB.Query(query + ` ORDER BY s.path`)
	} else {
		rows, err = s.DB.Query(query+` WHERE s.labels LIKE ? ORDER BY s.path`, "%public%")
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sources []Source
	for rows.Next() {
		var src Source
		if err := rows.Scan(&src.Path, &src.ContentID, &src.Hash, &src.Labels, &src.Size); err != nil {
			return nil, err
		}
		sources = append(sources, src)
	}
	return sources, rows.Err()
}

func (s *Store) MaterializeRealm(realm, dest string) error {
	sources, err := s.SourcesForRealm(realm)
	if err != nil {
		return err
	}
	for _, src := range sources {
		if err := s.writeSource(src, dest); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) writeSource(src Source, dest string) error {
	from := filepath.Join(s.Dir, "content", src.Hash[:2], src.Hash)
	to := filepath.Join(dest, filepath.FromSlash(src.Path))
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(to)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func (s *Store) Counts() (map[string]int, error) {
	tables := []string{"sources", "content", "realms", "work_contexts", "snapshots", "publications", "remotes", "mounts", "work_claims", "work_dependencies", "work_conflicts", "hook_runs"}
	counts := map[string]int{}
	for _, table := range tables {
		var n int
		if err := s.DB.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
			return nil, err
		}
		counts[table] = n
	}
	return counts, nil
}
