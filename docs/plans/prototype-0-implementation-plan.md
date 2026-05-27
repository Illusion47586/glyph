---
title: "Prototype 0 Implementation Plan: Local Self-Hosting Glyph In Go"
description: "Glyph project documentation."
---

## Summary

Build the first Glyph vertical slice as a Go CLI named `glyph`, with no Git repository initialized for this workspace. The prototype creates a `.glyph/` local store, imports this spec workspace from `glyph.yaml`, creates `public` and `maintainers` realms, supports local work contexts and virtual workspaces, publishes selected work to `public`, writes append-only audit events, exports `public` to a clean Git repository, and can push that export to a configured GitHub remote.

Deferred from Prototype 0: Git import, executable external mounts, MCP server, full command runner, credential storage UI, signing enforcement, hosted Glyph, full conflict resolution, and advanced policy.

## Key Changes

- Scaffold a Go module with a `glyph` CLI using Cobra.
- Use `.glyph/` as the only local store namespace.
- Use `glyph.yaml` as the bootstrap manifest.
- Use SQLite at `.glyph/store.db` as the canonical local source-control database.
- Store content blobs under `.glyph/content/`, addressed from SQLite.
- Store append-only audit events in `.glyph/audit/events.jsonl`.
- Support typed IDs for content, sources, realms, work contexts, snapshots, and publications.
- Implement `glyph init`, `glyph import`, `glyph status`, `glyph graph`.
- Implement `glyph work start/list/status/snapshot/discard`.
- Implement `glyph read`, `glyph write`, `glyph project`, `glyph diff`.
- Implement `glyph publish`, `glyph publication list/inspect`.
- Implement `glyph export git --realm public --out <dir>`.
- Implement `glyph remote add/list/inspect/sync` for `export-only` GitHub remotes.

## Test Plan

- Unit tests for manifest parsing, content hashing, store initialization, projection filtering, snapshot creation, publication state, audit JSONL, and store version checks.
- Integration tests for `glyph init && glyph import ./`, work context edits, publication, Git export, and remote sync against a local bare Git remote.

## Assumptions

- Runtime is Go.
- Store namespace is `.glyph/`.
- Bootstrap actor is `user:self:dhruv`.
- First realms are `public` and `maintainers`.
- GitHub sync uses installed `git` and existing credentials.
- This workspace still does not use Git for its own canonical history.
