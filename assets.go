package assets

import "embed"

// Files carries the documentation and agent guidance that should travel with
// an installed Glyph binary.
//
//go:embed docs/overview.mdx docs/quickstart.mdx docs/cli/*.mdx docs/concepts/*.mdx .agents/skills/glyph/SKILL.md
var Files embed.FS
