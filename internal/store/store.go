package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
)

const (
	DirName       = ".glyph"
	StoreVersion  = "1"
	BootstrapUser = "user:self:dhruv"
	HookOutputMax = 32 * 1024
)

type Store struct {
	Root string
	Dir  string
	DB   *sql.DB
}

type BootstrapManifest struct {
	Project struct {
		Name   string `yaml:"name"`
		Status string `yaml:"status"`
	} `yaml:"project"`
	Specs struct {
		Directory string `yaml:"directory"`
	} `yaml:"specs"`
	Realms struct {
		DefaultPublic  string `yaml:"default_public"`
		DefaultPrivate string `yaml:"default_private"`
	} `yaml:"realms"`
	Identity struct {
		BootstrapUser string `yaml:"bootstrap_user"`
	} `yaml:"identity"`
	GenesisImport struct {
		Include     []string `yaml:"include"`
		Exclude     []string `yaml:"exclude"`
		Transcripts struct {
			Default string `yaml:"default"`
			Note    string `yaml:"note"`
		} `yaml:"transcripts"`
	} `yaml:"genesis_import"`
	Defaults struct {
		Export struct {
			Git struct {
				Gitignore  string `yaml:"gitignore"`
				Gitinclude string `yaml:"gitinclude"`
			} `yaml:"git"`
		} `yaml:"export"`
		Viz struct {
			Export struct {
				Out string `yaml:"out"`
			} `yaml:"export"`
		} `yaml:"viz"`
	} `yaml:"defaults"`
	Checks struct {
		PublicExport []ExportCheck `yaml:"public_export"`
	} `yaml:"checks"`
}

type ExportCheck struct {
	Name    string `json:"name" yaml:"name"`
	Command string `json:"command" yaml:"command"`
}

type AuditEvent struct {
	Type      string         `json:"type"`
	Timestamp string         `json:"timestamp"`
	Actor     string         `json:"actor"`
	Data      map[string]any `json:"data,omitempty"`
}

type Source struct {
	Path      string
	ContentID string
	Hash      string
	Labels    string
	Size      int64
}

type WorkContext struct {
	Name          string
	BaseRealm     string
	WorkspacePath string
	Status        string
}

type WorkClaim struct {
	WorkName    string `json:"work"`
	Actor       string `json:"actor"`
	Provider    string `json:"provider"`
	SessionID   string `json:"session_id"`
	Mode        string `json:"mode"`
	Status      string `json:"status"`
	ClaimedAt   string `json:"claimed_at"`
	HeartbeatAt string `json:"heartbeat_at"`
	ExpiresAt   string `json:"expires_at"`
	Stale       bool   `json:"stale"`
}

type WorkDependency struct {
	WorkName      string `json:"work"`
	DependsOnWork string `json:"depends_on"`
	CreatedAt     string `json:"created_at"`
}

type WorkConflict struct {
	WorkName  string `json:"work"`
	OtherWork string `json:"other_work"`
	Path      string `json:"path"`
	Type      string `json:"type"`
	Detail    string `json:"detail"`
	CreatedAt string `json:"created_at"`
}

type Hook struct {
	Event      string `json:"event"`
	Path       string `json:"path"`
	Executable bool   `json:"executable"`
}

type HookContext struct {
	Event         string
	Work          string
	DestRealm     string
	Mode          string
	PublicationID string
	Actor         string
}

type HookRun struct {
	ID        string `json:"id"`
	Event     string `json:"event"`
	Path      string `json:"path"`
	Work      string `json:"work,omitempty"`
	DestRealm string `json:"dest_realm,omitempty"`
	Mode      string `json:"mode,omitempty"`
	ExitCode  int    `json:"exit_code"`
	Stdout    string `json:"stdout,omitempty"`
	Stderr    string `json:"stderr,omitempty"`
	StartedAt string `json:"started_at"`
	Duration  string `json:"duration"`
	Blocked   bool   `json:"blocked"`
}

func Init(root string) (*Store, error) {
	dir := filepath.Join(root, DirName)
	for _, sub := range []string{"content", "audit", "workspaces", "exports", "hooks"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			return nil, err
		}
	}
	if err := writeStoreManifest(root); err != nil {
		return nil, err
	}
	st, err := Open(root)
	if err != nil {
		return nil, err
	}
	if err := st.initSchema(); err != nil {
		st.Close()
		return nil, err
	}
	if _, err := st.DB.Exec(`INSERT OR IGNORE INTO realms(name, description) VALUES
		('public', 'Public projection'),
		('maintainers', 'Maintainer-private projection')`); err != nil {
		st.Close()
		return nil, err
	}
	_ = st.AppendAudit("store_init", BootstrapUser, map[string]any{"store_version": StoreVersion})
	return st, nil
}

func Open(start string) (*Store, error) {
	root, err := FindRoot(start)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(root, DirName, "store.db"))
	if err != nil {
		return nil, err
	}
	st := &Store{Root: root, Dir: filepath.Join(root, DirName), DB: db}
	if _, err := st.DB.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := st.DB.Exec(`PRAGMA journal_mode = WAL`); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := st.initSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return st, nil
}

func FindRoot(start string) (string, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err == nil && !info.IsDir() {
		abs = filepath.Dir(abs)
	}
	for {
		if _, err := os.Stat(filepath.Join(abs, DirName)); err == nil {
			return abs, nil
		}
		if _, err := os.Stat(filepath.Join(abs, "glyph.yaml")); err == nil {
			return abs, nil
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return "", errors.New("not inside a Glyph workspace")
		}
		abs = parent
	}
}

func (s *Store) Close() error {
	if s.DB == nil {
		return nil
	}
	return s.DB.Close()
}

func (s *Store) initSchema() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS meta(key TEXT PRIMARY KEY, value TEXT NOT NULL)`,
		`INSERT OR REPLACE INTO meta(key, value) VALUES('store_version', '1')`,
		`CREATE TABLE IF NOT EXISTS content(id TEXT PRIMARY KEY, hash TEXT NOT NULL, size INTEGER NOT NULL, path TEXT NOT NULL, created_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS sources(path TEXT PRIMARY KEY, content_id TEXT NOT NULL, labels TEXT NOT NULL, updated_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS realms(name TEXT PRIMARY KEY, description TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS work_contexts(name TEXT PRIMARY KEY, base_realm TEXT NOT NULL, workspace_path TEXT NOT NULL, status TEXT NOT NULL, created_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS snapshots(id TEXT PRIMARY KEY, work_name TEXT NOT NULL, reason TEXT NOT NULL, hash TEXT NOT NULL, created_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS publications(id TEXT PRIMARY KEY, work_name TEXT NOT NULL, dest_realm TEXT NOT NULL, status TEXT NOT NULL, actor TEXT NOT NULL, mode TEXT NOT NULL DEFAULT 'squash', created_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS remotes(name TEXT PRIMARY KEY, spec TEXT NOT NULL, mode TEXT NOT NULL, created_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS mounts(path TEXT PRIMARY KEY, spec TEXT NOT NULL, mode TEXT NOT NULL, pinned_revision TEXT NOT NULL, created_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS work_claims(work_name TEXT PRIMARY KEY, actor TEXT NOT NULL, provider TEXT NOT NULL, session_id TEXT NOT NULL, mode TEXT NOT NULL, status TEXT NOT NULL, claimed_at TEXT NOT NULL, heartbeat_at TEXT NOT NULL, expires_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS work_dependencies(work_name TEXT NOT NULL, depends_on_work TEXT NOT NULL, created_at TEXT NOT NULL, PRIMARY KEY(work_name, depends_on_work))`,
		`CREATE TABLE IF NOT EXISTS work_conflicts(work_name TEXT NOT NULL, other_work TEXT NOT NULL, path TEXT NOT NULL, type TEXT NOT NULL, detail TEXT NOT NULL, created_at TEXT NOT NULL, PRIMARY KEY(work_name, other_work, path, type))`,
		`CREATE TABLE IF NOT EXISTS hook_runs(id TEXT PRIMARY KEY, event TEXT NOT NULL, path TEXT NOT NULL, work_name TEXT NOT NULL, dest_realm TEXT NOT NULL, mode TEXT NOT NULL, exit_code INTEGER NOT NULL, stdout TEXT NOT NULL, stderr TEXT NOT NULL, started_at TEXT NOT NULL, duration_ms INTEGER NOT NULL, blocked INTEGER NOT NULL)`,
	}
	for _, stmt := range stmts {
		if _, err := s.DB.Exec(stmt); err != nil {
			return err
		}
	}
	if err := s.ensureColumn("publications", "mode", "TEXT NOT NULL DEFAULT 'squash'"); err != nil {
		return err
	}
	if err := s.ensureColumn("publications", "semantic_type", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.ensureColumn("publications", "semantic_scope", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.ensureColumn("publications", "semantic_description", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	return nil
}

func (s *Store) ensureColumn(table, column, definition string) error {
	rows, err := s.DB.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = s.DB.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + column + ` ` + definition)
	return err
}

func writeStoreManifest(root string) error {
	data := []byte("store:\n  version: 1\n  created_by: glyph/0.1.0\n  project: glyph\n  object_format: glyph-object-v1\n  audit_log: audit/events.jsonl\n")
	return os.WriteFile(filepath.Join(root, DirName, "manifest.yaml"), data, 0o644)
}

func LoadBootstrapManifest(root string) (*BootstrapManifest, error) {
	data, err := os.ReadFile(filepath.Join(root, "glyph.yaml"))
	if err != nil {
		return nil, err
	}
	var manifest BootstrapManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func (s *Store) AppendAudit(kind, actor string, data map[string]any) error {
	ev := AuditEvent{
		Type:      kind,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Actor:     actor,
		Data:      data,
	}
	b, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	path := filepath.Join(s.Dir, "audit", "events.jsonl")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(b, '\n')); err != nil {
		return err
	}
	return nil
}
