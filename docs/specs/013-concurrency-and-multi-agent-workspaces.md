# Spec 013: Concurrency And Multi-Agent Workspaces

Status: Draft
Date: 2026-05-27
Project: Glyph

## Thesis

Glyph should assume many humans and agents can work on the same project at the same time.

Concurrency should not be modeled as branches fighting for control of a filesystem. It should be modeled as isolated work contexts over a shared source graph, coordinated through leases, snapshots, dependency edges, conflict detection, review, and explicit publication.

## Problem

Agent-native development makes parallel work normal:

- Several agents may edit different parts of the project at once.
- A human may review one workspace while another agent continues work elsewhere.
- Two agents may accidentally attempt the same task.
- Long-running work may become stale after the destination realm changes.
- A crashed agent may leave a workspace claimed but inactive.
- Publication order can matter when changes depend on each other.

Git handles this through branches, worktrees, rebases, locks, and merge rituals. Glyph needs a clearer native model.

## Goals

- Allow many active work contexts over the same base realm.
- Avoid global editing locks for ordinary source changes.
- Prevent accidental concurrent mutation of the same work context.
- Detect stale, dependent, superseded, and conflicting work.
- Make agent activity observable without requiring trust in the agent.
- Keep publication explicit and policy-checked.
- Preserve enough history to explain concurrent decisions later.

## Non-Goals

- Building real-time collaborative editing in the first prototype.
- Silently resolving all conflicts.
- Requiring a central hosted coordinator for local concurrency.
- Making every active workspace visible to every user or agent.
- Treating Git branches as the concurrency primitive.

## Core Concepts

### Work Context Isolation

Each human or agent task should normally use its own work context.

A work context has:

- A stable ID and name
- Owner identity
- Optional agent identity
- Base realm and base snapshot
- Writable overlay
- Materialized workspace path, if any
- Claim or lease state
- Snapshot history
- Publication requests

Two agents may start from the same realm without blocking each other. Their work remains isolated until they declare dependency, request review, or attempt publication.

### Claim

A claim records which actor is actively driving a work context.

Fields:

- Work context ID
- Actor identity
- Provider
- Session ID
- Claimed at
- Last heartbeat
- Expiration time
- Claim mode

Claim modes:

- `exclusive`: one actor is expected to write the context.
- `shared-read`: other actors may inspect but not write.
- `handoff`: a new actor is taking over an inactive or explicitly transferred context.

Claims are coordination signals, not ownership of history. A stale claim does not delete work.

### Heartbeat

A heartbeat updates liveness for a claimed work context.

Agents should heartbeat during long tasks. Missing heartbeats mark the claim as stale after a policy-defined timeout. Stale claims may be taken over through explicit handoff.

### Base Snapshot

Every work context records the realm state it started from.

Publication compares the work context against:

- Its base snapshot
- The current destination realm
- Other active or pending publication requests when policy requires it

This allows Glyph to distinguish clean parallel work from stale or conflicting work.

### Dependency Edge

A dependency edge says one work context should be evaluated after another.

Dependencies can model:

- Stacked work
- Agent subtasks
- A refactor that must land before a feature
- A security fix that must precede public cleanup

Dependencies are not merges. They affect review and publication ordering.

### Conflict

A conflict is an incompatible concurrent change.

Conflict types include:

- Same source object changed differently
- Path rename/delete/edit collision
- Generated artifact divergence
- Policy object conflict
- Realm visibility conflict
- External mount conflict
- Remote publication conflict

Identical content changes are not conflicts, though they may be deduplicated or marked as duplicate work.

## Workspace States

Concurrent work should use clear states:

- `active`: work is being edited or observed.
- `idle`: no recent heartbeat, but not stale.
- `stale`: base realm or claim liveness is outdated.
- `ready`: work has a publication request or review-ready checkpoint.
- `blocked`: policy, dependency, check, or conflict prevents publication.
- `conflicted`: incompatible concurrent change detected.
- `superseded`: another work context made this one obsolete.
- `published`: work was integrated into a destination realm.
- `discarded`: work was intentionally abandoned but retained by policy.
- `pruned`: active projection was removed; retained history remains.

## Concurrency Model

Glyph should use optimistic concurrency for ordinary source edits.

Rules:

1. Active work contexts do not block each other by default.
2. SQLite serializes local store writes.
3. Materialized workspaces are isolated per work context.
4. Writes to the same work context require a valid claim unless policy allows shared editing.
5. Publication checks whether the destination realm changed since the base snapshot.
6. If the destination changed, Glyph computes whether the work is clean, stale, dependent, superseded, or conflicted.
7. Conflicted publication is blocked until explicitly resolved.
8. Stale but non-conflicting work may be rebased by projection, published with a dependency, or reviewed as-is according to policy.

## Agent Coordination

Agents should identify themselves with provider-scoped identities such as:

- `agent:codex:session-01`
- `agent:claude-code:session-02`
- `agent:cursor:session-03`

Agent activity should record:

- Work context claimed
- Files read
- Files written
- Commands requested
- Snapshots created
- Heartbeats
- Policy denials
- Publication requests

Agents should not be trusted to remember every snapshot or heartbeat. Glyph-mediated writes and command runs remain automatic capture boundaries.

## Conflict Detection

The first prototype should use conservative detection:

1. Compare changed path and source object sets against the destination realm delta since base.
2. If both sides changed the same source object to different content hashes, mark a content conflict.
3. If one side deleted or renamed a path and the other edited it, mark a path conflict.
4. If policy or visibility labels changed in incompatible ways, mark a policy or visibility conflict.
5. If changes are identical, mark duplicate/no-op rather than conflicted.
6. If changes touch disjoint paths and no policy relationship is affected, mark clean.

Future versions may add semantic merge, AST-level conflict detection, and generated artifact reconciliation.

## CLI Surface

Prototype commands:

```sh
glyph work claim auth-fix --actor agent:codex:session-01 --mode exclusive --json
glyph work heartbeat auth-fix --json
glyph work release auth-fix --json
glyph work conflicts auth-fix --json
glyph work depend feature-x refactor-y --json
glyph work stale auth-fix --json
glyph work prune --stale 30d --json
```

Publication should expose concurrency results:

```sh
glyph publish auth-fix --to public --mode squash --json
```

If publication is blocked, the JSON error should include structured conflict or dependency details when possible.

## Store Requirements

The local store should record:

- Work context base snapshot
- Claim records
- Heartbeat timestamps
- Dependency edges
- Conflict records
- Publication ordering decisions
- Supersession links
- Prune events

Store writes should be transactional. Commands that update work state should leave either a complete state transition or no transition.

SQLite should use settings appropriate for local concurrency, such as a busy timeout and write-ahead logging. This supports multiple CLI invocations without pretending SQLite is a distributed coordinator.

## Policy Requirements

Policy decides:

- Who can claim a work context
- Whether claims are exclusive or shared
- Claim timeout duration
- Who can take over stale work
- Whether stale work may publish
- Whether conflicts can be resolved by agents
- Which work contexts are visible to which actors
- Retention and pruning rules for stale, discarded, superseded, and published work

## Prototype Defaults

- One writer claim per work context.
- Claims default to exclusive.
- Claim timeout defaults to 15 minutes without heartbeat for agent sessions.
- Humans can override stale agent claims in local bootstrap mode.
- Publication uses optimistic concurrency checks.
- Conflicts block publication.
- Disjoint path changes can publish independently.
- Dependency edges affect publication ordering but do not merge work.
- Pruning stale projections is explicit in v1.

## Success Criteria

This spec is successful if a prototype can:

- Create two active work contexts from the same realm.
- Claim each work context for a different agent identity.
- Record heartbeats and mark inactive claims stale.
- Detect disjoint changes as clean parallel work.
- Detect same-file divergent changes as conflicts.
- Block publication on unresolved conflicts.
- Represent dependency ordering between two work contexts.
- Publish clean work without blocking unrelated active work.
- Prune stale or completed workspace projections without deleting retained history.
