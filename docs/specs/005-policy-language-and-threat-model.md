# Spec 005: Policy Language And Threat Model

Status: Draft
Date: 2026-05-27
Project: Glyph

## Thesis

Glyph's core promise depends on policy correctness.

If hidden source leaks through projections, metadata, exports, logs, or agent context, Glyph fails. The policy language must be simple enough to audit, expressive enough to model real projects, and conservative enough to fail closed.

## Problem

Git has no native visibility policy. Teams compensate with private repos, forks, secret managers, `.gitignore`, social rules, and CI discipline.

Glyph moves visibility into source control itself. That creates power and risk:

- A policy bug can leak private source.
- A generated artifact can expose hidden implementation.
- Git export can contain unreachable private objects.
- Agents can accidentally receive forbidden context.
- Logs, diffs, and review comments can reveal secrets even when files do not.

Glyph needs a policy model and threat model from the beginning.

## Goals

- Define what policy controls.
- Define expected leak surfaces.
- Provide a minimal policy language shape.
- Require fail-closed behavior.
- Make publication checks explicit and auditable.
- Support secrets, private packages, embargoed fixes, and agent scopes.

## Non-Goals

- Proving formal security properties in the first version.
- Replacing runtime secret managers.
- Designing organization-wide IAM.
- Supporting arbitrary policy code execution.
- Solving all supply-chain security problems.

## Policy Scope

Policy applies to:

- Glyph read access
- Work context creation
- File reads and writes
- Directory listings
- Search results
- Diffs
- Generated artifacts
- Test logs
- Agent context bundles
- Publication requests
- Git exports
- GitHub remote sync
- Import adoption

Policy must treat metadata as potentially sensitive.

## Policy Language Shape

The first policy language should be declarative.

It should express:

- Realms
- Groups and identities
- Read grants
- Write grants
- Publication grants
- Required reviewers
- Path rules
- Label rules
- Redaction rules
- Export rules
- Import rules

Illustrative sketch:

```yaml
realms:
  public:
    read: ["*"]
  maintainers:
    read: ["group:maintainers"]
    write: ["group:maintainers", "agent:*"]
  security:
    read: ["group:security"]
    write: ["group:security"]

publish:
  - from: maintainers
    to: public
    reviewers: 1
  - from: security
    to: public
    reviewers: 2
    require_labels: ["disclosure-approved"]

redactions:
  - match: ".env*"
    deny_realms: ["public"]
  - label: "secret-never-publish"
    deny_realms: ["*"]
  - label: "private-implementation"
    deny_realms: ["public"]
```

The syntax is not final. The required capability is more important than the spelling.

Policy should support explicit inheritance between realms, but inheritance must be shallow and inspectable. A realm may extend another realm, and the effective policy must be renderable for review.

When path rules and labels conflict, the most restrictive rule wins. Deny beats allow. Specific rules can explain why access was denied, but they should not silently override stronger redactions.

Policy must not execute custom code in the first version. Custom evaluators can be explored later only if sandboxed, deterministic, and auditable.

## Threat Model

### Protected Assets

- Private source files
- Secrets and credentials
- Embargoed vulnerability details
- Private package implementation
- Customer-specific code
- Agent prompts and transcripts
- Review comments
- Build logs
- Import/export metadata

### Actors

- Public anonymous user
- Maintainer
- Security maintainer
- CI system
- Coding agent
- External contributor
- Compromised dependency or tool
- Misconfigured remote origin

Actors should be represented by stable IDs with providers.

Example identities:

```text
user:self:dhruv
user:org/acme:alice
agent:codex:session-01
agent:claude-code:session-02
```

Policy grants should refer to these identities or groups derived from their providers.

### Threats

- Public projection includes private file contents.
- Public projection includes private file paths.
- Git export contains unreachable private blobs.
- Generated types or bundles reveal private implementation.
- Agent receives more context than intended.
- Agent writes private content into public files.
- CI logs expose hidden content.
- Commit messages or PR descriptions mention embargoed information.
- GitHub import adopts untrusted changes without policy checks.
- Policy change accidentally widens access.

## Required Invariants

1. **Fail closed**
   If policy cannot decide, access is denied.

2. **Policy before projection**
   Filtering happens before materialization, search, diffing, or export.

3. **Metadata is source**
   Paths, names, messages, logs, and transcripts are policy-controlled.

4. **Publication requires checks**
   Visibility widening is never implicit.

5. **Agents receive least privilege**
   Agent projections are scoped and auditable.

6. **Exports are clean**
   Git and GitHub exports are generated from authorized projections only.

7. **Policy changes are reviewed**
   Any policy update that widens access requires review.

## Publication Check Requirements

Before publishing, Glyph should check:

- Actor permission
- Source and destination realms
- Object labels
- Path rules
- Generated artifact provenance
- Secret scanning results
- Required reviewers
- Remote origin mode
- Metadata leak risks
- Audit record creation

If a check fails, Glyph should report the reason without exposing forbidden content to unauthorized users.

Glyph should detect generated artifacts that leak private code through provenance and scanning. Generated outputs must retain links to their inputs; if any private input contributes to a public artifact, publication requires an explicit generated-artifact rule and content scan.

Glyph should include built-in baseline secret scanning and allow external scanners as optional additional checks.

Policy errors should be explained with safe summaries, rule IDs, and paths only when the actor is allowed to know those paths. Unauthorized users should receive generic denial messages with trace IDs for maintainers.

## Prototype Defaults

- Policy is YAML.
- Deny by default.
- Realm inheritance is allowed only when effective policy can be rendered.
- Most restrictive rule wins when paths and labels conflict.
- `.env*` is denied from `public`.
- `secret-never-publish` is denied everywhere except local private storage.
- Publication to `public` requires at least one explicit action.
- Git export always uses a freshly generated repository.
- Agent context bundles are treated as projections and audited.
- Policy changes that widen access require human approval.
- No custom policy code execution in v1.
- Built-in baseline secret scanning is enabled by default.
- External scanners may be configured as additional checks.

## Success Criteria

This spec is successful if a prototype can:

- Load a declarative policy.
- Deny public access to private paths and labels.
- Block publication of `.env` files.
- Require review for security-to-public publication.
- Scope an agent context bundle by policy.
- Produce a clean Git export with no hidden objects.
- Record policy decisions in audit logs.
