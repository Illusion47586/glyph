//go:build !windows

package install

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func defaultBinDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "bin"), nil
}

func executableName() string {
	return "glyph"
}

func samePath(a, b string) bool {
	return a == b
}

func configureUserPath(binDir string) (bool, []string, error) {
	if pathContainsDir(os.Getenv("PATH"), binDir) {
		return false, nil, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false, nil, err
	}
	files := shellConfigFiles(home)
	var modified []string
	for _, file := range files {
		changed, err := ensureShellConfig(file, binDir)
		if err != nil {
			return false, modified, err
		}
		if changed {
			modified = append(modified, file)
		}
	}
	return len(modified) > 0, modified, nil
}

func shellConfigFiles(home string) []string {
	candidates := []string{
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".profile"),
		filepath.Join(home, ".config", "fish", "config.fish"),
	}
	var files []string
	for _, file := range candidates {
		if _, err := os.Stat(file); err == nil {
			files = append(files, file)
		}
	}
	if len(files) > 0 {
		return files
	}
	shell := strings.TrimSuffix(filepath.Base(os.Getenv("SHELL")), ".exe")
	switch shell {
	case "fish":
		return []string{filepath.Join(home, ".config", "fish", "config.fish")}
	case "bash":
		if runtime.GOOS == "darwin" {
			return []string{filepath.Join(home, ".bash_profile")}
		}
		return []string{filepath.Join(home, ".bashrc")}
	default:
		return []string{filepath.Join(home, ".zshrc")}
	}
}

func ensureShellConfig(file, binDir string) (bool, error) {
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return false, err
	}
	data, err := os.ReadFile(file)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	if strings.Contains(text, "BEGIN GLYPH PATH") || strings.Contains(text, shellPathEntry(binDir)) || strings.Contains(text, filepath.ToSlash(binDir)) {
		return false, nil
	}
	block := shellConfigBlock(file, binDir)
	prefix := ""
	if len(text) > 0 && !strings.HasSuffix(text, "\n") {
		prefix = "\n"
	}
	f, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return false, err
	}
	defer f.Close()
	_, err = f.WriteString(prefix + block)
	if err != nil {
		return false, err
	}
	return true, nil
}

func shellConfigBlock(file, binDir string) string {
	entry := shellPathEntry(binDir)
	if strings.HasSuffix(file, "config.fish") {
		return "\n# BEGIN GLYPH PATH\nset -gx PATH \"" + entry + "\" $PATH\n# END GLYPH PATH\n"
	}
	return "\n# BEGIN GLYPH PATH\n" + shSessionCommand(binDir) + "\n# END GLYPH PATH\n"
}

func currentSessionCommands(binDir string) []string {
	return []string{shSessionCommand(binDir)}
}
