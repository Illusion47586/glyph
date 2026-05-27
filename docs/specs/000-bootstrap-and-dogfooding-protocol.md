# Spec 000: Bootstrap And Dogfooding Protocol

Status: Draft
Date: 2026-05-27
Project: Glyph

## Thesis

Glyph should be built under its own conceptual constraints as early as possible.

Until Glyph can track itself, this workspace is a **pre-Glyph workspace**: a plain filesystem directory containing canonical specs and design artifacts. It must not be initialized as a Git repository merely for convenience. Glyph's first meaningful act of self-hosting should be to import this workspace as its genesis source graph.

## Problem

It is easy to design a replacement for Git while quietly depending on Git-shaped habits:

- Commits as the default unit of thought
- Branches as the default unit of parallel work
- Worktrees as the default unit of isolation
- Repository boundaries as the default unit of visibility
- Push/pull as the default publication model
- Git history as the default project memory

If Glyph is developed inside Git from day one, the project risks inheriting the assumptions it exists to challenge.

The bootstrap protocol defines how Glyph is specified, designed, and eventually self-hosted before Glyph itself exists.

## Goals

- Avoid initializing Git for the Glyph workspace.
- Treat specs as canonical source artifacts during the bootstrap phase.
- Preserve enough project memory to import into Glyph later.
- Make design decisions explicit without requiring commits.
- Define when Glyph is mature enough to track itself.
- Keep the process lightweight enough that it does not become ceremonial.

## Non-Goals

- Replacing version control manually with a pile of scripts.
- Creating a perfect audit log before Glyph exists.
- Designing the final Glyph storage engine.
- Defining every CLI command.
- Supporting multiple collaborators before the first prototype.

## Bootstrap Rule

Until Glyph reaches self-hosting readiness:

1. Do not run `git init` in this workspace.
2. Do not create a Git repository elsewhere as the canonical Glyph project history.
3. Treat files under `docs/specs/` as canonical project source.
4. Treat working notes outside `docs/specs/` as non-canonical unless promoted into a spec.
5. Prefer append-only or clearly superseding edits for major design changes.
6. Record important design changes in spec status sections or decision records.
7. When Glyph can import a directory, this workspace becomes the genesis source graph.

This is not anti-Git theater. It is an intentional pressure test: if Glyph cannot support the workflow needed to build Glyph, the design is not yet good enough.

## Canonical Artifacts

### Specs

Specs live in `docs/specs/` and are numbered.

Numbering:

- `000-*`: bootstrap, process, and project constitution
- `001-*` onward: product and system specs

Specs should include:

- Status
- Date
- Accepted-By, when status is `Accepted`
- Accepted-Date, when status is `Accepted`
- Thesis
- Problem
- Goals
- Non-goals
- Concepts or architecture
- Workflows where relevant
- Invariants where relevant
- Decisions or prototype defaults
- Prototype defaults where useful
- Success criteria

### Decision Records

Small decisions can live inside the relevant spec.

Large cross-cutting decisions should become separate specs or decision records under `docs/decisions/` once needed.

Examples:

- Naming and trademark direction
- Storage engine choice
- Policy language choice
- Git compatibility boundary
- Self-hosting readiness declaration

### Notes

Exploratory notes may live under `docs/notes/`.

Notes are not canonical unless referenced or promoted by a spec.

### Bootstrap Manifest

The bootstrap workspace should include a small `glyph.yaml` manifest.

The manifest is not a replacement for Glyph storage. It is a stable entrypoint for the future genesis importer.

It should record:

- Project name
- Spec directory
- Default public realm
- Bootstrap identity
- Genesis import include rules
- Genesis import exclude rules

### Prototype Code

Prototype code may be added before Glyph can track itself, but it is not the canonical project history. During bootstrap, code is an implementation artifact under active design control by the specs.

Once Glyph can import the workspace, prototype code becomes part of the genesis graph alongside the specs.

## Spec Status

Specs use simple statuses:

- `Draft`: useful but still changing
- `Accepted`: current design basis
- `Superseded`: replaced by a newer spec
- `Rejected`: intentionally not pursued

During bootstrap, a spec may be edited in place while it is `Draft`.

Once a spec is `Accepted`, major changes should either:

- Add a dated amendment section
- Create a superseding spec
- Create a decision record that updates interpretation

Accepted specs require lightweight human approval metadata:

```text
Status: Accepted
Accepted-By: user:self:dhruv
Accepted-Date: 2026-05-27
```

The exact identity spelling may evolve, but accepted specs should identify who accepted them and which identity provider asserted that identity.

## Identity Model During Bootstrap

All users and agents should have stable IDs and providers.

User providers may be:

- `self`: a self-asserted local identity
- An organization identity provider
- A hosted Glyph identity provider, if one exists later

Agent providers may be the agent or tool family:

- `codex`
- `claude-code`
- `cursor`
- Other coding-agent runtimes

Example identities:

```text
user:self:dhruv
user:org/acme:alice
agent:codex:session-01
agent:claude-code:session-02
```

Bootstrap specs may use these identifiers before the final identity system exists. The genesis import should preserve them as provenance metadata.

## Project Memory Before Glyph

Because there is no Git history, project memory must be carried in the documents themselves.

Each important spec should make these visible:

- What decision is being made
- Why it matters
- What alternatives were considered
- What remains unresolved
- What would prove the design wrong

The point is not to recreate commit history. The point is to make the source of decisions legible.

## Naming During Bootstrap

The working project and CLI name is **Glyph**.

`realm` is no longer the product name. It is a core primitive inside Glyph.

A realm is a permissioned projection over a source graph.

This avoids overloading product identity and system vocabulary.

## Genesis Source Graph

Glyph's first self-hosted import should create a genesis source graph from this workspace.

The genesis import should include:

- `docs/specs/`
- `docs/decisions/`, if present
- `docs/notes/`, if present
- Prototype source code, if present
- Tests, if present
- Build and package files, if present

The genesis import should include files only by default. Assistant conversation transcripts may be referenced or imported later as optional provenance, but they are not part of the default genesis graph.

The genesis import should record:

- Import timestamp
- Files included
- Content hashes for included files
- Files excluded
- Import actor identity
- Manifest path
- Initial realm policy
- Initial public projection
- Initial maintainer projection
- Known lack of pre-import fine-grained history

The minimum audit format is append-only JSONL. The first audit event should be written to a path such as `.glyph/audit/events.jsonl` and include:

```json
{"type":"genesis_import","timestamp":"2026-05-27T00:00:00Z","actor":"user:self:dhruv","manifest":"glyph.yaml","included":[{"path":"docs/specs/000-bootstrap-and-dogfooding-protocol.md","hash":"sha256:..."}],"excluded":[".git/**",".glyph/**","node_modules/**"],"realms":["public","maintainers"],"transcripts":"excluded-by-default"}
```

The lack of Git-style prior history is intentional. The specs are the project memory.

## Self-Hosting Readiness

Glyph is ready to track itself when it can:

1. Import this workspace into a source graph.
2. Materialize at least one filesystem projection.
3. Preserve file contents and paths across import/export.
4. Represent at least two realms, such as `public` and `maintainers`.
5. Prevent a private file from appearing in the public projection.
6. Record a publication event from one realm to another.
7. Expose a simple file read/write API for an agent.
8. Export the public projection into a normal directory or Git repository.
9. Produce a readable audit record for import and publication events.

Once these are true, Glyph should begin tracking its own specs and prototype code.

The first public export target should be a Git repository. A hosted read-only Glyph projection is desirable later, but it is not required for first self-hosting.

## Bootstrap Workflow

### Create Or Update A Spec

1. Discuss the concept.
2. Agree on the intended scope.
3. Write or update the spec file.
4. Self-review for placeholders, contradictions, vague language, and scope creep.
5. Mark the spec `Accepted` only when it is stable enough to guide implementation.

### Make A Major Design Change

1. Identify which spec owns the concept.
2. If the spec is `Draft`, edit it directly.
3. If the spec is `Accepted`, add an amendment or create a superseding spec.
4. Update affected later-spec references.

### Start Prototype Work

Prototype work can begin once the specs define:

- Realms and projections
- Work graph model
- Storage object model
- Agent file API
- Minimum CLI or API surface
- Self-hosting import path

The first prototype should be evaluated against Spec 000's self-hosting readiness checklist.

## Invariants

1. **No Git by default**
   Git must not become Glyph's hidden project substrate during bootstrap.

2. **Specs are source**
   The spec corpus is the authoritative project memory until Glyph can track itself.

3. **Dogfooding pressure stays visible**
   Any step that would be easier with Git should be treated as design input for Glyph.

4. **Self-hosting is a milestone, not a slogan**
   Glyph tracks itself only after it can satisfy the readiness checklist.

5. **No fake history**
   The genesis import should not fabricate commit history. It should honestly record that prior history lived in the spec documents.

6. **Compatibility is downstream**
   Git export matters, but it must not dictate the bootstrap process.

## Prototype Defaults

- No Git repository is initialized.
- Specs are written as Markdown.
- Canonical specs live in `docs/specs/`.
- A minimal `glyph.yaml` manifest exists before self-hosting.
- Accepted specs require `Accepted-By` and `Accepted-Date`.
- Bootstrap identities include both an actor type and provider.
- The current product and CLI name is `Glyph`.
- The first public realm will include specs by default.
- The first public export target is a Git repository.
- A hosted Glyph projection is deferred until after the first compatibility export.
- The genesis import includes files only by default.
- The genesis import audit format is append-only JSONL.
- Any future private notes or experiments must be explicitly excluded by genesis import policy.
- Self-hosting begins only after the readiness checklist passes.

## Success Criteria

This spec is successful if:

- The project can progress through specs without Git.
- The team can tell which files are canonical.
- Major decisions remain understandable without commit history.
- The first Glyph prototype has a clear dogfooding target.
- The genesis import can honestly represent how the project began.
