package store

import (
	"database/sql"
	"fmt"
)

func (s *Store) addWorkGraph(g *Graph) error {
	rows, err := s.DB.Query(`SELECT name, base_realm, workspace_path, status, created_at FROM work_contexts ORDER BY name`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var name, base, workspace, status, created string
		if err := rows.Scan(&name, &base, &workspace, &status, &created); err != nil {
			return err
		}
		id := "work:" + name
		g.addNode(id, "work", name, map[string]string{"base_realm": base, "workspace": workspace, "status": status, "created_at": created})
		g.addEdge(id, "realm:"+base, "based_on", nil)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	snapshots, err := s.DB.Query(`SELECT id, work_name, reason, hash, created_at FROM snapshots ORDER BY created_at`)
	if err != nil {
		return err
	}
	defer snapshots.Close()
	for snapshots.Next() {
		var id, work, reason, hash, created string
		if err := snapshots.Scan(&id, &work, &reason, &hash, &created); err != nil {
			return err
		}
		g.addNode(id, "snapshot", shortHash(hash), map[string]string{"work": work, "reason": reason, "hash": hash, "created_at": created})
		g.addEdge("work:"+work, id, "captured", nil)
	}
	return snapshots.Err()
}

func (s *Store) addPublicationGraph(g *Graph) error {
	rows, err := s.DB.Query(`SELECT id, work_name, dest_realm, status, actor, mode, created_at FROM publications ORDER BY created_at`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, work, realm, status, actor, mode, created string
		if err := rows.Scan(&id, &work, &realm, &status, &actor, &mode, &created); err != nil {
			return err
		}
		g.addNode(id, "publication", id, map[string]string{"work": work, "realm": realm, "status": status, "actor": actor, "mode": mode, "created_at": created})
		g.addEdge("work:"+work, id, "published", map[string]string{"mode": mode})
		g.addEdge(id, "realm:"+realm, "published_to", nil)
	}
	return rows.Err()
}

func (s *Store) addConcurrencyGraph(g *Graph) error {
	if err := s.addClaimsGraph(g); err != nil {
		return err
	}
	if err := s.addDependenciesGraph(g); err != nil {
		return err
	}
	return s.addConflictsGraph(g)
}

func (s *Store) addClaimsGraph(g *Graph) error {
	rows, err := s.DB.Query(`SELECT work_name, actor, provider, session_id, mode, status, claimed_at, heartbeat_at, expires_at FROM work_claims ORDER BY work_name`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var work, actor, provider, session, mode, status, claimed, heartbeat, expires string
		if err := rows.Scan(&work, &actor, &provider, &session, &mode, &status, &claimed, &heartbeat, &expires); err != nil {
			return err
		}
		id := "claim:" + work + ":" + actor
		g.addNode(id, "claim", actor, map[string]string{"work": work, "provider": provider, "session_id": session, "mode": mode, "status": status, "claimed_at": claimed, "heartbeat_at": heartbeat, "expires_at": expires})
		g.addEdge("work:"+work, id, "claimed_by", nil)
	}
	return rows.Err()
}

func (s *Store) addDependenciesGraph(g *Graph) error {
	rows, err := s.DB.Query(`SELECT work_name, depends_on_work, created_at FROM work_dependencies ORDER BY work_name, depends_on_work`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var work, dependsOn, created string
		if err := rows.Scan(&work, &dependsOn, &created); err != nil {
			return err
		}
		g.addEdge("work:"+work, "work:"+dependsOn, "depends_on", map[string]string{"created_at": created})
	}
	return rows.Err()
}

func (s *Store) addConflictsGraph(g *Graph) error {
	rows, err := s.DB.Query(`SELECT work_name, other_work, path, type, detail, created_at FROM work_conflicts ORDER BY work_name, other_work, path`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var work, other, path, typ, detail, created string
		if err := rows.Scan(&work, &other, &path, &typ, &detail, &created); err != nil {
			return err
		}
		id := "conflict:" + work + ":" + other + ":" + path
		g.addNode(id, "conflict", path, map[string]string{"work": work, "other_work": other, "type": typ, "detail": detail, "created_at": created})
		g.addEdge("work:"+work, id, "conflicts_with", nil)
		g.addEdge(id, "work:"+other, "conflicts_with", nil)
	}
	return rows.Err()
}

func int64String(v int64) string {
	return fmt.Sprintf("%d", v)
}

func nullableString(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	return v.String
}
