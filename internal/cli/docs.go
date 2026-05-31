package cli

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	assets "glyph"

	"github.com/spf13/cobra"
)

type embeddedDoc struct {
	ID          string `json:"id"`
	Path        string `json:"path"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Kind        string `json:"kind"`
	Content     string `json:"content,omitempty"`
}

type embeddedSkill struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Content     string `json:"content,omitempty"`
}

func docsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Read bundled Glyph documentation",
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(docsListCmd(), docsShowCmd())
	return cmd
}

func docsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List bundled documentation",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			docs, err := listEmbeddedDocs()
			if err != nil {
				return err
			}
			if modeFrom(cmd).JSON {
				return writeResponse(cmd, "docs_list", docs)
			}
			for _, doc := range docs {
				if err := humanf(cmd, "%s\t%s\t%s\n", doc.ID, doc.Kind, doc.Title); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func docsShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show one bundled document",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			doc, err := embeddedDocByID(args[0])
			if err != nil {
				return err
			}
			if modeFrom(cmd).JSON {
				return writeResponse(cmd, "doc", doc)
			}
			return humanf(cmd, "%s", doc.Content)
		},
	}
}

func skillsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Read bundled agent skills",
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(skillsListCmd(), skillsShowCmd())
	return cmd
}

func skillsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List bundled agent skills",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			skills, err := listEmbeddedSkills()
			if err != nil {
				return err
			}
			if modeFrom(cmd).JSON {
				return writeResponse(cmd, "skills_list", skills)
			}
			for _, skill := range skills {
				if err := humanf(cmd, "%s\t%s\n", skill.Name, skill.Title); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func skillsShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show one bundled agent skill",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			skill, err := embeddedSkillByName(args[0])
			if err != nil {
				return err
			}
			if modeFrom(cmd).JSON {
				return writeResponse(cmd, "skill", skill)
			}
			return humanf(cmd, "%s", skill.Content)
		},
	}
}

func listEmbeddedDocs() ([]embeddedDoc, error) {
	var docs []embeddedDoc
	for _, root := range []string{"docs"} {
		if err := fs.WalkDir(assets.Files, root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !isDocPath(path) {
				return nil
			}
			doc, err := readEmbeddedDoc(path, false)
			if err != nil {
				return err
			}
			docs = append(docs, doc)
			return nil
		}); err != nil {
			return nil, err
		}
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].ID < docs[j].ID })
	return docs, nil
}

func embeddedDocByID(id string) (embeddedDoc, error) {
	docs, err := listEmbeddedDocs()
	if err != nil {
		return embeddedDoc{}, err
	}
	for _, doc := range docs {
		if doc.ID == id {
			return readEmbeddedDoc(doc.Path, true)
		}
	}
	return embeddedDoc{}, fmt.Errorf("document %q not found", id)
}

func readEmbeddedDoc(path string, includeContent bool) (embeddedDoc, error) {
	content, err := readEmbeddedText(path)
	if err != nil {
		return embeddedDoc{}, err
	}
	meta := frontMatter(content)
	doc := embeddedDoc{
		ID:          docID(path),
		Path:        path,
		Title:       titleOrFallback(meta["title"], docID(path)),
		Description: meta["description"],
		Kind:        docKind(path),
	}
	if includeContent {
		doc.Content = content
	}
	return doc, nil
}

func listEmbeddedSkills() ([]embeddedSkill, error) {
	var skills []embeddedSkill
	if err := fs.WalkDir(assets.Files, ".agents/skills", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Base(path) != "SKILL.md" {
			return nil
		}
		skill, err := readEmbeddedSkill(path, false)
		if err != nil {
			return err
		}
		skills = append(skills, skill)
		return nil
	}); err != nil {
		return nil, err
	}
	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })
	return skills, nil
}

func embeddedSkillByName(name string) (embeddedSkill, error) {
	skills, err := listEmbeddedSkills()
	if err != nil {
		return embeddedSkill{}, err
	}
	for _, skill := range skills {
		if skill.Name == name {
			return readEmbeddedSkill(skill.Path, true)
		}
	}
	return embeddedSkill{}, fmt.Errorf("skill %q not found", name)
}

func readEmbeddedSkill(path string, includeContent bool) (embeddedSkill, error) {
	content, err := readEmbeddedText(path)
	if err != nil {
		return embeddedSkill{}, err
	}
	meta := frontMatter(content)
	name := meta["name"]
	if name == "" {
		name = filepath.Base(filepath.Dir(path))
	}
	skill := embeddedSkill{
		Name:        name,
		Path:        path,
		Title:       titleOrFallback(meta["name"], name),
		Description: meta["description"],
	}
	if includeContent {
		skill.Content = content
	}
	return skill, nil
}

func readEmbeddedText(path string) (string, error) {
	b, err := assets.Files.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func isDocPath(path string) bool {
	if !(strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".mdx")) {
		return false
	}
	if strings.HasPrefix(path, "docs/specs/") || strings.HasPrefix(path, "docs/plans/") {
		return false
	}
	return path == "docs/overview.mdx" ||
		path == "docs/quickstart.mdx" ||
		strings.HasPrefix(path, "docs/cli/") ||
		strings.HasPrefix(path, "docs/concepts/")
}

func docID(path string) string {
	id := strings.TrimPrefix(path, "docs/")
	id = strings.TrimSuffix(id, filepath.Ext(id))
	return filepath.ToSlash(id)
}

func docKind(path string) string {
	switch {
	case strings.HasPrefix(path, "docs/cli/"):
		return "cli"
	case strings.HasPrefix(path, "docs/concepts/"):
		return "concept"
	default:
		return "guide"
	}
}

func frontMatter(content string) map[string]string {
	meta := map[string]string{}
	if !strings.HasPrefix(content, "---\n") {
		return meta
	}
	end := strings.Index(content[len("---\n"):], "\n---")
	if end < 0 {
		return meta
	}
	block := content[len("---\n") : len("---\n")+end]
	for _, line := range strings.Split(block, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"`)
		if key != "" {
			meta[key] = value
		}
	}
	return meta
}

func titleOrFallback(title, fallback string) string {
	if title != "" {
		return title
	}
	return fallback
}
