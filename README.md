# Glyph

Glyph is an agent-native source control system built around permissioned source graphs, continuous work capture, and explicit publication.

It is not Git with a nicer wrapper. Glyph treats Git and GitHub as compatibility targets while keeping its own model centered on:

- realms as permissioned source views
- work contexts instead of branches
- snapshots and checkpoints instead of mandatory commits
- publications instead of implicit pushes
- agent-friendly JSON CLI operations
- local SQLite-backed source-control state in `.glyph/`

## Status

Glyph is in bootstrap/prototype mode.

This repository is being dogfooded with Glyph itself. The local `.glyph/` directory is the canonical source-control store, and GitHub is currently used as an export-only public mirror.

## Current Prototype

The Go CLI currently supports:

- `glyph init`
- `glyph import`
- `glyph status --json`
- work contexts
- read/write/project/diff operations
- checkpoints and publications
- local hook execution
- concurrency claims and conflicts
- Git export and export-only GitHub remotes
- generated `.gitignore` / `.gitinclude`
- a local mesh visualizer

## Try It Locally

```sh
go test ./...
go run ./cmd/glyph status --json
go run ./cmd/glyph viz export --json
```

The visualizer writes:

```text
.glyph/visualizer/index.html
.glyph/visualizer/graph.json
```

## Design Docs

The design is being developed as numbered specs in [docs/specs](docs/specs).

Start with:

- [Spec 000: Bootstrap And Dogfooding Protocol](docs/specs/000-bootstrap-and-dogfooding-protocol.md)
- [Spec 001: Realms As Permissioned Source Views](docs/specs/001-realms-permissioned-source-views.md)
- [Spec 002: Work Graph Without Commits Or Branches](docs/specs/002-work-graph-without-commits-or-branches.md)
- [Spec 006: Git Import/Export Compatibility](docs/specs/006-git-import-export-compatibility.md)
- [Spec 013: Concurrency And Multi-Agent Workspaces](docs/specs/013-concurrency-and-multi-agent-workspaces.md)

## License

License is not finalized yet.
