package install

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Result struct {
	Binary                 string   `json:"binary"`
	BinDir                 string   `json:"bin_dir"`
	InstalledPath          string   `json:"installed_path"`
	Installed              bool     `json:"installed"`
	PathAlreadyConfigured  bool     `json:"path_already_configured"`
	PathConfigured         bool     `json:"path_configured"`
	CurrentSessionCommands []string `json:"current_session_commands,omitempty"`
	ModifiedFiles          []string `json:"modified_files,omitempty"`
}

func Install(exePath, binDir string) (*Result, error) {
	if exePath == "" {
		var err error
		exePath, err = os.Executable()
		if err != nil {
			return nil, err
		}
	}
	if binDir == "" {
		var err error
		binDir, err = defaultBinDir()
		if err != nil {
			return nil, err
		}
	}
	exePath, err := filepath.Abs(exePath)
	if err != nil {
		return nil, err
	}
	binDir, err = filepath.Abs(binDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return nil, err
	}
	installedPath := filepath.Join(binDir, executableName())
	installed, err := copyExecutable(exePath, installedPath)
	if err != nil {
		return nil, err
	}
	already := pathContainsDir(os.Getenv("PATH"), binDir)
	configured, modified, err := configureUserPath(binDir)
	if err != nil {
		return nil, err
	}
	return &Result{
		Binary:                 exePath,
		BinDir:                 binDir,
		InstalledPath:          installedPath,
		Installed:              installed,
		PathAlreadyConfigured:  already,
		PathConfigured:         configured,
		CurrentSessionCommands: currentSessionCommands(binDir),
		ModifiedFiles:          modified,
	}, nil
}

func copyExecutable(from, to string) (bool, error) {
	fromInfo, err := os.Stat(from)
	if err != nil {
		return false, err
	}
	if toInfo, err := os.Stat(to); err == nil && os.SameFile(fromInfo, toInfo) {
		return false, nil
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	in, err := os.Open(from)
	if err != nil {
		return false, err
	}
	defer in.Close()
	tmp, err := os.CreateTemp(filepath.Dir(to), ".glyph-install-*")
	if err != nil {
		return false, err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := io.Copy(tmp, in); err != nil {
		_ = tmp.Close()
		return false, err
	}
	if err := tmp.Chmod(0o755); err != nil {
		_ = tmp.Close()
		return false, err
	}
	if err := tmp.Close(); err != nil {
		return false, err
	}
	if err := os.Rename(tmpName, to); err != nil {
		return false, err
	}
	return true, nil
}

func pathContainsDir(pathValue, dir string) bool {
	if pathValue == "" {
		return false
	}
	want, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	want = filepath.Clean(want)
	for _, entry := range filepath.SplitList(pathValue) {
		got, err := filepath.Abs(entry)
		if err != nil {
			continue
		}
		if samePath(filepath.Clean(got), want) {
			return true
		}
	}
	return false
}

func shellPathEntry(dir string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.ToSlash(dir)
	}
	home = filepath.Clean(home)
	clean := filepath.Clean(dir)
	if clean == home {
		return "$HOME"
	}
	if strings.HasPrefix(clean, home+string(os.PathSeparator)) {
		return "$HOME/" + filepath.ToSlash(strings.TrimPrefix(clean, home+string(os.PathSeparator)))
	}
	return filepath.ToSlash(clean)
}

func shSessionCommand(dir string) string {
	return fmt.Sprintf("export PATH=\"%s:$PATH\"", shellPathEntry(dir))
}
