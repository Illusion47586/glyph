package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func (s *Store) ListHooks() ([]Hook, error) {
	dir := filepath.Join(s.Dir, "hooks")
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return []Hook{}, nil
	}
	if err != nil {
		return nil, err
	}
	hooks := make([]Hook, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		hooks = append(hooks, Hook{
			Event:      entry.Name(),
			Path:       filepath.Join(dir, entry.Name()),
			Executable: info.Mode()&0o111 != 0,
		})
	}
	sort.Slice(hooks, func(i, j int) bool { return hooks[i].Event < hooks[j].Event })
	return hooks, nil
}

func (s *Store) RunHook(ctx HookContext) (*HookRun, error) {
	if ctx.Event == "" {
		return nil, errors.New("hook event is required")
	}
	if ctx.Actor == "" {
		ctx.Actor = BootstrapUser
	}
	path := filepath.Join(s.Dir, "hooks", ctx.Event)
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("hook %s is a directory", path)
	}
	if info.Mode()&0o111 == 0 {
		return nil, fmt.Errorf("hook %s is not executable", path)
	}

	start := time.Now().UTC()
	runCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, path)
	cmd.Dir = s.Root
	cmd.Env = append(os.Environ(),
		"GLYPH_EVENT="+ctx.Event,
		"GLYPH_ROOT="+s.Root,
		"GLYPH_STORE="+s.Dir,
		"GLYPH_WORK="+ctx.Work,
		"GLYPH_DEST_REALM="+ctx.DestRealm,
		"GLYPH_PUBLICATION_MODE="+ctx.Mode,
		"GLYPH_PUBLICATION_ID="+ctx.PublicationID,
	)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	runErr := cmd.Run()
	duration := time.Since(start)
	exitCode := 0
	if runErr != nil {
		exitCode = 1
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			exitCode = 124
		}
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
	}
	run := &HookRun{
		ID:        "hookrun:" + nowNano(),
		Event:     ctx.Event,
		Path:      path,
		Work:      ctx.Work,
		DestRealm: ctx.DestRealm,
		Mode:      ctx.Mode,
		ExitCode:  exitCode,
		Stdout:    truncateHookOutput(stdoutBuf.String()),
		Stderr:    truncateHookOutput(stderrBuf.String()),
		StartedAt: start.Format(time.RFC3339),
		Duration:  duration.String(),
		Blocked:   strings.HasPrefix(ctx.Event, "pre-") && exitCode != 0,
	}
	durationMS := duration.Milliseconds()
	if _, err := s.DB.Exec(`INSERT INTO hook_runs(id, event, path, work_name, dest_realm, mode, exit_code, stdout, stderr, started_at, duration_ms, blocked) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID, run.Event, run.Path, run.Work, run.DestRealm, run.Mode, run.ExitCode, run.Stdout, run.Stderr, run.StartedAt, durationMS, boolInt(run.Blocked)); err != nil {
		return nil, err
	}
	_ = s.AppendAudit("hook_run", ctx.Actor, map[string]any{
		"id":         run.ID,
		"event":      run.Event,
		"path":       run.Path,
		"work":       run.Work,
		"dest_realm": run.DestRealm,
		"mode":       run.Mode,
		"exit_code":  run.ExitCode,
		"blocked":    run.Blocked,
		"duration":   run.Duration,
	})
	if runErr != nil {
		return run, fmt.Errorf("hook %s failed with exit code %d", ctx.Event, exitCode)
	}
	return run, nil
}

func truncateHookOutput(s string) string {
	if len(s) <= HookOutputMax {
		return s
	}
	return s[:HookOutputMax] + "\n[truncated]"
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
