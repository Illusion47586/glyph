---
title: "Spec 002: Work Graph Without Commits Or Branches"
description: "Glyph design specification."
---

Status: Draft
Date: 2026-05-27
Project: Glyph

## Thesis

Commits and branches are bad native primitives for agent-era source control.

Glyph should track work continuously as it happens, then let humans and agents shape, explain, review, and publish that work later. The native primitive is not a commit or a branch. It is a **work graph**: a structured record of edits, snapshots, intentions, dependencies, reviews, and publication events.

## Problem

Git makes developers think about history while they are still thinking about code.

The result:

- Work is lost unless explicitly committed.
- People manufacture commits for the machine instead of for understanding.
- Branches become overloaded as isolation, naming, review, deployment, and collaboration units.
- Agents inherit human Git rituals that do not match how they work.
- Parallel work requires worktrees, stashes, rebases, and other accidental complexity.

Glyph should let users edit code first. Tracking should be automatic. History shaping should be optional and later.

## Goals

- Capture source changes continuously without requiring explicit commits.
- Replace branches with named work contexts and graph relationships.
- Represent human and agent work uniformly.
- Allow multiple simultaneous work contexts over the same source graph.
- Preserve intent, provenance, review, and publication metadata.
- Support later export to Git commits without making Git commits canonical.

## Non-Goals

- Defining the full storage engine.
- Designing every CLI command.
- Eliminating meaningful history. Glyph should improve history, not erase it.
- Making every keystroke a permanent public event.
- Perfectly reproducing Git branch semantics.

## Core Concepts

### Work Graph

The work graph records active and historical work over the source graph.

It contains:

- Work contexts
- Snapshots
- Edits
- Intent records
- Dependencies
- Review events
- Publication events
- Provenance links

The work graph is not a linear commit history. It is a graph of evolving work.

### Work Context

A work context is a scoped unit of active work.

Examples:

- `auth-fix`
- `security/openssl-advisory`
- `agent/refactor-router`
- `docs/bootstrap`

A work context has:

- Identity or owner
- Visible realms
- Base projection
- Current working state
- Captured snapshots
- Intent and task metadata
- Review and publication state

Work contexts replace branches as the primary unit of parallel work.

Work contexts may depend on other work contexts without merging. Dependency edges allow Glyph to represent stacked work, related agent tasks, or prerequisite changes while keeping each context independently reviewable and publishable.

### Workspace Integration

Workspace integration is the act of folding a completed work context into another realm or context.

Integration supports two history modes:

- `preserve`: publish the workspace's internal checkpoints and relevant snapshots as reviewable history.
- `squash`: publish the final source state as one coherent publication event while retaining detailed workspace history privately according to policy.

Squashing changes the visible shape of history. It must not silently destroy captured work unless retention policy explicitly allows purging.

Preserved integration is useful for audit-heavy, security-sensitive, collaborative, or long-running work. Squashed integration is useful for exploratory agent work, generated code, noisy refactors, and changes where the final state matters more than every intermediate step.

### Snapshot

A snapshot is a captured state of a work context.

Glyph may create snapshots:

- On file write
- Before and after commands
- Before tool or agent actions
- Before publication
- At user request
- On idle or checkpoint intervals

Snapshots are cheap and automatic. They are not necessarily user-facing commits.

Snapshots should be both content-addressed and time-addressed. Content addresses provide integrity and deduplication; timestamps and event IDs provide human timeline navigation and auditability.

The first prototype should capture edits at Glyph-mediated boundaries, not at keystroke granularity:

- Before an agent write
- After an agent write
- Before command execution
- After command execution if files changed
- Before publication requests
- At explicit human or agent snapshot requests

Agents may request snapshots for milestones, but Glyph must not rely on agents remembering to do so. For filesystem-backed workspaces, v1 may scan and hash files before and after agent tool calls or commands.

Glyph should expose an explicit milestone command named `glyph checkpoint`. A checkpoint creates a user-facing named snapshot for history shaping, review, or recovery, but it is not required for safety because automatic capture still happens at mediated boundaries.

### Edit

An edit is a source-level change observed by Glyph.

Edits can be produced by:

- Human filesystem writes
- Agent API writes
- Refactoring tools
- Generated code
- Import translators

Glyph should preserve enough edit structure to explain what changed without requiring users to pre-plan commits.

### Intent

Intent is metadata describing why work exists.

Examples:

- "Fix auth token refresh race"
- "Generate bindings for new API"
- "Patch embargoed vulnerability"
- "Explore replacing parser"

Intent can come from humans, agents, issue links, prompts, or explicit commands.

Agent prompts and tool logs should live in the audit graph. Work records should store policy-checked references and summaries, such as task summary, agent identity, transcript IDs, tool log IDs, files read, files written, and commands run.

### Publication

Publication is the act of moving selected work into another realm, usually from a private or active context into `public`.

Publication may produce:

- Glyph publication event
- Public source graph update
- Review artifact
- Git commit export
- GitHub pull request or push

Publication is not the same as committing.

### Pruning

Pruning removes inactive workspace projections and other disposable workspace material after work is finished.

Pruning is separate from deleting history. A pruned workspace should no longer occupy active workspace storage, but its source graph objects, snapshots, publication records, and audit events remain governed by retention policy.

Pruning may happen after:

- Successful publication
- Explicit discard
- Supersession by another work context
- Expiration of an inactive workspace
- Policy-driven cleanup

The default prune behavior should be conservative:

- Keep published history and audit records.
- Keep discarded work for the configured retention window.
- Remove materialized filesystem projections that can be regenerated.
- Preserve enough metadata to explain what happened and recover when policy allows it.

## Required Invariants

1. **Work is captured by default**
   Glyph should not rely on users remembering to save history manually.

2. **History shaping is separate from capture**
   Users can organize, squash, explain, or discard work after it has been captured.

3. **Branches are not required for parallel work**
   Multiple work contexts can exist over the same source graph without Git branches or worktrees.

4. **Publication is explicit**
   Captured work does not become public merely because it exists.

5. **Intent is first-class**
   Work records should preserve why work happened, not only what bytes changed.

6. **Agents are accountable**
   Agent-produced work records include agent identity, allowed context, and tool provenance.

7. **Git commits are export artifacts**
   Git commits may be generated from publication events but are not Glyph's native work unit.

## Work Context Lifecycle

1. Create a work context from a realm projection.
2. Edit through filesystem projection, API, or agent tools.
3. Glyph captures edits and snapshots automatically.
4. User or agent annotates intent where needed.
5. Work is reviewed, split, combined, discarded, or published.
6. Publication integrates the work into the destination realm using a preserve or squash history mode.
7. Publication creates an audit record and updates the destination realm.
8. Completed or discarded workspace projections may be pruned according to policy.
9. Translators may export the publication to Git or GitHub.

Discarded work should be retained by policy. The default retention window should be finite, such as 30 days for normal discarded work, with longer retention for audit/security contexts and immediate purge only when policy explicitly requires it.

## Relationship To Realms

Work contexts exist inside visibility boundaries.

A work context may:

- Start from one or more realm projections.
- Contain private work not visible to `public`.
- Depend on another work context without merging into it.
- Request publication into another realm.
- Be visible only to selected users or agents.

Glyph policies decide who can read, write, review, and publish work contexts.

## Prototype Defaults

- A work context is a named directory-like projection plus metadata.
- Snapshots are captured on explicit Glyph API writes and before publication.
- Agent writes, command execution, and publication requests are automatic snapshot boundaries.
- Filesystem writes can be captured by scanning in the first prototype.
- Work context IDs are stable.
- Git export maps one publication event to one Git commit by default.
- Branch import from Git is represented as provenance, not native Glyph branches.
- Publication defaults to `squash` for noisy agent work unless policy or user flags request `preserve`.
- Pruning removes active workspace projections but retains history according to policy.

## Success Criteria

This spec is successful if a prototype can:

- Create two simultaneous work contexts over the same source graph.
- Capture file changes without explicit commits.
- Record snapshots before publication.
- Attach intent and provenance to work.
- Publish selected work into `public`.
- Publish work with preserved or squashed history.
- Prune completed workspace projections without losing retained history.
- Export a publication as a Git commit without treating the commit as canonical.
