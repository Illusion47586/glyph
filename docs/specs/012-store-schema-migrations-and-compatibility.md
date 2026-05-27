---
title: "Spec 012: Store Schema, Migrations, And Compatibility"
description: "Glyph design specification."
---

Status: Draft
Date: 2026-05-27
Project: Glyph

## Thesis

Glyph will dogfood early, so the local store must evolve without losing the project it is trying to preserve.

Schema versioning and migrations are not enterprise polish. They are survival gear.

## Problem

The first `.glyph/` store will contain the genesis graph, specs, work contexts, publications, policy, and audit records. SQLite is the canonical local store database at `.glyph/store.db`; if its schema changes without a migration story, self-hosting becomes brittle.

Glyph needs to know:

- Which store version is present?
- Which CLI/API versions can read it?
- How are migrations applied?
- Can migrations be audited or rolled back?
- What happens when a newer store is opened by an older CLI?

## Goals

- Define store manifest requirements.
- Define schema versioning.
- Define migration events.
- Protect audit history during migrations.
- Support forward-safe failure.
- Keep v1 local-first.

## Non-Goals

- Defining distributed replication migrations.
- Supporting every historical experimental schema forever.
- Building a hosted migration service.
- Freezing all object formats before prototype work.

## Store Manifest

The `.glyph/` directory should contain a store manifest.

Example:

```yaml
store:
  version: 1
  created_by: glyph/0.1.0
  created_at: 2026-05-27T00:00:00Z
  project: glyph
  object_format: glyph-object-v1
  audit_log: audit/events.jsonl
```

The manifest is separate from bootstrap `glyph.yaml`. `glyph.yaml` describes the project before self-hosting; `.glyph/manifest.yaml` describes the actual Glyph store.

The SQLite database at `.glyph/store.db` is the authoritative local schema-bearing artifact. The manifest records the expected schema version so incompatible CLIs fail closed.

## Migration Rules

Migrations must:

- Read the existing store version.
- Refuse unknown future versions.
- Snapshot or back up metadata before destructive changes.
- Append a migration audit event.
- Preserve object IDs when possible.
- Never rewrite append-only audit history.
- Report lossy transformations explicitly.

## Compatibility

The CLI should declare:

- Minimum readable store version
- Maximum readable store version
- Current writable store version

If a store is too new, Glyph should fail with a clear error rather than attempting partial reads.

## Prototype Defaults

- First store version is `1`.
- Store metadata lives in `.glyph/manifest.yaml`.
- Canonical local store data lives in `.glyph/store.db`.
- Audit events remain append-only JSONL.
- Migrations are explicit commands, not automatic surprises.
- Unknown newer stores fail closed.
- Lossy migrations require human confirmation.

## Success Criteria

This spec is successful if a prototype can:

- Create `.glyph/manifest.yaml`.
- Refuse unsupported store versions.
- Apply a simple migration with an audit event.
- Preserve existing audit logs.
- Report migration compatibility in `glyph status`.
