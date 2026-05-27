# Spec 010: External Mount Lifecycle

Status: Draft
Date: 2026-05-27
Project: Glyph

## Thesis

External source mounts should preserve the useful part of Git submodules without inheriting their worst workflow failures.

A mount is a first-class relationship between source graphs or repositories, not an accidental directory full of files.

## Problem

Projects often need multiple repositories:

- A public package and private package in one workspace
- A vendored dependency with its own upstream
- A docs or examples repo mounted into a parent project
- Separate GitHub repositories for distribution, compliance, or community reasons

Git submodules and subtrees solve pieces of this but make update, review, and publication semantics confusing. Glyph needs explicit mount lifecycle rules.

## Goals

- Define mount creation, update, review, and removal.
- Preserve separate upstream identity.
- Support pinned mounts first.
- Support vendored Git export by default.
- Leave room for submodule export later.
- Prevent parent policies from accidentally leaking child source.

## Non-Goals

- Reimplementing every Git submodule feature.
- Automatically merging mounted repositories.
- Solving dependency package management.
- Supporting nested mounts in the first prototype.

## Mount Object

A mount records:

- Mount ID
- Parent source graph
- Mount path
- Source type
- Remote origin
- Pinned revision or source graph reference
- Export mode
- Import/update policy
- Allowed realms
- Local write policy
- Last sync event

The mount path belongs to the parent graph. The mounted content keeps its own upstream identity.

## Mount Modes

- `pinned`: parent records a specific upstream revision.
- `tracking`: parent can propose updates from upstream.
- `vendored-export`: exported Git repository includes mounted files.
- `submodule-export`: exported Git repository includes a Git submodule pointer.

The first prototype supports `pinned` plus `vendored-export`.

## Lifecycle

### Add

Adding a mount creates a mount object and a publication request if the mount will appear in a public realm.

### Update

Updating a mount creates a work context or publication request that changes the pinned revision. Updates are reviewable parent-project work.

### Local Edits

Local edits inside a mounted path are not ordinary parent edits.

Policy decides whether they are:

- Rejected
- Captured as patches against the mount
- Routed to a child work context
- Proposed upstream to the mounted repository

The v1 default is to reject unmanaged local edits inside mounts.

### Remove

Removing a mount records a parent graph change. It does not delete the upstream repository or child source graph.

## Policy Rules

Mount policy controls:

- Which realms can see the mount
- Whether mounted files appear in public export
- Whether local edits are allowed
- Whether upstream updates can be imported automatically
- Whether submodule export is allowed
- Whether child private files can ever appear in parent projections

Parent policy cannot widen child visibility unless the child source or remote permits it.

## Prototype Defaults

- Mounts are pinned.
- Vendored export is default.
- Submodule export is deferred.
- Nested mounts are unsupported.
- Unmanaged local edits inside mounts are rejected.
- Mount updates require review before publication.
- Parent publication checks include mounted content scanning.

## Success Criteria

This spec is successful if a prototype can:

- Add a GitHub repository as a mounted subdirectory.
- Record the pinned revision.
- Materialize the mount in a workspace.
- Export the public projection with vendored mounted files.
- Propose a mount update as reviewable work.
- Prevent unmanaged edits inside mounted paths.
