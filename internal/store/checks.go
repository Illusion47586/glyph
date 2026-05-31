package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type CheckOptions struct {
	Realm      string
	Out        string
	Keep       bool
	Names      []string
	GitOptions GitExportOptions
	Timeout    time.Duration
}

type CheckResult struct {
	Realm  string     `json:"realm"`
	Out    string     `json:"out"`
	Kept   bool       `json:"kept"`
	OK     bool       `json:"ok"`
	Checks []CheckRun `json:"checks"`
}

type CheckRun struct {
	Name     string `json:"name"`
	Command  string `json:"command"`
	OK       bool   `json:"ok"`
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
	Duration string `json:"duration"`
}

func (s *Store) CheckRealmExport(opts CheckOptions) (*CheckResult, error) {
	if opts.Realm == "" {
		opts.Realm = "public"
	}
	if opts.Timeout == 0 {
		opts.Timeout = 2 * time.Minute
	}
	manifest, err := LoadBootstrapManifest(s.Root)
	if err != nil {
		return nil, err
	}
	checks := checksForRealm(manifest, opts.Realm)
	checks, err = filterChecks(checks, opts.Names)
	if err != nil {
		return nil, err
	}
	if len(checks) == 0 {
		return &CheckResult{Realm: opts.Realm, OK: true, Checks: []CheckRun{}}, nil
	}
	out := opts.Out
	if out == "" {
		out, err = os.MkdirTemp("", "glyph-check-"+opts.Realm+"-*")
		if err != nil {
			return nil, err
		}
	} else {
		if err := ensureEmptyOrMissing(out); err != nil {
			return nil, err
		}
		if err := os.MkdirAll(out, 0o755); err != nil {
			return nil, err
		}
	}
	if !opts.Keep {
		defer os.RemoveAll(out)
	}
	gitOpts, err := opts.GitOptions.normalized()
	if err != nil {
		return nil, err
	}
	if err := s.MaterializeRealm(opts.Realm, out); err != nil {
		return nil, err
	}
	if _, err := s.GenerateGitCompatibilityFiles(out, gitOpts); err != nil {
		return nil, err
	}
	result := &CheckResult{Realm: opts.Realm, Out: out, Kept: opts.Keep, OK: true}
	for _, check := range checks {
		run := runExportCheck(out, check, opts.Timeout)
		if !run.OK {
			result.OK = false
		}
		result.Checks = append(result.Checks, run)
	}
	if !result.OK {
		return result, fmt.Errorf("public export checks failed")
	}
	return result, nil
}

func checksForRealm(manifest *BootstrapManifest, realm string) []ExportCheck {
	if realm == "public" {
		return manifest.Checks.PublicExport
	}
	return nil
}

func filterChecks(checks []ExportCheck, names []string) ([]ExportCheck, error) {
	for _, check := range checks {
		if check.Name == "" {
			return nil, fmt.Errorf("check name is required")
		}
		if check.Command == "" {
			return nil, fmt.Errorf("check %q command is required", check.Name)
		}
	}
	if len(names) == 0 {
		return checks, nil
	}
	wanted := map[string]bool{}
	for _, name := range names {
		wanted[name] = true
	}
	var out []ExportCheck
	for _, check := range checks {
		if wanted[check.Name] {
			out = append(out, check)
			delete(wanted, check.Name)
		}
	}
	if len(wanted) > 0 {
		var missing []string
		for name := range wanted {
			missing = append(missing, name)
		}
		return nil, fmt.Errorf("unknown check(s): %s", strings.Join(missing, ", "))
	}
	return out, nil
}

func runExportCheck(dir string, check ExportCheck, timeout time.Duration) CheckRun {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := shellCommand(ctx, check.Command)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	run := CheckRun{
		Name:     check.Name,
		Command:  check.Command,
		OK:       err == nil,
		ExitCode: exitCode(err),
		Duration: time.Since(start).String(),
	}
	if len(out) > 0 {
		if err == nil {
			run.Stdout = trimOutput(string(out))
		} else {
			run.Stderr = trimOutput(string(out))
		}
	}
	if ctx.Err() == context.DeadlineExceeded {
		run.OK = false
		run.ExitCode = -1
		run.Stderr = trimOutput(run.Stderr + "\ncheck timed out")
	}
	return run
}

func shellCommand(ctx context.Context, command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "cmd", "/C", command)
	}
	return exec.CommandContext(ctx, "sh", "-c", command)
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

func trimOutput(out string) string {
	out = strings.TrimSpace(out)
	if len(out) <= HookOutputMax {
		return out
	}
	return out[:HookOutputMax] + "\n... truncated ..."
}
