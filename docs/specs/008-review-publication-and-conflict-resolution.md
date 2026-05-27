# Spec 008: Review, Publication, And Conflict Resolution

Status: Draft
Date: 2026-05-27
Project: Glyph

## Thesis

Review and publication are the bridge between private work and visible source.

Glyph must make that bridge explicit: what is being proposed, who reviewed it, what checks passed, what conflicts exist, and exactly which realms become able to see the result.

## Problem

Git collapses several different ideas into commits, branches, pull requests, and merges. Glyph separates them into work contexts, publication requests, policy checks, and projection exports. That separation needs a clear review model.

Without one, Glyph cannot safely answer:

- Can this agent change become public?
- Did the required people approve it?
- Did CI pass?
- Did two work contexts edit the same source object?
- Does an imported GitHub pull request become canonical?
- What happens when publication is revoked or superseded?

## Goals

- Define review and publication request states.
- Define how approvals and checks attach to publication.
- Define conflict detection between work contexts and imported proposals.
- Support partial publication.
- Support preserved and squashed workspace integration.
- Support pruning completed workspace projections without erasing retained history.
- Support supersession and revocation without rewriting visibility history.
- Give GitHub pull requests a clear mapping into Glyph.

## Non-Goals

- Designing a hosted review UI.
- Recreating Git merge semantics exactly.
- Solving every semantic merge conflict.
- Defining the full CI integration protocol.

## Core Concepts

### Publication Request

A publication request proposes moving selected work from one realm or work context into another realm.

Fields:

- Request ID
- Source work context or realm
- Destination realm
- Included source objects
- Excluded source objects
- Actor
- Required reviewers
- Approvals
- External check results
- Conflict status
- Policy decision
- Final publication object, if accepted

### Review State

Publication requests use these states:

- `draft`: still being prepared
- `ready`: available for review
- `blocked`: policy, conflict, or check failure
- `approved`: required reviewers approved
- `published`: destination realm updated
- `rejected`: intentionally declined
- `superseded`: replaced by a newer request
- `revoked`: future access withdrawn after publication

Publication history remains append-only. `revoked` and `superseded` never erase the earlier visibility event.

### Approval

An approval is an identity-bound review event.

Approvals record:

- Reviewer identity
- Provider
- Timestamp
- Scope of approval
- Policy version
- Optional comment

Approvals should be invalidated or rechecked when the publication request materially changes.

### Conflict

A conflict exists when two work contexts or imported proposals make incompatible changes to the same source object, path, generated artifact, policy object, or publication target.

Conflicts can be:

- Content conflicts
- Path conflicts
- Visibility conflicts
- Policy conflicts
- Mount conflicts
- External remote conflicts

Glyph should distinguish conflicts from dependencies. A work context may depend on another without being in conflict.

### Integration Mode

A publication request declares how the source work should appear in the destination realm's visible history.

Supported modes:

- `preserve`: carry selected checkpoints and relevant snapshots into the destination realm as visible history.
- `squash`: publish the final selected source state as one publication event.

Both modes must run the same policy checks, conflict checks, and visibility checks. The mode changes history presentation, not safety requirements.

Squashed publication must still retain private provenance according to policy. A reviewer with sufficient permission should be able to inspect the detailed work graph even if the public realm sees one clean event.

## Conflict Resolution

The first prototype should use conservative, explicit conflict handling:

1. Detect changed source objects and paths.
2. If two contexts changed the same object, mark a conflict unless the change is identical.
3. If a path was renamed or deleted in one context and edited in another, mark a path conflict.
4. If publication would widen visibility differently than a dependency expects, mark a visibility conflict.
5. Require a human or policy-authorized actor to resolve conflicts.

Automatic textual merge can be offered later, but v1 should not silently merge conflicting work into public.

## Partial Publication

A publication request may include only part of a work context.

Partial publication must:

- Identify included and excluded source objects.
- Check that excluded private objects are not referenced by included public objects.
- Re-run generated artifact and secret checks.
- Record the partial selection in the publication audit.

Partial publication can be combined with either integration mode. A squashed partial publication exposes only the selected final state. A preserved partial publication exposes selected history only for included source objects.

## Workspace Pruning

After a work context is published, rejected, superseded, or discarded, Glyph may prune its active workspace projection.

Pruning may remove:

- Materialized filesystem directories
- Temporary indexes
- Cached diffs
- Regenerable projection metadata

Pruning must not remove:

- Published source graph objects
- Required audit events
- Retained snapshots
- Publication request records
- Policy decisions
- Approval records

The user-facing operation should be explicit:

```sh
glyph work prune auth-fix
glyph work prune --published
glyph work prune --inactive 30d
```

Automatic pruning can be configured later, but the first prototype should prefer explicit pruning so users trust the lifecycle.

## GitHub Mapping

Imported GitHub pull requests become Glyph publication requests by default.

GitHub comments, commits, and check runs become external provenance attached to the request. They do not become canonical review state until adopted by Glyph policy.

Exported Glyph publication requests may create GitHub pull requests as public review artifacts, but Glyph remains canonical for Glyph-native projects.

## Prototype Defaults

- Publication requests are required for moving work into `public`.
- Review state is stored in Glyph, not GitHub.
- Required approvals are policy-driven.
- GitHub Actions checks attach as external check results.
- Conflicts block publication until explicitly resolved.
- Partial publication is allowed only at source-object granularity in v1.
- Publication supports `squash` and `preserve` history modes.
- Pruning is explicit in v1 and removes projections, not retained history.
- Revocation and supersession append new events.

## Success Criteria

This spec is successful if a prototype can:

- Create a publication request from a work context.
- Attach approvals and external check results.
- Detect conflicting edits between two work contexts.
- Block publication on unresolved conflicts.
- Publish an approved request into `public`.
- Publish with squashed or preserved history.
- Prune a completed workspace projection while retaining required history.
- Import a GitHub pull request as a Glyph publication request.
- Supersede or revoke a publication without deleting history.
