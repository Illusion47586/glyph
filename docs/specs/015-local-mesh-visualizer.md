---
title: "Spec 015: Local Mesh Visualizer"
description: "Glyph design specification."
---

Status: Draft
Date: 2026-05-27
Project: Glyph

## Thesis

Glyph's source graph should be visible.

Humans and agents need a way to inspect the mesh of work contexts, snapshots, publications, source objects, content objects, claims, conflicts, hooks, remotes, and audit events without spelunking through SQLite tables by hand.

## Goals

- Provide a local visualizer for `.glyph/` state.
- Export machine-readable graph data.
- Render a static HTML view that requires no hosted service.
- Show relationships between work, source, publication, hooks, and concurrency state.
- Keep private/local visibility explicit; the visualizer is a local diagnostic tool in v0.

## Non-Goals

- Building a hosted dashboard.
- Real-time collaborative graph streaming.
- Replacing CLI status/diff commands.
- Exposing private `.glyph` data publicly.

## Graph Model

The visualizer exports nodes and edges.

Initial node types:

- `store`
- `realm`
- `work`
- `snapshot`
- `publication`
- `source`
- `content`
- `claim`
- `conflict`
- `hook_run`
- `remote`
- `mount`

Initial edge types:

- `contains`
- `belongs_to`
- `based_on`
- `points_to`
- `captured`
- `published`
- `claimed_by`
- `conflicts_with`
- `ran_hook`
- `syncs_to`
- `mounted_at`

Nodes should include stable IDs, labels, type, and detail maps. Edges should include source, target, type, and optional detail maps.

## CLI Surface

Prototype command:

```sh
glyph viz export --out .glyph/visualizer --json
```

The command writes:

```text
.glyph/visualizer/
  index.html
  graph.json
```

`graph.json` is the canonical export artifact. `index.html` is a local viewer over that artifact.

## UI Requirements

The first visualizer should be simple but useful:

- Filter by node type.
- Search by ID, label, path, actor, or realm.
- Show graph nodes and edges.
- Show selected node details as JSON.
- Show graph summary counts.

The v0 renderer can use plain HTML, CSS, and browser JavaScript. It should not require a server or external network dependencies.

## Security Requirements

Visualizer exports may reveal private source paths, actor identities, hooks, and audit metadata.

Default behavior:

- Export from local `.glyph` only.
- Write output under a local path.
- Do not publish or sync visualizer output unless explicitly requested.
- Treat generated `graph.json` as sensitive unless generated from a public realm projection in a future scoped export mode.

## Prototype Defaults

- Static export only.
- All local store metadata visible in the generated graph.
- No external JavaScript dependencies.
- No live updates.
- Graph data includes metadata, not raw file contents.

## Success Criteria

This spec is successful if a prototype can:

- Export `.glyph` graph data to `graph.json`.
- Generate a static `index.html` viewer.
- Include work contexts, snapshots, publications, sources, content, claims, conflicts, hooks, remotes, and mounts.
- Open locally without a server.
- Let agents consume the same `graph.json`.
