//go:build windows

package install

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func defaultBinDir() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, "AppData", "Local")
	}
	return filepath.Join(base, "Glyph", "bin"), nil
}

func executableName() string {
	return "glyph.exe"
}

func samePath(a, b string) bool {
	return strings.EqualFold(a, b)
}

func configureUserPath(binDir string) (bool, []string, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return false, nil, err
	}
	defer k.Close()
	current, _, err := k.GetStringValue("Path")
	if err != nil && err != registry.ErrNotExist {
		return false, nil, err
	}
	if pathContainsDir(current, binDir) {
		return false, nil, nil
	}
	next := binDir
	if strings.TrimSpace(current) != "" {
		next += string(os.PathListSeparator) + current
	}
	if err := k.SetStringValue("Path", next); err != nil {
		return false, nil, err
	}
	return true, nil, nil
}

func currentSessionCommands(binDir string) []string {
	return []string{`$env:Path = "` + binDir + `;$env:Path"`}
}
