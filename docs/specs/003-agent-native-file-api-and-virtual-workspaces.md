# Spec 003: Agent-Native File API And Virtual Workspaces

Status: Draft
Date: 2026-05-27
Project: Glyph

## Thesis

Source control should not require a full operating system checkout as its primary interface.

Glyph should expose source through policy-aware file APIs and virtual workspaces. Filesystems remain useful, but they become projection targets rather than the foundation of the model.

## Problem

Git assumes a local repository, a working tree, and a real filesystem. Coding agents often need something else:

- Read a bounded set of files.
- Modify files through structured tool calls.
- Receive relevant context without seeing every secret or private package.
- Work in isolated contexts without worktrees.
- Run in sandboxes, browsers, remote workers, or constrained runtimes.

Glyph should support agents and tools that interact with source through APIs, not only shell commands over mounted files.

## Goals

- Provide a native file read/write API for agents.
- Support virtual workspaces backed by realm projections.
- Allow filesystem mounts or checkouts as optional interfaces.
- Enforce policy before reads, writes, search, diffs, and exports.
- Capture provenance for API and agent writes.
- Make workspaces cheap enough for many simultaneous agents.

## Non-Goals

- Requiring FUSE or kernel-level filesystem integration in the first prototype.
- Defining the complete network protocol.
- Replacing all local editor workflows.
- Building a full remote execution platform.
- Making agents trusted by default.

## Core Concepts

### File API

The File API is a policy-aware interface for reading and writing source graph projections.

Minimum operations:

- `read(path)`
- `write(path, content)`
- `list(path)`
- `stat(path)`
- `search(query)`
- `diff(base, target)`
- `snapshot()`
- `publish(request)`

Every operation is evaluated against realm policy and work context permissions.

### Virtual Workspace

A virtual workspace is an addressable source projection plus writable overlay.

It has:

- Workspace ID
- Backing realm projection
- Work context
- Read policy
- Write policy
- Captured changes
- Optional filesystem materialization

Virtual workspaces replace many uses of Git worktrees.

### Materialization

Materialization turns a projection into a concrete interface:

- Filesystem directory
- In-memory file map
- API response
- Agent context bundle
- Archive
- Git export

Materialization must happen after policy filtering.

### Writable Overlay

A writable overlay records changes made on top of a projection.

The overlay can be committed to the Glyph source graph as work, published, discarded, or exported. It does not require copying the whole repository.

## Agent Requirements

Agents should be able to:

- Request a workspace for a task.
- Read only allowed files.
- Ask for additional context with a reason.
- Modify files through structured writes.
- Run tests or commands through allowed execution hooks.
- Produce a reviewable change proposal.
- Publish only through policy-checked workflows.

Agents should not need to:

- Clone a repository.
- Manage branches.
- Stage files.
- Rebase.
- Discover secrets by accident.

The File API should be exposed through MCP by default for agent interoperability. MCP is an integration surface over Glyph's internal local API, not the internal architecture.

## Virtual Workspace Lifecycle

1. Create workspace from a realm projection and task intent.
2. Attach workspace to a work context.
3. Agent or human reads and writes through File API or materialized checkout.
4. Glyph records writes, snapshots, and provenance.
5. Workspace is reviewed, published, suspended, or discarded.
6. Workspace may be garbage-collected according to policy after retention.

## Policy Requirements

Policy applies to:

- Path reads
- Path writes
- Directory listings
- Search results
- Diffs
- Generated context bundles
- Test outputs and logs
- Publication requests

Denied content should not appear as empty files unless policy explicitly allows redacted placeholders. The default behavior should be invisibility plus explainable denial to authorized users.

Virtual workspaces should support streaming file reads so large repositories and large files do not require loading entire projections into memory.

Commands inside virtual workspaces must run through a Glyph-mediated command runner. The runner receives only the workspace projection, records before/after snapshots, captures logs as audit objects, and filters outputs according to policy before exposing them to agents or users.

The minimal human editor integration is a materialized local directory plus status/diff commands. Deeper editor extensions can come later.

## Prototype Defaults

- The first File API is local and process-level.
- The File API is exposed to agents through an MCP wrapper by default.
- Virtual workspaces are stored as metadata plus overlay files.
- Virtual workspaces support streaming reads.
- Filesystem materialization is a generated directory, not a kernel mount.
- Search is limited to allowed projected files.
- Agents receive workspace IDs and use API operations.
- Commands run through a Glyph-mediated command runner.
- Denied files are invisible by default unless policy explicitly allows redacted placeholders.
- Human editor support starts with materialized directories.
- Permission denials are recorded in audit logs.

## Success Criteria

This spec is successful if a prototype can:

- Create a virtual workspace from `public`.
- Read and write files through an API.
- Capture writes in a work context.
- Materialize a workspace as a local directory.
- Prevent reads of files outside the workspace projection.
- Let an agent produce a publishable change without using Git.
