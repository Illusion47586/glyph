# Spec 009: Identity, Trust, Credentials, And Signing

Status: Draft
Date: 2026-05-27
Project: Glyph

## Thesis

Glyph's security model depends on knowing who or what acted, which provider asserted that identity, what authority they had, and whether the resulting records can be trusted later.

Identity strings are enough for bootstrap. They are not enough for publication, remote sync, or hosted collaboration.

## Problem

Glyph uses identities for approvals, agent provenance, policy grants, publication, GitHub remotes, and audit records. If identity and credential handling stay informal, Glyph cannot safely enforce policy.

Important questions:

- Who accepted a spec?
- Which org grants a user maintainer access?
- Which agent wrote a change?
- Which token pushed to GitHub?
- Can an audit event be verified later?
- Can a compromised agent publish code?

## Goals

- Define stable identity format.
- Distinguish user, agent, service, and external identities.
- Define provider trust.
- Define credential scope and storage requirements.
- Define signing requirements for audit-sensitive events.
- Support local bootstrap without requiring hosted identity.

## Non-Goals

- Building a full identity provider in v1.
- Replacing GitHub auth.
- Defining enterprise SSO integrations in detail.
- Requiring cryptographic signing for every prototype object.

## Identity Format

Identities have an actor type, provider, and subject.

Examples:

```text
user:self:dhruv
user:org/acme:alice
agent:codex:session-01
agent:claude-code:session-02
service:github-actions:run-123
external:github:owner/repo#pull/42
```

The provider asserts the subject. `self` is allowed during bootstrap but should be treated as local trust only.

## Trust Model

Glyph should distinguish:

- Self-asserted local identities
- Organization-provided identities
- Agent provider identities
- External platform identities
- Service identities

Policy should be able to grant authority to:

- Exact identities
- Groups
- Provider-scoped roles
- Service accounts
- Agent classes

Agent identities should not inherit human authority automatically. A human may delegate a scoped task to an agent, but the audit record must preserve the agent as the actor.

## Credentials

Credentials include:

- GitHub tokens
- Signing keys
- Local user keys
- Agent session tokens
- Service account tokens

Credentials must be scoped to the operation:

- Export-only remotes need write access to the target repository, not broad account access.
- Import-only remotes do not need write access.
- Agents should receive workspace-scoped tokens, not project-wide tokens.
- Publication tokens should be short-lived where possible.

Glyph should never store raw credentials in public source objects. Local credential storage belongs in `.glyph/credentials/` or the operating system credential store, excluded from public projections by policy.

## Signing

The first prototype does not need full object signing, but it should reserve the model.

Events that should eventually be signed:

- Accepted specs
- Policy changes
- Publication approvals
- Publication events
- Remote sync events
- Genesis import events
- Revocation events

Signing should cover event contents, actor identity, timestamp, relevant policy version, and referenced object IDs.

## Prototype Defaults

- Bootstrap identities use string IDs.
- `self` identities are local trust only.
- Agent identities include provider and session.
- Credentials are never source graph objects visible to `public`.
- Remote credentials are scoped by remote mode.
- Signing is optional in v1 but event schemas include fields for signatures.
- Human approval is required for any policy change that widens access.

## Success Criteria

This spec is successful if a prototype can:

- Record user and agent identities with providers.
- Store approval and publication actor IDs.
- Scope a GitHub remote token to its sync mode.
- Keep credentials out of public projections.
- Include signature fields in audit events even if unsigned.
- Distinguish human authority from delegated agent action.
