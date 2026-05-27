package store

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (s *Store) addTimeline(g *Graph) error {
	seen := map[string]bool{}
	if err := s.addAuditEvents(g, seen); err != nil {
		return err
	}
	if err := s.addSnapshotEvents(g, seen); err != nil {
		return err
	}
	if err := s.addPublicationEvents(g, seen); err != nil {
		return err
	}
	return s.addHookEvents(g, seen)
}

func (s *Store) addAuditEvents(g *Graph, seen map[string]bool) error {
	path := filepath.Join(s.Dir, "audit", "events.jsonl")
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 8*1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		var ev AuditEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			return fmt.Errorf("decode audit event line %d: %w", line, err)
		}
		id := fmt.Sprintf("audit:%06d:%s", line, ev.Type)
		details, nodes := eventDetails(ev.Data)
		g.addEvent(id, ev.Type, eventLabel(ev.Type, details), ev.Timestamp, ev.Actor, details, nodes)
		seen[eventSeenKey(ev.Type, ev.Timestamp, details)] = true
	}
	return scanner.Err()
}

func (s *Store) addSnapshotEvents(g *Graph, seen map[string]bool) error {
	rows, err := s.DB.Query(`SELECT id, work_name, reason, hash, created_at FROM snapshots ORDER BY created_at`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, work, reason, hash, created string
		if err := rows.Scan(&id, &work, &reason, &hash, &created); err != nil {
			return err
		}
		details := map[string]string{"snapshot": id, "work": work, "reason": reason, "hash": hash}
		if seen[eventSeenKey("snapshot_created", created, details)] {
			continue
		}
		g.addEvent("derived:"+id, "snapshot_created", "snapshot "+shortHash(hash), created, "", details, []string{"work:" + work, id})
	}
	return rows.Err()
}

func (s *Store) addPublicationEvents(g *Graph, seen map[string]bool) error {
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
		details := map[string]string{"publication": id, "work": work, "dest_realm": realm, "status": status, "mode": mode}
		if seen[eventSeenKey("publication_published", created, details)] {
			continue
		}
		nodes := []string{"work:" + work, id, "realm:" + realm}
		g.addEvent("derived:"+id, "publication_published", "published "+work, created, actor, details, nodes)
	}
	return rows.Err()
}

func (s *Store) addHookEvents(g *Graph, seen map[string]bool) error {
	rows, err := s.DB.Query(`SELECT id, event, work_name, dest_realm, mode, exit_code, started_at, blocked FROM hook_runs ORDER BY started_at`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, event, work, realm, mode, started string
		var exitCode, blocked int
		if err := rows.Scan(&id, &event, &work, &realm, &mode, &exitCode, &started, &blocked); err != nil {
			return err
		}
		details := map[string]string{"event": event, "work": work, "dest_realm": realm, "mode": mode, "exit_code": intString(exitCode), "blocked": boolString(blocked == 1)}
		if seen[eventSeenKey("hook_run", started, details)] {
			continue
		}
		g.addEvent("derived:"+id, "hook_run", "hook "+event, started, "", details, []string{"work:" + work, id})
	}
	return rows.Err()
}

func eventDetails(data map[string]any) (map[string]string, []string) {
	details := map[string]string{}
	var nodes []string
	for _, key := range sortedAnyKeys(data) {
		value := data[key]
		switch key {
		case "included":
			paths := includedPaths(value)
			details["included_count"] = intString(len(paths))
			for _, path := range paths {
				nodes = append(nodes, "source:"+path)
			}
		case "excluded":
			details["excluded_count"] = intString(len(anySlice(value)))
		default:
			details[key] = valueString(value)
		}
	}
	nodes = append(nodes, relatedNodes(details)...)
	return details, nodes
}

func relatedNodes(details map[string]string) []string {
	var nodes []string
	for _, key := range []string{"work", "depends_on", "other_work"} {
		if v := details[key]; v != "" {
			nodes = append(nodes, "work:"+v)
		}
	}
	for _, key := range []string{"realm", "dest_realm"} {
		if v := details[key]; v != "" {
			nodes = append(nodes, "realm:"+v)
		}
	}
	for _, key := range []string{"snapshot", "publication"} {
		if v := details[key]; v != "" {
			nodes = append(nodes, v)
		}
	}
	if v := details["remote"]; v != "" {
		nodes = append(nodes, "remote:"+v)
	}
	if v := details["name"]; details["spec"] != "" && v != "" {
		nodes = append(nodes, "remote:"+v)
	}
	if v := details["path"]; v != "" {
		nodes = append(nodes, "source:"+v)
	}
	return nodes
}

func includedPaths(value any) []string {
	var paths []string
	for _, item := range anySlice(value) {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if path, ok := m["path"].(string); ok && path != "" {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths
}

func anySlice(value any) []any {
	if items, ok := value.([]any); ok {
		return items
	}
	return nil
}

func sortedAnyKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func eventLabel(typ string, details map[string]string) string {
	if work := details["work"]; work != "" {
		return strings.ReplaceAll(typ, "_", " ") + ": " + work
	}
	if path := details["path"]; path != "" {
		return strings.ReplaceAll(typ, "_", " ") + ": " + path
	}
	return strings.ReplaceAll(typ, "_", " ")
}

func eventSeenKey(typ, timestamp string, details map[string]string) string {
	return typ + "\x00" + timestamp + "\x00" + details["work"] + "\x00" + details["snapshot"] + "\x00" + details["publication"]
}

func uniqueEventStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func valueString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case float64, bool:
		return fmt.Sprintf("%v", v)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	}
}
