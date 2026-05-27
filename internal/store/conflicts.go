package store

import (
	"fmt"
	"os"
	"sort"
	"time"
)

func (s *Store) WorkConflicts(work string) ([]WorkConflict, error) {
	if _, err := s.Work(work); err != nil {
		return nil, err
	}
	target, err := s.workChangeSet(work)
	if err != nil {
		return nil, err
	}
	if _, err := s.DB.Exec(`DELETE FROM work_conflicts WHERE work_name = ?`, work); err != nil {
		return nil, err
	}
	workContexts, err := s.ListWork()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	conflicts := make([]WorkConflict, 0)
	for _, other := range workContexts {
		if other.Name == work || other.Status == "pruned" || other.Status == "discarded" {
			continue
		}
		otherChanges, err := s.workChangeSet(other.Name)
		if err != nil {
			return nil, err
		}
		for path, targetChange := range target {
			otherChange, ok := otherChanges[path]
			if !ok {
				continue
			}
			if targetChange.Status == otherChange.Status && targetChange.Hash == otherChange.Hash {
				continue
			}
			conflict := WorkConflict{
				WorkName:  work,
				OtherWork: other.Name,
				Path:      path,
				Type:      conflictType(targetChange, otherChange),
				Detail:    fmt.Sprintf("%s:%s vs %s:%s", targetChange.Status, shortHash(targetChange.Hash), otherChange.Status, shortHash(otherChange.Hash)),
				CreatedAt: now,
			}
			conflicts = append(conflicts, conflict)
			if _, err := s.DB.Exec(`INSERT OR REPLACE INTO work_conflicts(work_name, other_work, path, type, detail, created_at) VALUES(?, ?, ?, ?, ?, ?)`,
				conflict.WorkName, conflict.OtherWork, conflict.Path, conflict.Type, conflict.Detail, conflict.CreatedAt); err != nil {
				return nil, err
			}
		}
	}
	sort.Slice(conflicts, func(i, j int) bool {
		if conflicts[i].Path == conflicts[j].Path {
			return conflicts[i].OtherWork < conflicts[j].OtherWork
		}
		return conflicts[i].Path < conflicts[j].Path
	})
	return conflicts, nil
}

func conflictType(a, b workChange) string {
	if a.Status == "D" || b.Status == "D" {
		return "path"
	}
	return "content"
}

func shortHash(hash string) string {
	if hash == "" {
		return "none"
	}
	if len(hash) < 12 {
		return hash
	}
	return hash[:12]
}

func (s *Store) PruneWork(work string) error {
	wc, err := s.Work(work)
	if err != nil {
		return err
	}
	if wc.Status == "active" || wc.Status == "ready" || wc.Status == "blocked" || wc.Status == "conflicted" {
		return fmt.Errorf("work context %q is %s and cannot be pruned", work, wc.Status)
	}
	if err := os.RemoveAll(wc.WorkspacePath); err != nil {
		return err
	}
	if _, err := s.DB.Exec(`UPDATE work_contexts SET status = 'pruned' WHERE name = ?`, work); err != nil {
		return err
	}
	_, _ = s.DB.Exec(`UPDATE work_claims SET status = 'released' WHERE work_name = ? AND status = 'active'`, work)
	return s.AppendAudit("work_pruned", BootstrapUser, map[string]any{"work": work, "workspace": wc.WorkspacePath})
}
