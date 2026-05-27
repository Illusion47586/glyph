# Spec 004: Storage Model And Object Graph

Status: Draft
Date: 2026-05-27
Project: Glyph

## Thesis

Glyph needs a canonical object graph that can represent source, work, visibility, policy, and provenance without inheriting Git's object model.

The storage model should be content-addressed where useful, append-friendly for auditability, and flexible enough to support projections, work contexts, and translators.

## Problem

Git stores blobs, trees, commits, and refs. That model is excellent for distributed snapshots, but Glyph needs additional native concepts:

- Realms
- Projections
- Work contexts
- Automatic snapshots
- Visibility labels
- Publication events
- Agent provenance
- Policy decisions
- Git import/export metadata

Trying to encode these as branches, commits, refs, and commit messages would make Glyph a Git wrapper. Glyph needs its own object graph.

## Goals

- Define the major object types Glyph stores.
- Separate content storage from visibility and work metadata.
- Support efficient projections.
- Support audit records for import, publication, and policy changes.
- Allow local-first prototypes.
- Leave room for future distributed sync.

## Non-Goals

- Selecting the final database technology.
- Defining cryptographic signing in detail.
- Defining distributed consensus.
- Optimizing for huge monorepos in the first prototype.
- Finalizing wire formats.

## Object Types

### Content Object

A content object stores bytes and basic content metadata.

Fields:

- Object ID
- Content hash
- Byte length
- Media type or detected kind
- Optional encoding metadata

Content objects do not by themselves imply visibility.

### Tree Object

A tree object maps paths to content objects or other tree objects.

Fields:

- Tree ID
- Entries
- Parent tree references where useful
- Generated or materialized status

### Source Object

A source object is a policy-aware unit of source identity.

It can represent:

- File
- Directory
- Generated artifact
- External import
- Metadata object

Fields:

- Source object ID
- Current content or tree reference
- Path identity
- Labels
- Ownership metadata
- Provenance links

### External Mount Object

An external mount object records a source root mounted at a path inside another source graph.

Fields:

- Mount ID
- Mount path
- Source type
- Remote origin
- Pinned revision or source graph reference
- Import policy
- Export policy
- Allowed realms
- Local write policy

External mount objects preserve the boundary between a parent project and a separately maintained dependency or subproject.

### Realm Primitive Object

A realm object defines a named projection policy.

Fields:

- Realm ID
- Name
- Description
- Policy reference
- Grants
- Default labels
- Publication rules

### Work Context Object

A work context object records active or historical work.

Fields:

- Work context ID
- Owner identity
- Agent or human provenance
- Base projection
- Overlay references
- Snapshots
- Intent
- Status

### Snapshot Object

A snapshot object records a work context state at a point in time.

Fields:

- Snapshot ID
- Work context ID
- Tree reference
- Timestamp
- Trigger
- Actor
- Parent snapshot references

### Publication Object

A publication object records visibility widening or source promotion.

Fields:

- Publication ID
- Source realm
- Destination realm
- Included objects
- Excluded objects
- Actor
- Review approvals
- Policy checks
- Timestamp
- Translator outputs

### Policy Object

A policy object records versioned access and publication rules.

Fields:

- Policy ID
- Policy document
- Version
- Author
- Effective time
- Superseded policy reference

### Provenance Object

A provenance object records where work or content came from.

Examples:

- Human edit
- Agent write
- Git import
- Generated file
- Refactoring tool
- Package update

## Storage Invariants

1. **Content is not visibility**
   Possessing bytes in storage does not mean they are visible in a realm.

2. **Policies are versioned**
   Policy changes are stored as objects with history.

3. **Publication is auditable**
   Every visibility-widening operation creates a publication object.

4. **Imports are honest**
   Imported Git data is provenance, not native Glyph truth unless adopted through realm objects.

5. **Objects are addressable**
   Important objects have stable IDs suitable for audit, references, and translators.

6. **Materialized projections are disposable**
   Filesystem checkouts and Git exports can be regenerated from canonical objects.

## Storage Decisions

Object IDs should use typed hash-based identifiers by default, such as `content:sha256:...`, `tree:sha256:...`, and `work:...`. Content-like objects should be content-addressed. Event-like objects may include typed generated IDs plus content hashes for integrity.

Glyph should borrow Git's plumbing discipline without inheriting Git's porcelain model. The useful lesson from "Write Yourself a Git" is that a small version-control core can be built from simple, inspectable primitives: repository discovery, object read/write, content hashing, tree materialization, reference-like pointers, status comparisons, and export commands. Glyph should keep similarly sharp plumbing commands for agents and debugging, while replacing commits, branches, staging, and refs with work contexts, realms, snapshots, publications, and policy.

The canonical object graph should store complete file states logically, not patch chains. Diffs are derived review artifacts and may be cached, but source reconstruction should not depend on replaying diffs.

Complete logical snapshots must not mean physically copying every file every time. Snapshot and tree objects should store path-to-object mappings, while file bytes are stored as content-addressed objects and deduplicated by hash. If two snapshots reference identical file content, both snapshots point to the same content object.

For large files, Glyph should support chunked content objects so an update to one region does not require storing another full copy of the file. The first prototype may begin with whole-file content addressing and add chunking behind the same content object abstraction later.

Path identity should survive renames as a stable source object. Paths can change, but the source object remains the continuity anchor.

Policies should be stored as source graph objects and can be visible through realms according to policy. Public projects should normally expose public policy, while private policy details may remain restricted.

The first prototype should make policy objects, publication objects, audit events, and imported provenance append-only. Source content can be superseded by newer objects rather than mutated.

Local storage should use SQLite as the canonical local source-control database. In Git terms, `.glyph/store.db` is the closest equivalent to the `.git` directory's metadata and object graph: it records source objects, realms, work contexts, snapshots, publications, remotes, mounts, schema version, and indexes.

Large byte content may live as content-addressed files beside the SQLite database under `.glyph/content/`, but those blobs are still part of the `.glyph` store and are addressed from SQLite. Audit events live under `.glyph/audit/` as append-only JSONL so they remain easy to inspect and stream.

Large binary files should be represented as content objects with metadata and optional chunking. The first prototype can store them as external content-addressed blobs and avoid diffing them deeply.

Glyph should keep an index-like table for fast status and projection operations, but it should not expose a staging area as a user primitive. Unlike Git's index, Glyph's index is an implementation detail for answering questions such as:

- Which paths changed in a workspace projection?
- Which content hashes belong to this snapshot?
- Which source objects differ between a workspace and its base realm?
- Which workspace projections are stale, conflicting, or ready to publish?

The index may represent incomplete or conflicted states. Published tree and realm projections must remain complete, unambiguous states.

## Plumbing Commands

Glyph should expose a small set of low-level commands for debugging, agent integration, and compatibility testing. These commands should be stable enough for tooling but clearly lower-level than the human workflow.

Candidate plumbing commands:

```sh
glyph object hash <path> --json
glyph object read <id> --json
glyph object write --type content --json < file
glyph tree read <id> --json
glyph tree materialize <id> --out ./dir --json
glyph store find --json
glyph store fsck --json
```

These mirror the educational shape of Git plumbing such as object hashing, object inspection, tree reading, checkout/materialization, and repository discovery. They should operate on Glyph objects and policies, not Git objects.

## Prototype Storage

The first prototype can use a simple local directory store:

```text
.glyph/
  store.db
  manifest.yaml
  objects/
  content/
  policies/
  work/
  realms/
  publications/
  audit/
```

This directory is not Git. It is Glyph's initial local source-control database. SQLite is the canonical local index and graph store; generated filesystem projections and Git exports are disposable materializations.

The exact layout may change, but the prototype should keep canonical state separate from materialized workspaces.

## Success Criteria

This spec is successful if a prototype can:

- Store file content separately from visibility policy.
- Create source, realm, work context, snapshot, and publication objects.
- Create external mount objects for linked repositories.
- Regenerate a public projection from stored objects.
- Record a policy change.
- Record an agent edit as provenance.
- Export public state without reading hidden objects directly.
