---
title: "Spec 014: Hooks And Automation Extension Points"
description: "Glyph design specification."
---

Status: Draft
Date: 2026-05-27
Project: Glyph

## Thesis

Glyph should support hook-like automation, but hooks must be explicit, auditable, non-interactive, and policy-aware.

Git hooks are useful because they let teams attach local behavior to source-control events. They are also awkward because they are usually local, hidden, inconsistently versioned, and easy to surprise humans or agents with. Glyph should keep the power while making hook behavior inspectable and agent-safe.

## Goals

- Define hook events around Glyph-native operations.
- Let hooks block dangerous operations such as publication.
- Record hook execution in audit history.
- Keep hooks non-interactive and deterministic by default.
- Support local prototype hooks without requiring hosted infrastructure.
- Leave room for future policy-managed and hosted hooks.

## Non-Goals

- Recreating every Git hook name.
- Running arbitrary hooks without policy.
- Building a CI system in the first prototype.
- Making hooks the only automation model.

## Hook Types

### Local Hooks

Local hooks live inside the `.glyph/` store and apply to one checkout or local store.

Prototype location:

```text
.glyph/hooks/pre-publish
.glyph/hooks/post-publish
```

Local hooks are useful for bootstrap and personal safety checks. They are not automatically public project policy.

### Project Hooks

Project hooks are versioned hook definitions stored in source, such as `glyph.hooks.yaml`.

Project hooks should be reviewed like code and may be visible only in selected realms. They can be added after the local hook runner exists.

### Policy Hooks

Policy hooks are enforced by realm or organization policy. They may run locally, in CI, or in a hosted Glyph service.

Policy hooks decide whether an operation can proceed. Local hooks may add stricter checks but cannot bypass policy hooks.

## Initial Hook Events

Prototype events:

- `pre-publish`: runs before a work context is published into a destination realm.
- `post-publish`: runs after successful publication.

Future events:

- `pre-write`
- `post-write`
- `pre-command`
- `post-command`
- `pre-export`
- `post-export`
- `pre-sync`
- `post-sync`
- `pre-prune`
- `post-prune`

## Hook Contract

Hooks are non-interactive commands.

They receive context through environment variables:

- `GLYPH_EVENT`
- `GLYPH_ROOT`
- `GLYPH_STORE`
- `GLYPH_WORK`
- `GLYPH_DEST_REALM`
- `GLYPH_PUBLICATION_MODE`
- `GLYPH_PUBLICATION_ID`, for post-publication hooks

Exit behavior:

- Exit code `0`: hook passed.
- Non-zero exit code: hook failed.
- Failed `pre-*` hooks block the operation.
- Failed `post-*` hooks record failure but do not roll back already-completed publication in the first prototype.

Hooks must not prompt. In agent mode, failures should return structured JSON errors.

## Audit Requirements

Each hook run records:

- Hook event
- Hook path or configured command
- Actor
- Work context
- Destination realm, when applicable
- Exit code
- Started timestamp
- Duration
- Captured stdout and stderr, bounded by policy
- Whether it blocked the operation

Hook output may contain secrets. Audit storage and realm visibility must treat hook logs as potentially sensitive.

## Security Requirements

Hooks execute code. Glyph must treat them as privileged automation.

The first local prototype may run executable local hooks directly, but future versions should route hooks through the command runner and sandbox policy from Spec 011.

Policy should control:

- Which hooks may run
- Which actors may install or modify hooks
- Which events hooks can block
- Timeout duration
- Environment variables exposed to hooks
- Whether hook logs are retained or redacted

## CLI Surface

Prototype commands:

```sh
glyph hook list --json
glyph hook run pre-publish --work auth-fix --to public --mode squash --json
```

Hooks run automatically around supported operations:

```sh
glyph publish auth-fix --to public --mode squash --json
```

## Prototype Defaults

- Local hooks live in `.glyph/hooks/`.
- Only executable files are run.
- `pre-publish` blocks publication on failure.
- `post-publish` records failure but does not roll back publication.
- Hook timeout defaults to 30 seconds.
- Hook runs are recorded in SQLite and audit JSONL.
- Hook output capture is bounded.
- Project and policy hooks are specified but deferred.

## Success Criteria

This spec is successful if a prototype can:

- List installed local hooks.
- Run a local hook with Glyph context environment variables.
- Block publication when `pre-publish` fails.
- Publish when `pre-publish` succeeds.
- Run `post-publish` after successful publication.
- Record hook runs in the local store and audit log.
