package store

func (s *Store) addHookGraph(g *Graph) error {
	rows, err := s.DB.Query(`SELECT id, event, path, work_name, dest_realm, mode, exit_code, started_at, duration_ms, blocked FROM hook_runs ORDER BY started_at`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, event, path, work, realm, mode, started string
		var exitCode, durationMS, blocked int
		if err := rows.Scan(&id, &event, &path, &work, &realm, &mode, &exitCode, &started, &durationMS, &blocked); err != nil {
			return err
		}
		g.addNode(id, "hook_run", event, map[string]string{
			"path":        path,
			"work":        work,
			"dest_realm":  realm,
			"mode":        mode,
			"exit_code":   intString(exitCode),
			"started_at":  started,
			"duration_ms": intString(durationMS),
			"blocked":     boolString(blocked == 1),
		})
		if work != "" {
			g.addEdge("work:"+work, id, "ran_hook", nil)
		}
	}
	return rows.Err()
}

func (s *Store) addRemoteMountGraph(g *Graph) error {
	if err := s.addRemoteGraph(g); err != nil {
		return err
	}
	return s.addMountGraph(g)
}

func (s *Store) addRemoteGraph(g *Graph) error {
	rows, err := s.DB.Query(`SELECT name, spec, mode, created_at FROM remotes ORDER BY name`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var name, spec, mode, created string
		if err := rows.Scan(&name, &spec, &mode, &created); err != nil {
			return err
		}
		id := "remote:" + name
		g.addNode(id, "remote", name, map[string]string{"spec": spec, "mode": mode, "created_at": created})
		g.addEdge("store:"+s.Root, id, "syncs_to", nil)
	}
	return rows.Err()
}

func (s *Store) addMountGraph(g *Graph) error {
	rows, err := s.DB.Query(`SELECT path, spec, mode, pinned_revision, created_at FROM mounts ORDER BY path`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var path, spec, mode, pinned, created string
		if err := rows.Scan(&path, &spec, &mode, &pinned, &created); err != nil {
			return err
		}
		id := "mount:" + path
		g.addNode(id, "mount", path, map[string]string{"spec": spec, "mode": mode, "pinned_revision": pinned, "created_at": created})
		g.addEdge("store:"+s.Root, id, "mounted_at", nil)
	}
	return rows.Err()
}

func intString(v int) string {
	return int64String(int64(v))
}

func boolString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
