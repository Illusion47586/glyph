package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Store) Publish(work, destRealm string) (string, error) {
	return s.PublishWithOptions(PublishOptions{Work: work, DestRealm: destRealm, Mode: "squash"})
}

func (s *Store) PublishWithMode(work, destRealm, mode string) (string, error) {
	return s.PublishWithOptions(PublishOptions{Work: work, DestRealm: destRealm, Mode: mode})
}

type PublishOptions struct {
	Work                string
	DestRealm           string
	Mode                string
	SemanticType        string
	SemanticScope       string
	SemanticDescription string
}

func (o PublishOptions) normalized() (PublishOptions, error) {
	o.SemanticType = strings.TrimSpace(o.SemanticType)
	o.SemanticScope = strings.TrimSpace(o.SemanticScope)
	o.SemanticDescription = strings.TrimSpace(o.SemanticDescription)
	if o.Mode == "" {
		o.Mode = "squash"
	}
	if o.Mode != "squash" && o.Mode != "preserve" {
		return o, fmt.Errorf("unsupported publication mode %q", o.Mode)
	}
	if o.SemanticType == "" && o.SemanticScope == "" && o.SemanticDescription == "" {
		return o, nil
	}
	if o.SemanticType == "" {
		return o, fmt.Errorf("--semantic-type is required when semantic publication metadata is set")
	}
	if o.SemanticDescription == "" {
		return o, fmt.Errorf("--semantic-description is required when semantic publication metadata is set")
	}
	if strings.ContainsAny(o.SemanticType, " ():\n\t") {
		return o, fmt.Errorf("semantic type must not contain whitespace, parentheses, or colon")
	}
	if strings.ContainsAny(o.SemanticScope, " ():\n\r\t") {
		return o, fmt.Errorf("semantic scope must not contain whitespace, parentheses, or colon")
	}
	if strings.ContainsAny(o.SemanticDescription, "\n\r") {
		return o, fmt.Errorf("semantic description must be a single line")
	}
	return o, nil
}

func (s *Store) PublishWithOptions(opts PublishOptions) (string, error) {
	opts, err := opts.normalized()
	if err != nil {
		return "", err
	}
	work := opts.Work
	destRealm := opts.DestRealm
	mode := opts.Mode
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
	if _, err := s.DB.Exec(`INSERT INTO publications(id, work_name, dest_realm, status, actor, mode, created_at, semantic_type, semantic_scope, semantic_description) VALUES(?, ?, ?, 'published', ?, ?, ?, ?, ?, ?)`, id, work, destRealm, BootstrapUser, mode, now, opts.SemanticType, opts.SemanticScope, opts.SemanticDescription); err != nil {
		return "", err
	}
	if _, err := s.DB.Exec(`UPDATE work_contexts SET status = 'published' WHERE name = ?`, work); err != nil {
		return "", err
	}
	auditData := map[string]any{"publication": id, "work": work, "dest_realm": destRealm, "mode": mode}
	if opts.SemanticType != "" {
		auditData["semantic_type"] = opts.SemanticType
		auditData["semantic_scope"] = opts.SemanticScope
		auditData["semantic_description"] = opts.SemanticDescription
	}
	if err := s.AppendAudit("publication_published", BootstrapUser, auditData); err != nil {
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
	rows, err := s.DB.Query(`SELECT id, work_name, dest_realm, status, actor, mode, created_at, semantic_type, semantic_scope, semantic_description FROM publications ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pubs []map[string]string
	for rows.Next() {
		var id, work, realm, status, actor, mode, created, semanticType, semanticScope, semanticDescription string
		if err := rows.Scan(&id, &work, &realm, &status, &actor, &mode, &created, &semanticType, &semanticScope, &semanticDescription); err != nil {
			return nil, err
		}
		pub := map[string]string{"id": id, "work": work, "realm": realm, "status": status, "actor": actor, "mode": mode, "created": created}
		if semanticType != "" {
			pub["semantic_type"] = semanticType
			pub["semantic_scope"] = semanticScope
			pub["semantic_description"] = semanticDescription
		}
		pubs = append(pubs, pub)
	}
	return pubs, rows.Err()
}
