# Spec 007: CLI And Agent API

Status: Draft
Date: 2026-05-27
Project: Glyph

## Thesis

Glyph needs two first-class interfaces: a human CLI and an agent contract.

The `glyph` CLI should make Glyph understandable and operable by humans. The agent contract should let coding agents work directly with scoped source projections, work contexts, and publication requests without pretending to be humans typing Git commands.

The first agent contract can be a documented CLI protocol plus a skill document. A full MCP server can be added later when it clearly pays for itself.

## Problem

Most developer tools expose one interface and force everyone through it. Git's CLI became the universal substrate for humans, scripts, CI, and agents.

Glyph should not repeat that mistake.

At the same time, Glyph should avoid building a second integration surface too early. If agents can reliably use `glyph --json` with a small skill document that explains the operating model, command protocol, and safety rules, that is enough for bootstrap.

Humans need:

- Clear commands
- Status and review views
- Publication controls
- GitHub remote configuration
- Debuggable policy behavior

Agents need:

- Structured read/write operations
- Scoped context
- Stable workspace IDs
- Machine-readable errors
- Provenance capture
- Safe publication requests

## Goals

- Define the minimum human CLI shape.
- Define the minimum Agent API shape.
- Keep commands aligned with Glyph concepts, not Git concepts.
- Support bootstrap, realms, work contexts, projections, policy, and GitHub remotes.
- Make agent operations auditable by default.

## Non-Goals

- Final command names for every future feature.
- Building a full TUI or hosted UI.
- Making the CLI Git-compatible.
- Supporting every automation protocol.
- Designing SDKs for every language.
- Requiring MCP before agents can use Glyph effectively.

## CLI Principles

- Use Glyph vocabulary directly.
- Prefer explicit publication over implicit pushes.
- Show current realm and work context clearly.
- Avoid branch/stage/rebase terminology.
- Make policy denials explainable.
- Make GitHub remote behavior visible.
- Support a global `--json` flag for stable non-interactive agent output.
- Never prompt by default; commands must either complete from arguments/flags/stdin or return a structured error.

## Minimum CLI Surface

### Project

```sh
glyph init
glyph import ./path
glyph status --json
glyph graph
```

### Realms

```sh
glyph realm list
glyph realm create public
glyph realm inspect public
glyph realm grant public user:alice read
```

### Work Contexts

```sh
glyph work start auth-fix --from public
glyph work list
glyph work status auth-fix
glyph work snapshot auth-fix
glyph checkpoint auth-fix --message "auth refresh fix is ready for review"
glyph work prune auth-fix
glyph work discard auth-fix
```

### Files And Projections

```sh
glyph read auth-fix src/auth.ts
glyph write auth-fix src/auth.ts
glyph project auth-fix ./workspace/auth-fix
glyph diff auth-fix
```

### Publication

```sh
glyph publish auth-fix --to public
glyph publish auth-fix --to public --mode squash
glyph publish auth-fix --to public --mode preserve
glyph publication list
glyph publication inspect pub_01
```

### Policy

```sh
glyph policy check
glyph policy explain public src/secret.ts
glyph policy apply glyph.policy.yaml
```

### Remotes

```sh
glyph remote add origin github:owner/repo --mode export-only
glyph remote list
glyph remote sync origin
glyph remote inspect origin
```

### Mounts

```sh
glyph mount add vendor/parser github:owner/parser --mode pinned
glyph mount list
glyph mount update vendor/parser
glyph mount inspect vendor/parser
```

These commands are illustrative but should guide prototype design.

`glyph work start` should create a virtual workspace automatically. Users should not need a separate command to make work editable.

The CLI should expose `glyph checkpoint` for explicit milestones, while automatic snapshots remain the primary safety mechanism.

All commands should support non-interactive operation. Commands that need input must accept it via arguments, flags, stdin, or environment variables. In `--json` mode, successful command output should use a stable envelope:

```json
{"ok":true,"type":"status","data":{}}
```

Errors should be machine-readable:

```json
{"ok":false,"error":{"code":"not_found","message":"work context not found"}}
```

`glyph read --json` should return file bytes in an envelope instead of writing raw bytes. UTF-8 content should use `"encoding":"utf-8"` and binary content should use `"encoding":"base64"`:

```json
{"ok":true,"type":"read","data":{"work":"auth-fix","path":"src/auth.ts","encoding":"utf-8","content":"..."}}
```

## CLI Defaults

Projects may define command defaults in `glyph.yaml` so humans and agents do not have to repeat common flags.

Example:

```yaml
defaults:
  export:
    git:
      gitignore: generated
      gitinclude: generated
  viz:
    export:
      out: .glyph/visualizer
```

Explicit CLI flags always override `glyph.yaml` defaults. Defaults must never trigger publication, sync, destructive cleanup, or visibility widening by themselves.

## Agent Contract Principles

- Protocol-first, transport-later.
- Scoped by workspace and policy.
- Machine-readable success and failure.
- Every write records provenance.
- Publication requests are explicit.
- Agents can request context widening but cannot grant it to themselves.
- A checked-in skill document can be the first integration layer.
- The CLI must remain complete enough for non-interactive agent operation.

## Skill-Based Agent Integration

Prototype agents should be able to use Glyph through:

- The `glyph` CLI
- Global `--json`
- stdin for file writes
- stable exit codes
- stable JSON success and error envelopes
- a project skill document that explains how the agent should operate

The skill document should teach agents to:

- Start or reuse a work context before editing.
- Read and write through `glyph read` and `glyph write` when practical.
- Use `glyph project` only when a tool needs a real directory.
- Prefer `glyph diff` and `glyph checkpoint` before publication.
- Never publish implicitly.
- Choose `--mode squash` for noisy exploratory work and `--mode preserve` when audit-visible history matters.
- Prune completed workspace projections only after publication, discard, or explicit user request.
- Include concise write reasons for provenance.
- Treat policy denials as constraints, not obstacles to bypass.
- Use Git export or remote sync only when explicitly requested.

This keeps the first implementation small: one CLI, one source-control store, one documented agent protocol.

## Future Local API

A local API may still be useful once the CLI protocol stabilizes. It should mirror the CLI concepts instead of inventing a separate model.

### Workspace

- `create_workspace(task, from_realm, allowed_paths?)`
- `get_workspace(workspace_id)`
- `close_workspace(workspace_id)`

### Files

- `list(workspace_id, path)`
- `read(workspace_id, path)`
- `write(workspace_id, path, content, reason)`
- `delete(workspace_id, path, reason)`
- `search(workspace_id, query)`
- `diff(workspace_id)`

### Work

- `snapshot(workspace_id, reason)`
- `set_intent(workspace_id, intent)`
- `get_provenance(workspace_id)`
- `request_review(workspace_id)`
- `request_publication(workspace_id, to_realm, history_mode)`
- `prune_workspace(workspace_id, retention_policy?)`

### Policy

- `explain_denial(workspace_id, operation, path)`
- `request_context(workspace_id, reason, paths_or_realms)`

### Commands

- `run(workspace_id, command, policy)`

Command execution must be optional and sandboxed by policy. File access remains mediated by the workspace projection.

Every Agent API write should include a reason string. The reason can be short, but it becomes part of provenance and helps humans review agent work.

Command execution should be included in the first prototype only through a policy-mediated runner. Direct unmanaged shell access is not part of the Agent API.

The self-hosting API surface should be stable enough to support import, workspace creation, read, write, diff, snapshot/checkpoint, publication request, and Git export. Everything else may remain experimental.

Remote sync should be explicit by default. Automatic sync can be configured later per remote, but first prototype behavior should avoid surprising publication or import.

## Agent Metadata

Agent operations should record:

- Agent identity
- Agent provider
- Model or runtime identity where available
- User request or task summary
- Workspace ID
- Files read
- Files written
- Commands requested
- Tool outputs referenced
- Snapshots created
- Publication requests

Agent identities should include a provider, such as `agent:codex:session-01`, `agent:claude-code:session-02`, or `agent:cursor:session-03`.

This is not for surveillance theater. It is so humans can review agent work with context.

## Optional MCP Surface

Glyph may eventually expose an MCP server for coding agents.

The MCP server can wrap the Agent API and expose tools such as:

- `glyph_create_workspace`
- `glyph_read_file`
- `glyph_write_file`
- `glyph_search`
- `glyph_diff`
- `glyph_snapshot`
- `glyph_request_publication`
- `glyph_explain_policy_denial`

MCP should be an integration surface, not the internal architecture. It should not be required for early dogfooding if a CLI plus skill document is sufficient.

## Prototype Defaults

- The first CLI is named `glyph`.
- The first agent integration is `glyph --json` plus a project skill document.
- Global `--json` is supported for agent-friendly output.
- Commands are non-interactive by default.
- Agent workspaces are tied to work contexts.
- Writes require a reason string.
- `glyph work start` creates a virtual workspace.
- `glyph checkpoint` is the explicit milestone command.
- `glyph publish --mode squash` publishes a clean final-state event while retaining detailed history by policy.
- `glyph publish --mode preserve` exposes selected workspace history in the destination realm.
- `glyph work prune` removes active projections and caches, not retained source history.
- Command execution is available only through a policy-mediated runner.
- Denials return structured error codes.
- GitHub remote support starts with `export-only`.
- Remote sync is explicit by default.
- MCP support is optional and can follow once the CLI protocol proves useful.

## Success Criteria

This spec is successful if a prototype can:

- Initialize a Glyph project.
- Create realms and work contexts.
- Read and write files through a local Agent API.
- Materialize a workspace for human editing.
- Show status without Git concepts.
- Publish a work context to `public`.
- Configure a GitHub remote origin in `export-only` mode.
- Configure a GitHub repository as a mounted subdirectory.
- Export public state through the remote translator.
