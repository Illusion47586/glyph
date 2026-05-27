---
title: "Spec 006: Git Import/Export Compatibility"
description: "Glyph design specification."
---

Status: Draft
Date: 2026-05-27
Project: Glyph

## Thesis

Glyph should be agent-native without becoming ecosystem-hostile.

Coding agents should interact with Glyph through direct, structured APIs for reading, writing, reviewing, and publishing source graph changes. Humans and existing platforms should still be able to use GitHub, Git, and other version-control systems through translation layers.

Glyph is the canonical model. Git compatibility is an adapter, not the foundation.

## Problem

Git is the common language of today's developer ecosystem, but it is a poor native interface for coding agents:

- Agents do not need full filesystem checkouts for every task.
- Agents need scoped context, not repository-wide access.
- Agents need structured read/write APIs, not shelling out to `git`.
- Agents need to attach intent, provenance, confidence, and review metadata to work.
- Agents should not have to reason in branches, staging areas, remotes, and rebases.

At the same time, Glyph cannot ignore the existing world:

- GitHub is where open source collaboration happens.
- CI/CD systems expect Git repositories and commits.
- Package registries, deploy systems, and review tools expect Git metadata.
- Developers will need migration paths from Git.
- Public Glyph projects should still be mirrorable to GitHub.

Glyph needs a translation layer that preserves compatibility without letting Git dictate the internal model.

## Goals

- Provide first-class APIs for coding agents.
- Make agent permissions, context, and publication boundaries explicit.
- Support Git and GitHub compatibility through projections and translators.
- Allow Glyph projects to publish public views to GitHub.
- Allow a GitHub repository to be configured as a remote origin when teams want to use GitHub infrastructure.
- Allow importing existing Git repositories into Glyph.
- Preserve useful Git metadata without adopting Git's primitives as Glyph's primitives.
- Support future translators for other version-control systems.

## Non-Goals

- Perfect round-trip fidelity for every Git edge case in the first version.
- Reimplementing Git as Glyph's storage layer.
- Exposing raw source graph access to agents by default.
- Depending on GitHub as Glyph's hosted backend.
- Supporting every GitHub feature in the first prototype.

## Core Concepts

### Agent API

The Agent API is the native interface for coding agents.

It should support:

- Listing available realms and projections
- Reading files from a projection
- Writing files into an agent-scoped work context
- Searching allowed source
- Requesting relevant context bundles
- Running or requesting tests
- Producing proposed changes
- Attaching intent and provenance metadata
- Asking for publication or review

The Agent API should be structured and policy-aware. Agents should not need to infer project state from a raw filesystem unless a filesystem projection is explicitly provided.

### Work Context

A work context is a scoped area of active change.

For agents, a work context replaces the mental model of a branch plus working tree plus staging area. It contains:

- Allowed projection
- Files read
- Files written
- Commands or tools invoked
- Intent metadata
- Generated changes
- Review status
- Publication target

Work contexts can later map to Glyph's work graph. This spec only requires that agents have a structured place to do work.

### VCS Translator

A VCS translator maps between Glyph and an external version-control system.

Initial translators:

- Git export translator
- Git import translator
- GitHub publication translator

Future translators may target systems such as Mercurial, Perforce, Fossil, or platform-specific source archives.

### External Source Mount Translator

An external source mount translator maps a subdirectory in a Glyph project to another version-control root.

For Git and GitHub, this supports the useful workflows of submodules and subtrees:

- Link a GitHub repository at a subdirectory.
- Preserve that repository's own remote origin.
- Pin the mounted source to a revision.
- Optionally export the mount as a Git submodule, vendored subtree, or generated directory.
- Optionally import upstream changes from the mounted repository.

The mount translator should make the chosen export mode explicit. Silent conversion between pointer-style mounts and vendored contents is not allowed.

### Projection Export

Projection export materializes a Glyph projection into another system.

For GitHub compatibility, the most important export is:

- Glyph `public` projection -> Git repository -> GitHub remote

Export must obey realm policy. Hidden objects must not leak through commits, file paths, tags, branch names, commit messages, generated artifacts, or review metadata.

Git compatibility files may be generated during projection export:

- `.gitignore`: generated from Glyph exclude rules and hard safety excludes.
- `.gitinclude`: generated from Glyph include rules for humans or tools that want to inspect the Glyph import allowlist.

Generation must be explicit. Glyph should not overwrite an exported projection's `.gitignore` or `.gitinclude` unless the user requests overwrite behavior.

Prototype flags:

```sh
glyph export git --realm public --out ./export --gitignore generated
glyph export git --realm public --out ./export --gitinclude generated
glyph export git --realm public --out ./export --gitignore none --gitinclude none
```

Allowed values:

- `none`: do not generate the file.
- `generated`: write a Glyph-generated compatibility file only if the exported projection does not already contain that path.
- `overwrite`: write the generated compatibility file even if the exported projection already contains that path.

Generated files should include a header identifying them as Glyph-generated and should preserve hard safety defaults, including `.glyph/**` and `.git/**`.

### Remote Origin

A remote origin is an external system that Glyph can synchronize with through a translator.

For GitHub, a remote origin can provide:

- Public repository hosting
- Pull request UI
- Issue tracking
- CI/CD triggers
- Release infrastructure
- Package and deployment integrations
- Contributor discovery and social proof

A remote origin does not have to be Glyph's canonical store. It is a configured sync target or source with explicit direction, policy, and lossiness.

Possible modes:

- `export-only`: Glyph publishes an authorized projection to GitHub.
- `import-only`: Glyph imports from an existing GitHub repository.
- `bidirectional-public`: Glyph syncs public-compatible changes both ways.
- `compatibility-first`: GitHub/Git remains operationally canonical while Glyph provides agent APIs, policy overlays, and metadata where possible.

Private realms require Glyph-native storage unless the remote origin can enforce equivalent policy semantics.

### External Source Mount

A GitHub repository can also be configured as a mounted subdirectory rather than as the project origin.

Example:

```text
glyph mount add vendor/parser github:owner/parser --path vendor/parser --mode pinned
```

Mount modes:

- `pinned`: parent project records a specific upstream revision.
- `tracking`: parent project can update to the latest allowed upstream state.
- `vendored-export`: public Git export includes the mounted files.
- `submodule-export`: public Git export emits a Git submodule pointer.

The first prototype should only require `pinned` metadata and `vendored-export` for public Git export. Native submodule export can come later.

Mounted GitHub repositories should export as vendored files by default for compatibility. Git submodule pointer export should be opt-in because submodules impose extra user workflow requirements.

### Import

Import converts an external repository into Glyph's source graph.

For Git, import should preserve:

- File contents
- Paths
- Commit authorship
- Commit timestamps
- Commit messages
- Tags
- Branch tips
- Basic rename information where available

Glyph should not promise that imported Git commits remain the native unit of work forever. Git history can be represented as imported provenance while Glyph creates its own work graph.

## Agent-Native Requirements

Glyph should be pleasant for agents in ways Git is not:

1. **Scoped source access**
   Agents receive the smallest projection needed for the task.

2. **API-first file operations**
   Agents can read and update files through API calls without requiring a full OS checkout.

3. **Intentful changes**
   Agent changes include task intent, touched files, rationale, and confidence metadata.

4. **Reviewable provenance**
   Humans can see which agent produced work, what context it saw, and what it attempted.

5. **Policy-aware context**
   Agents cannot accidentally inspect private realms or publish hidden objects through Git export.

6. **Multiple simultaneous agents**
   Agents can work in separate contexts without branches or worktrees.

7. **Publication as explicit transition**
   Agent work becomes public only through Glyph publication checks.

## Translation-Layer Requirements

Glyph's compatibility layer should obey these rules:

1. **Glyph is canonical**
   External VCS systems are projections, imports, or mirrors.

2. **Policy applies before translation**
   Translators operate only on already-authorized projections.

3. **Translation is auditable**
   Imports and exports produce records describing source, destination, included objects, excluded objects, and warnings.

4. **Compatibility is practical**
   Glyph should support common GitHub workflows before exotic Git internals.

5. **No private-object ghosts**
   Exported Git repositories must not contain unreachable private blobs, hidden refs, private paths, or private commit metadata.

6. **Lossiness is explicit**
   When Glyph cannot preserve semantics across systems, the translator must report what was lost.

## GitHub Compatibility

Glyph should support GitHub as a publication and collaboration surface.

Minimum useful GitHub compatibility:

- Configure a GitHub repository as a Glyph remote origin.
- Configure a GitHub repository as an external source mount at a subdirectory.
- Export `public` realm to a Git repository.
- Push exported repository to GitHub.
- Represent Glyph publication events as Git commits or pull requests.
- Import GitHub repository contents into Glyph.
- Pull public-compatible changes from GitHub into Glyph.
- Preserve links between Glyph work contexts and exported GitHub commits.
- Avoid exporting private realms, hidden metadata, or private agent context.

GitHub can be a mirror, import source, remote origin, and infrastructure provider. For Glyph-native projects, Glyph remains the canonical source graph unless the project explicitly chooses a compatibility-first mode.

GitHub pull requests should map to Glyph publication requests when imported into a Glyph-native project. Exported public pull requests may also exist as GitHub review artifacts, but the canonical review/publication state lives in Glyph.

Imported Git branches should be provenance by default. They may seed work contexts during migration, but they should not become realms unless explicitly mapped by policy.

GitHub issues and review comments that mention private Glyph context should be treated as external public metadata by default. Private context must not be pushed to GitHub; inbound GitHub metadata that appears to reference private context should be linked as provenance and flagged for human review.

Agents may request broader context, but only humans or policy-authorized actors can approve widening the projection.

The minimum Agent API for the first prototype is the local workspace/file API defined in Spec 007: create workspace, read, write, list, search, diff, snapshot, request review, and request publication.

Glyph should expose MCP as the default agent integration surface, implemented as a wrapper over the internal local API.

When GitHub and Glyph both contain public changes, the safest default is no automatic reconciliation. Glyph should import GitHub changes as proposals/provenance and require explicit human adoption.

GitHub Actions status checks should be represented as external check results attached to publication requests. Policy may require named checks to pass before export or publication.

## Remote Origin Modes

### Export-Only Origin

Glyph pushes an authorized projection to GitHub.

This is the safest default for Glyph-native projects:

1. Glyph maintains the canonical source graph.
2. Publication moves objects into `public`.
3. Git translator creates a clean Git export.
4. GitHub origin receives commits, tags, releases, or pull requests.

GitHub users can read, clone, open issues, and consume releases, but direct GitHub code changes are not automatically canonical.

### Import-Only Origin

Glyph imports from GitHub but does not publish back.

This is useful for migration, analysis, or agent work over an existing project before maintainers adopt Glyph as canonical.

### Bidirectional Public Origin

Glyph synchronizes public-compatible changes with GitHub in both directions.

This mode is useful when a project wants GitHub's collaboration UI to remain active:

1. Glyph exports public publications to GitHub.
2. Glyph imports GitHub pull requests, commits, and tags as external proposals or provenance.
3. Imported changes pass through Glyph policy before becoming canonical.
4. Private Glyph objects are never pushed to GitHub unless explicitly published.

Bidirectional mode must be conservative. GitHub can propose changes to Glyph; it should not bypass Glyph publication policy.

### Compatibility-First Mode

Some teams may want Glyph's agent APIs and projection model while keeping GitHub/Git as the operational source of truth.

Glyph may support this mode, but it should be labeled clearly:

- GitHub/Git remains canonical for public code.
- Glyph stores additional metadata, agent contexts, and policy overlays.
- Private realms may be limited or require Glyph-native storage.
- Some Glyph features may be unavailable because Git cannot represent them.

This mode is an adoption bridge, not the ideal architecture.

## Example Workflows

### Agent Fix Published To GitHub

1. User asks an agent to fix a public bug.
2. Glyph creates an agent work context from the `public` projection plus any allowed maintainer context.
3. Agent reads and writes through the Agent API.
4. Glyph records the proposed work in the source graph.
5. Human reviews the work.
6. Human publishes selected changes to `public`.
7. Git translator exports the updated `public` projection.
8. GitHub translator pushes a commit or opens a pull request.

### GitHub Repository As Remote Origin

1. User configures a GitHub repository as `origin`.
2. Glyph records the origin URL, sync mode, allowed realms, and credentials policy.
3. Glyph imports existing public state from GitHub if requested.
4. Glyph publishes future `public` changes to GitHub through clean Git export.
5. GitHub CI, releases, issues, and pull requests can continue to operate.
6. Any inbound GitHub changes are imported as proposals or provenance and checked by Glyph policy.

### GitHub Repository Mounted As Subdirectory

1. User configures a GitHub repository as an external source mount at `vendor/parser`.
2. Glyph records the mount path, remote origin, pinned revision, and export mode.
3. Parent project projections include the mounted source according to policy.
4. Public Git export can include vendored files for compatibility.
5. The mounted source can still sync with its own GitHub repository.
6. Updates to the mount are tracked as parent-project work, not as accidental directory edits.

### Import Existing GitHub Project

1. User points Glyph at a GitHub repository.
2. Git translator imports Git objects as provenance.
3. Glyph creates an initial `public` realm from the repository contents.
4. Glyph records branches and tags as imported Git metadata.
5. Future Glyph work uses realms and work contexts instead of native Git branches.
6. Public state can continue to mirror back to GitHub.

### Private Realm With Public Mirror

1. Maintainers work in private realms.
2. Public users see GitHub as a normal open source repository.
3. Glyph stores private work in the canonical source graph.
4. Publication checks decide which changes enter `public`.
5. GitHub receives only the authorized public projection.

## Metadata Mapping

Git compatibility must map intent, not just object names.

Git primitives are overloaded. A Git branch can mean active work, public release lane, deploy environment, remote tracking pointer, or review artifact. A Git commit can mean savepoint, review unit, publication, imported history, or merge marker. Glyph should translate based on mode and purpose rather than pretending that one Git primitive always equals one Glyph primitive.

### Primitive Mapping

| Git primitive | Glyph primitive |
| --- | --- |
| commit | publication event, checkpoint, snapshot, or imported provenance |
| branch | work context, realm, or remote tracking view |
| HEAD | current realm projection pointer or exported Git branch tip |
| working tree | materialized workspace projection |
| index/staging area | internal projection/workspace index, not user-facing |
| merge commit | publication integrating multiple work contexts |
| squash merge | `publish --mode squash` |
| preserve-history merge | `publish --mode preserve` |
| tag | named source graph reference or release marker |
| remote | Glyph remote origin |
| submodule/subtree | external mount |

### Commit Mapping

Glyph decides what a Git commit represents from context:

- Git commit imported from an existing repository -> imported provenance object.
- Git commit exported from `publish --mode squash` -> one Glyph publication event.
- Git commits exported from `publish --mode preserve` -> selected checkpoints or snapshots from the work context.
- Git merge commit imported from GitHub -> imported integration provenance.
- Git commit used only for GitHub PR review -> external review artifact, not canonical Glyph history.

Glyph-native source reconstruction must not depend on Git commit replay. Imported commits can seed realms, work contexts, checkpoints, or provenance, but the source graph remains canonical after adoption.

### Branch Mapping

Glyph decides what a Git branch represents from purpose:

- `main`, `master`, or configured public branch on import -> initial `public` realm seed or remote tracking view.
- Feature branch on import -> imported work context or imported publication proposal.
- Protected release branch -> realm or named projection, if explicitly configured.
- GitHub PR branch exported from Glyph -> temporary external review branch for a publication request.
- Remote tracking branch -> remote tracking view, not a native realm unless policy maps it.

Imported branches should be provenance by default. They should become realms only through explicit mapping because realms carry visibility and policy semantics that Git branches do not have.

Example branch mapping config:

```yaml
git:
  import:
    branches:
      main:
        map_to: realm
        realm: public
      "feature/*":
        map_to: work_context
      "release/*":
        map_to: realm
        realm_prefix: release/
```

### Publication Mapping

Glyph publication is the canonical visibility transition.

Export behavior:

- `publish --mode squash` -> one Git commit on the exported branch.
- `publish --mode preserve` -> multiple Git commits derived from selected checkpoints/snapshots.
- Publication with multiple dependencies -> merge-style Git history may be generated, but Glyph records the integration as a publication event.
- Publication request exported for GitHub review -> GitHub pull request plus temporary Git branch.
- Accepted publication -> exported public branch update.

Import behavior:

- GitHub pull request -> Glyph publication request.
- GitHub review approval -> external review provenance until adopted by policy.
- GitHub Actions check run -> external check result attached to the publication request.
- GitHub merge/squash/rebase action -> imported integration provenance and candidate public realm update.

### Realm Mapping

Realms are permissioned source views. Git branches are not permission boundaries.

Export behavior:

- `public` realm usually exports to `main`.
- Additional public-compatible realms may export to named Git branches if configured.
- Private realms must not export to Git unless the destination can enforce equivalent policy semantics.

Import behavior:

- A Git branch may seed a realm only when explicitly configured.
- A branch name alone must not create a private or public realm.
- Remote branch tips can be stored as remote tracking views for comparison and provenance.

### Operation Mapping

Glyph operations map to Git/GitHub operations only at the translator boundary:

| Glyph operation | Git/GitHub export |
| --- | --- |
| `work start` | optional temporary branch for PR export |
| `checkpoint` | optional commit in preserve export |
| `publish --mode squash` | one commit or squash-merge result |
| `publish --mode preserve` | commit series or merge result |
| `publication request` | GitHub pull request |
| `remote sync` | push/pull through configured translator mode |
| `mount add` | subtree, vendored directory, or submodule depending on export mode |

Initial Git export can map Glyph concepts as follows:

- Public projection files -> Git tree
- Publication event -> Git commit
- Publication author -> Git author or committer metadata
- Glyph work context ID -> commit trailer
- Glyph review ID -> commit trailer or pull request metadata
- Glyph source graph object IDs -> optional commit trailers

Example commit trailers:

```text
Glyph-Publication: pub_01
Glyph-Work-Context: work_01
Glyph-Projection: public
```

This mapping is allowed to be boring. The goal is compatibility, not making Git beautiful.

## Security Considerations

Translation layers are dangerous because they can smuggle hidden state into public systems.

Important risks:

- Git object databases can contain unreachable blobs.
- Commit messages can mention private filenames or vulnerabilities.
- Branch names can reveal hidden projects.
- Tags can reveal private release plans.
- Pull request descriptions can include private agent reasoning.
- CI logs can reveal generated private content.
- GitHub issue links can reveal embargoed work.

Glyph translators must produce clean exports, not filtered copies of dirty repositories.

For Git export, this means building a fresh Git repository from the authorized projection rather than copying an existing `.git` directory.

For GitHub remote origins, credentials must be scoped to the sync mode. An export-only origin should not need broad repository administration access. An import-only origin should not need write access.

## Prototype Defaults

- The first agent integration is the `glyph --json` CLI plus project skill document.
- Agent work happens in explicit work contexts.
- The first VCS translator targets Git.
- The first platform translator targets GitHub.
- GitHub can be configured as a remote origin in `export-only` mode.
- GitHub repositories can be configured as pinned external source mounts.
- GitHub pull requests import as Glyph publication requests for Glyph-native projects.
- Imported Git branches are provenance by default.
- Imported Git commits are provenance by default.
- Git branches become realms only through explicit mapping.
- Git feature branches may seed work contexts or publication proposals.
- `publish --mode squash` exports as one Git commit by default.
- `publish --mode preserve` may export selected checkpoints as a commit series.
- GitHub PR branches are external review artifacts, not canonical Glyph work.
- Inbound GitHub changes are never adopted automatically.
- GitHub Actions checks attach to publication requests as external check results.
- Git export creates a fresh repository from the `public` projection.
- Git import preserves Git commits as provenance metadata, not native Glyph work objects.
- Mounted GitHub repositories export as vendored files by default.
- Lossy translations are acceptable when warnings are explicit.

## Success Criteria

This spec is successful if Glyph can support a prototype where:

- An agent reads and writes source through a Glyph-native API.
- Agent work is captured in a scoped work context.
- A human can review and publish agent work into `public`.
- The `public` projection can be exported into a clean Git repository.
- The clean Git repository can be pushed to GitHub.
- A GitHub repository can be configured as an `export-only` remote origin.
- A GitHub repository can be mounted as a subdirectory with a pinned revision.
- Existing Git repository contents can be imported into Glyph as initial public state.
- Private Glyph objects do not appear in exported Git objects or metadata.
