---
title: "Spec 001: Realms as Permissioned Source Views"
description: "Glyph design specification."
---

Status: Draft
Date: 2026-05-27
Project: Glyph

## Thesis

Open source should not mean every byte of a project's source graph is public at every moment.

Glyph treats openness as a publication policy, not a repository topology. A project can have one underlying source graph and many controlled views over it: public source, maintainer work, embargoed security fixes, private monorepo packages, deployment configuration, agent sandboxes, and customer-specific code.

The first primitive is the **realm**: a named, permissioned projection over a canonical source graph.

## Problem

Git equates collaboration boundaries with repository boundaries. This creates persistent pressure to split source into multiple repos, hide work in private forks, pass secrets through side channels, and delay important fixes because the only publication mode is effectively "visible to anyone who can see this repo."

This model is especially bad for agent-native workflows:

- Agents need precise context windows, not blanket repository access.
- Security fixes often need embargoed collaboration before publication.
- Open source maintainers need private in-flight work without losing public transparency for released work.
- Monorepos need package-level visibility without fragmenting history.
- Secrets and environment-specific files should be impossible to leak through ordinary source publication paths.

Glyph's answer is to make visibility a first-class source-control concept.

## Goals

- Allow one source graph to produce multiple permissioned views.
- Make public/private/embargoed visibility explicit and reviewable.
- Support open source projects that contain private work without splitting repositories.
- Support monorepos with mixed public and private subpackages.
- Give agents least-privilege source access through scoped projections.
- Prevent accidental leakage across visibility boundaries.
- Preserve enough structure for future Git export and migration.

## Non-Goals

- Replacing commits, branches, and staging. That belongs in a later work-graph spec.
- Defining the full distributed sync protocol.
- Designing the complete CLI.
- Building hosted organization billing, UI, or admin flows.
- Solving runtime secret management. Glyph can prevent source publication leaks, but it is not a secret manager.

## Core Concepts

### Source Graph

The source graph is Glyph's canonical store of project objects. It contains file contents, directory structure, change objects, identities, metadata, policies, and visibility labels.

The source graph is not directly equivalent to a working tree or Git repository. It is the authoritative object space from which projections are derived.

### Realm

A realm is a named visibility policy over the source graph.

A realm is always a named policy. Ad hoc capability tokens may exist later as grants, invitations, or access links, but they are not realms.

Examples:

- `public`: source visible to everyone
- `maintainers`: in-flight work visible to trusted maintainers
- `security`: embargoed fixes and vulnerability discussion
- `infra`: deployment config and operational code
- `agent/auth-fix`: scoped context for an agent working on auth
- `package/private-billing`: a private package inside an otherwise public monorepo

A realm does not have to duplicate source. It selects and transforms graph objects according to policy.

### Projection

A projection is the materialized view produced by evaluating a realm for an identity.

Projection targets may include:

- Filesystem checkout
- API file reads
- Code search
- Diff/review UI
- Agent context bundle
- Git-compatible export

Different identities may receive different projections from the same realm if policy requires it, but this should be used sparingly. A realm should usually be understandable as a stable named view.

### External Source Mount

An external source mount links another source root into a subdirectory of a Glyph project.

This is the Glyph equivalent of the useful part of Git submodules: one project can include another project at a path while allowing each project to keep its own remote origin, visibility policy, publication lifecycle, and GitHub repository.

Examples:

- `vendor/parser` is backed by a public GitHub repository.
- `packages/private-billing` is backed by a separate private GitHub repository.
- `examples/sdk` is backed by another Glyph source graph and exported to its own GitHub repo.

An external source mount should record:

- Mount path
- Source type, such as `git`, `github`, or `glyph`
- Remote origin
- Pinned revision or source graph reference
- Allowed realms
- Import policy
- Export policy
- Whether local changes are allowed

Mounts are not just copied directories. They preserve the fact that the mounted subtree has a separate upstream identity.

### Visibility Label

A visibility label is metadata attached to paths, source graph objects, or changes. Labels describe where source is allowed to appear and how work may be published.

Label scopes:

- Path labels define default policy for a subtree, which is useful for monorepos.
- Object labels attach durable policy to a specific source object and should survive moves.
- Change labels attach policy to work or publication events, which is useful for embargoes, generated artifacts, and review requirements.

Examples:

- `public`
- `maintainers`
- `security-embargo`
- `private-package:billing`
- `secret-never-publish`
- `agent-readable`

Labels are not sufficient by themselves. A realm policy decides how labels are interpreted. Conflict and precedence rules belong in the policy spec.

### Grant

A grant gives an identity or group access to a realm, source object class, action, or publication path.

Examples:

- Alice can read `security`.
- CI can read `infra` but cannot publish from it.
- An agent can read `public` plus files tagged `agent-readable` in `maintainers`.
- Maintainers can propose publication from `security` to `public`, but publication requires review.

### Publication

Publication moves source graph objects, changes, or derived projections from a more restricted realm into a less restricted realm.

Examples:

- Promote a security fix from `security` to `public`.
- Publish a private package interface while keeping implementation private.
- Reveal an in-flight change after review.
- Export the public realm to GitHub.

Publication is a policy-checked operation, not a push to a branch.

Publication history is append-only. Glyph may supersede a publication, revoke future access, or redact future projections, but it must not erase the fact that an object became visible to a realm.

### Redaction Boundary

A redaction boundary is an explicit rule that prevents objects from crossing into a realm or projection.

Examples:

- Files matching `.env*` cannot appear in `public`.
- Objects labeled `secret-never-publish` cannot be published anywhere.
- Security advisory discussion cannot be exported to public issue trackers before disclosure.
- Private package implementation files cannot appear in `public`, but generated type declarations can.

Redaction boundaries should fail closed. If Glyph cannot prove a projection is allowed, it should not produce it.

Secrets should be blocked by both policy and content scanning. Policy blocks known risky paths and labels, such as `.env*` or `secret-never-publish`; content scanning catches accidental credentials in otherwise publishable files. If either mechanism denies publication, Glyph must fail closed.

### Mixed-Visibility Files

The first prototype should reject mixed-visibility source files for publication.

If a file contains both public and private information, Glyph should not attempt line-level visibility. The project must instead create an explicit generated or redacted artifact, such as a public interface file, generated type declaration, scrubbed advisory, or extracted public module.

Line-level visibility may be explored later, but it is too risky for the initial model.

## Required Invariants

1. **No invisible publication**
   Any object newly visible in a less restricted realm must be attributable to an explicit publication event or policy change.

2. **No accidental widening**
   Renames, moves, merges, generated files, and agent edits must not silently widen visibility.

3. **Policy before materialization**
   A filesystem checkout, API response, search result, diff, or export must be filtered before it is materialized.

4. **Denied by default**
   Unlabeled new objects start in the current author's working realm, not in `public`.

5. **Auditable visibility history**
   Glyph must record when an object became visible to which realm, by what policy, and by whose authority.

6. **Least-privilege agents**
   An agent receives a projection, not raw graph access, unless explicitly granted.

7. **Git export cannot bypass policy**
   Any Git-compatible export is just another projection and must obey realm policy.

## Canonical Workflows

### Public Project With Private In-Flight Work

1. A maintainer edits code in a private maintainer realm.
2. Glyph captures the work in the source graph.
3. Public users continue to see only the public projection.
4. When ready, the maintainer publishes selected changes to `public`.
5. Glyph records the publication event and exports the public projection.

This replaces the assumption that open source work must happen in public branches or private forks.

### Embargoed Security Fix

1. A vulnerability is discovered and a `security` realm is created.
2. Trusted maintainers and security agents are granted access.
3. Fixes, tests, reproductions, and advisory drafts remain hidden from `public`.
4. Glyph verifies that publication contains the fix but not exploit notes, secrets, or private discussion.
5. On disclosure, selected objects are published to `public`.

This makes delayed publication a first-class source-control operation rather than a social workaround.

### Mixed-Visibility Monorepo

1. A monorepo contains public packages and private packages in one source graph.
2. `public` exposes public packages and stable interfaces.
3. `package/private-billing` exposes implementation only to authorized users.
4. Cross-package dependencies are represented in metadata so public projections do not contain broken references.
5. Publication can reveal generated API surfaces without revealing private implementation.

This avoids splitting source history merely to satisfy visibility constraints.

### Project With Linked GitHub Subdirectory

1. A Glyph project declares `vendor/parser` as an external source mount.
2. The mount points to a separate GitHub repository.
3. The parent project can materialize `vendor/parser` as a subdirectory.
4. The mounted project keeps its own remote origin and publication rules.
5. Parent-project publication can include the mount pointer, a vendored projection, or both depending on policy.

This supports projects that need multiple GitHub repositories without forcing all source into one Git repository.

### Agent-Scoped Workspace

1. A user asks an agent to fix a bug in auth.
2. Glyph creates `agent/auth-fix`, a projection containing only relevant allowed files, tests, and metadata.
3. The agent edits against that projection through file APIs or a mounted checkout.
4. Glyph captures the work in the source graph under the agent realm.
5. A human reviews and publishes selected changes.

The agent never needs blanket repository access.

## Policy Model

Glyph policy should be declarative, versioned, and reviewable.

A minimal policy needs to express:

- Realm primitive names and descriptions
- Groups
- Read grants
- Write grants
- Publication grants
- Label rules
- Path rules
- Generated artifact rules
- Export rules
- External source mount rules
- Required reviewers for widening visibility

The first policy language should be declarative YAML with no custom code execution. The v1 surface should be limited to `realms`, `groups`, `labels`, `paths`, `publish`, `redactions`, and `exports`.

Example policy sketch:

```yaml
realms:
  public:
    read: ["*"]
    publish_from:
      maintainers:
        reviewers: 1
      security:
        reviewers: 2
        require_labels: ["disclosure-approved"]

  maintainers:
    read: ["group:maintainers"]
    write: ["group:maintainers", "agent:*"]

  security:
    read: ["group:security"]
    write: ["group:security", "agent:security-fix"]

redactions:
  - match: ".env*"
    deny_realms: ["public"]
  - label: "secret-never-publish"
    deny_realms: ["*"]
  - label: "exploit-notes"
    deny_realms: ["public"]
```

This syntax is illustrative, not final.

## Publication Checks

Before any object becomes visible in a less restricted realm, Glyph should run publication checks:

- Does the actor have permission to publish this object?
- Does the destination realm allow this object's labels?
- Do path rules allow this object?
- Do generated artifacts contain redacted source content?
- Do diffs, code search indexes, previews, logs, and review comments leak hidden content?
- Are required reviewers satisfied?
- Is the publication event recorded?

Publication checks apply to files and metadata. A public diff can leak private filenames, comments, issue titles, stack traces, package names, or generated code. Glyph should treat metadata as source graph objects subject to policy.

## Filesystem And API Access

Glyph should not require a traditional full working tree to be useful.

The same projection model should support:

- Mounted filesystem views
- Copy-on-write local checkouts
- API-based file reads and writes
- Agent context bundles
- Browser-based editors
- CI projections

This keeps the source-control model independent from any single operating system abstraction. Filesystems become one projection target, not the foundation of the system.

## Relationship To Git

Glyph should be able to export a realm projection to Git, especially for public open source compatibility.

In the first version:

- Git is an export/import format, not the canonical model.
- `public` can map to a normal Git repository.
- Private realms do not need Git branches.
- Publication to `public` can produce Git commits for ecosystem compatibility.
- Git history should never contain objects that were not visible in the exported realm at export time.

The first usable prototype only requires export-only Git compatibility: generate a clean Git repository from the `public` realm, create commits from Glyph publication events, and optionally push to a configured GitHub remote. Git import can follow after the first export path works.

The exact mapping between Glyph work objects and Git commits belongs in a later spec.

## Security Considerations

The main risk is false confidence. A visibility system is worse than no system if users believe private objects are hidden while side channels leak them.

Important leakage surfaces:

- File contents
- File paths
- Directory names
- Diffs
- Deleted content
- Generated files
- Build logs
- Test snapshots
- Search indexes
- Review comments
- Issue links
- CI artifacts
- Agent prompts and transcripts
- Error reports
- Git exports

Glyph must treat all of these as possible projections of the source graph.

## UX Principles

- Visibility should be visible. Users should know which realm they are in.
- Publication should feel deliberate, not bureaucratic.
- Redaction failures should explain what blocked publication without exposing hidden content to unauthorized users.
- Agents should receive scoped context by default.
- Public users should experience normal open source workflows where possible.
- Maintainers should not have to split repos to express privacy.

## Prototype Defaults

The first prototype should make conservative choices:

- Realms are named policies, not ad hoc tokens.
- Labels can be attached to paths and graph objects.
- Labels can also be attached to changes.
- Visibility history is append-only.
- Mixed public/private files are rejected for publication unless a generated redacted artifact is explicitly declared.
- Secrets are blocked by both policy rules and content scanning.
- Agent transcripts are stored in an audit graph and referenced from the source graph only through policy-checked metadata.
- Git compatibility means exporting the `public` projection into a normal Git repository; importing arbitrary Git history is out of scope for the first prototype.
- External source mounts can link separate GitHub or Git repositories into subdirectories while preserving their distinct origins.

## Success Criteria

This spec is successful if it enables a prototype where:

- A project has one canonical source graph.
- At least two realms expose different file projections.
- Private files cannot appear in the public projection.
- A maintainer can publish selected work from a private realm to public.
- Publication produces an audit record.
- An agent can receive and modify a scoped projection.
- A public Git export can be produced without private objects.

## Later Specs

- Spec 002: Work Graph Without Commits Or Branches
- Spec 003: Agent-Native File API And Virtual Workspaces
- Spec 004: Storage Model And Object Graph
- Spec 005: Policy Language And Threat Model
- Spec 006: Git Import/Export Compatibility
- Spec 007: CLI And Agent API
