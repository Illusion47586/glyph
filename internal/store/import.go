package store

import (
	"os"
	"path/filepath"
	"strings"
)

func (s *Store) ImportWorkspace() (int, error) {
	manifest, err := LoadBootstrapManifest(s.Root)
	if err != nil {
		return 0, err
	}
	var imported []map[string]string
	count := 0
	err = filepath.WalkDir(s.Root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(s.Root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if d.IsDir() && isExcludedDir(rel, manifest.GenesisImport.Exclude) {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		if !isIncluded(rel, manifest.GenesisImport.Include) || isExcluded(rel, manifest.GenesisImport.Exclude) {
			return nil
		}
		src, err := s.StoreFile(path, rel, "public")
		if err != nil {
			return err
		}
		imported = append(imported, map[string]string{"path": rel, "hash": src.Hash})
		count++
		return nil
	})
	if err != nil {
		return count, err
	}
	if err := s.AppendAudit("genesis_import", actor(manifest), map[string]any{
		"manifest":    "glyph.yaml",
		"included":    imported,
		"excluded":    manifest.GenesisImport.Exclude,
		"realms":      []string{"public", "maintainers"},
		"transcripts": "excluded-by-default",
	}); err != nil {
		return count, err
	}
	return count, nil
}

func actor(manifest *BootstrapManifest) string {
	if manifest.Identity.BootstrapUser != "" {
		return manifest.Identity.BootstrapUser
	}
	return BootstrapUser
}

func isIncluded(path string, patterns []string) bool {
	for _, pattern := range patterns {
		pattern = filepath.ToSlash(pattern)
		if pattern == path {
			return true
		}
		if strings.HasSuffix(pattern, "/**") {
			prefix := strings.TrimSuffix(pattern, "/**")
			if path == prefix || strings.HasPrefix(path, prefix+"/") {
				return true
			}
		}
	}
	return false
}

func isExcluded(path string, patterns []string) bool {
	for _, pattern := range patterns {
		pattern = filepath.ToSlash(pattern)
		if pattern == path {
			return true
		}
		if strings.HasSuffix(pattern, "/**") {
			prefix := strings.TrimSuffix(pattern, "/**")
			if path == prefix || strings.HasPrefix(path, prefix+"/") {
				return true
			}
		}
	}
	return false
}

func isExcludedDir(path string, patterns []string) bool {
	return isExcluded(path, patterns)
}
