package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (s *Store) ClaimWork(work, actorID, mode string, ttl time.Duration) (*WorkClaim, error) {
	if actorID == "" {
		return nil, errors.New("actor is required")
	}
	if mode == "" {
		mode = "exclusive"
	}
	if mode != "exclusive" && mode != "shared-read" && mode != "handoff" {
		return nil, fmt.Errorf("unsupported claim mode %q", mode)
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	if _, err := s.Work(work); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	expires := now.Add(ttl)
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var existingActor, existingExpires, existingStatus string
	err = tx.QueryRow(`SELECT actor, expires_at, status FROM work_claims WHERE work_name = ?`, work).Scan(&existingActor, &existingExpires, &existingStatus)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return nil, err
	default:
		exp, parseErr := time.Parse(time.RFC3339, existingExpires)
		if parseErr != nil {
			return nil, parseErr
		}
		active := existingStatus == "active" && now.Before(exp)
		if active && existingActor != actorID && mode != "handoff" {
			return nil, fmt.Errorf("work context %q is already claimed by %s", work, existingActor)
		}
	}

	provider, session := actorParts(actorID)
	claim := WorkClaim{
		WorkName:    work,
		Actor:       actorID,
		Provider:    provider,
		SessionID:   session,
		Mode:        mode,
		Status:      "active",
		ClaimedAt:   now.Format(time.RFC3339),
		HeartbeatAt: now.Format(time.RFC3339),
		ExpiresAt:   expires.Format(time.RFC3339),
	}
	_, err = tx.Exec(`INSERT OR REPLACE INTO work_claims(work_name, actor, provider, session_id, mode, status, claimed_at, heartbeat_at, expires_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		claim.WorkName, claim.Actor, claim.Provider, claim.SessionID, claim.Mode, claim.Status, claim.ClaimedAt, claim.HeartbeatAt, claim.ExpiresAt)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	_ = s.AppendAudit("work_claimed", actorID, map[string]any{"work": work, "mode": mode, "expires_at": claim.ExpiresAt})
	return &claim, nil
}

func actorParts(actorID string) (string, string) {
	parts := strings.Split(actorID, ":")
	if len(parts) >= 3 {
		return parts[1], strings.Join(parts[2:], ":")
	}
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "self", actorID
}

func (s *Store) HeartbeatWork(work, actorID string, ttl time.Duration) (*WorkClaim, error) {
	if actorID == "" {
		return nil, errors.New("actor is required")
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	now := time.Now().UTC()
	expires := now.Add(ttl)
	res, err := s.DB.Exec(`UPDATE work_claims SET heartbeat_at = ?, expires_at = ?, status = 'active' WHERE work_name = ? AND actor = ?`,
		now.Format(time.RFC3339), expires.Format(time.RFC3339), work, actorID)
	if err != nil {
		return nil, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, fmt.Errorf("work context %q is not claimed by %s", work, actorID)
	}
	claim, err := s.WorkClaim(work)
	if err != nil {
		return nil, err
	}
	_ = s.AppendAudit("work_heartbeat", actorID, map[string]any{"work": work, "expires_at": claim.ExpiresAt})
	return claim, nil
}

func (s *Store) ReleaseWork(work, actorID string) error {
	res, err := s.DB.Exec(`UPDATE work_claims SET status = 'released' WHERE work_name = ? AND actor = ? AND status = 'active'`, work, actorID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("work context %q is not actively claimed by %s", work, actorID)
	}
	return s.AppendAudit("work_released", actorID, map[string]any{"work": work})
}

func (s *Store) WorkClaim(work string) (*WorkClaim, error) {
	var claim WorkClaim
	if err := s.DB.QueryRow(`SELECT work_name, actor, provider, session_id, mode, status, claimed_at, heartbeat_at, expires_at FROM work_claims WHERE work_name = ?`, work).
		Scan(&claim.WorkName, &claim.Actor, &claim.Provider, &claim.SessionID, &claim.Mode, &claim.Status, &claim.ClaimedAt, &claim.HeartbeatAt, &claim.ExpiresAt); err != nil {
		return nil, err
	}
	exp, err := time.Parse(time.RFC3339, claim.ExpiresAt)
	if err != nil {
		return nil, err
	}
	claim.Stale = claim.Status == "active" && time.Now().UTC().After(exp)
	return &claim, nil
}

func (s *Store) AddDependency(work, dependsOn string) (*WorkDependency, error) {
	if work == dependsOn {
		return nil, errors.New("work context cannot depend on itself")
	}
	if _, err := s.Work(work); err != nil {
		return nil, err
	}
	if _, err := s.Work(dependsOn); err != nil {
		return nil, err
	}
	dep := WorkDependency{WorkName: work, DependsOnWork: dependsOn, CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	if _, err := s.DB.Exec(`INSERT OR REPLACE INTO work_dependencies(work_name, depends_on_work, created_at) VALUES(?, ?, ?)`, dep.WorkName, dep.DependsOnWork, dep.CreatedAt); err != nil {
		return nil, err
	}
	_ = s.AppendAudit("work_dependency_added", BootstrapUser, map[string]any{"work": work, "depends_on": dependsOn})
	return &dep, nil
}
