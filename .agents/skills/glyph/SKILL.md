---
name: glyph
description: Use Glyph as the source-control system for this project through the non-interactive CLI protocol.
---

# Glyph Agent Skill

Use this skill when working in the Glyph repository or any project that uses Glyph for source control.

Glyph is an agent-native source-control system. The CLI is the first agent integration surface. Use `glyph --json` for machine-readable output whenever the result will be parsed or used for follow-up actions.

## Core Rules

- Treat `.glyph/store.db` as the local source-control database, equivalent in role to a `.git` directory.
- Do not run `git init` in the working project during bootstrap.
- Do not publish, export, or sync unless the user asks for it.
- Prefer explicit work contexts over untracked edits.
- Include concise reasons when writing files through Glyph.
- Treat policy errors as constraints to report or request context for, not as prompts to bypass Glyph.

## Standard Workflow

1. Check store state:

```sh
glyph status --json
```

2. Start or reuse a work context:

```sh
glyph work start task-name --from public --json
glyph work list --json
```

3. Read files through Glyph when practical:

```sh
glyph read task-name path/to/file --json
```

4. Write through stdin with a reason:

```sh
glyph write task-name path/to/file --reason "short reason" --json < edited-file
```

5. Use a projected workspace only when tools require a real directory:

```sh
glyph project task-name /tmp/glyph-task-name --json
```

6. Review and checkpoint:

```sh
glyph diff task-name --json
glyph checkpoint task-name --message "short milestone" --json
```

7. Publish only on explicit request:

```sh
glyph publish task-name --to public --json
```

## JSON Contract

Successful commands return:

```json
{"ok":true,"type":"status","data":{}}
```

Failed commands return:

```json
{"ok":false,"error":{"code":"not_found","message":"work context not found"}}
```

`glyph read --json` returns content with an explicit encoding:

```json
{"ok":true,"type":"read","data":{"work":"task-name","path":"file.txt","encoding":"utf-8","content":"..."}}
```

If `encoding` is `base64`, decode `content` before editing.

## Git Compatibility

Git is an export and remote compatibility layer, not the canonical store during bootstrap.

Use Git export only when requested:

```sh
glyph export git --realm public --out /tmp/glyph-public-export --json
```

Use remote sync only when requested:

```sh
glyph remote sync origin --json
```
