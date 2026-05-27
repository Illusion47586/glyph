# Spec 011: Command Runner And Execution Sandboxing

Status: Draft
Date: 2026-05-27
Project: Glyph

## Thesis

Agents need to run commands, but command execution is one of the easiest ways to leak source, secrets, credentials, and private metadata.

Glyph's command runner must be policy-mediated from the beginning.

## Problem

A command can leak through:

- Filesystem reads
- Environment variables
- Network requests
- Logs
- Test snapshots
- Generated artifacts
- Exit messages
- Package manager scripts
- CI output

If agents can run arbitrary shell commands over a full checkout, Glyph's projection model is bypassed.

## Goals

- Define a safe command execution boundary.
- Snapshot before and after commands.
- Restrict filesystem access to workspace projections.
- Control environment variables and credentials.
- Capture logs as audit objects.
- Filter command output through policy before exposing it.
- Support local-first execution in v1.

## Non-Goals

- Building a full container platform.
- Guaranteeing perfect sandbox isolation in the first prototype.
- Supporting untrusted arbitrary code execution without host controls.
- Replacing CI systems.

## Command Runner

The command runner executes commands inside a workspace projection.

Inputs:

- Workspace ID
- Command and arguments
- Actor identity
- Environment policy
- Network policy
- Filesystem policy
- Timeout and resource limits

Outputs:

- Exit code
- Redacted stdout/stderr
- Raw logs stored as policy-controlled audit objects
- Files changed
- Before and after snapshots
- Provenance event

## Sandbox Policy

The first prototype should support:

- Working directory restricted to workspace materialization
- Environment allowlist
- No secret injection by default
- Network disabled by default for agent commands
- Timeout
- Maximum output size
- File change scan after command

Network and credentials can be enabled per command through policy.

## Log Handling

Raw logs are sensitive. They may contain paths, secrets, private source, or generated content.

Glyph should:

- Store raw logs in the audit graph.
- Expose redacted logs by default.
- Run secret scanning over logs.
- Attach logs to work context provenance.
- Block publication if logs/generated artifacts reveal denied content.

## Prototype Defaults

- Commands are disabled unless run through Glyph's command runner.
- Network is disabled by default.
- Environment variables are allowlisted.
- Secrets are never injected by default.
- Commands have timeouts and output limits.
- Before/after snapshots are mandatory.
- Logs are audit objects and redacted before display to unauthorized actors.

## Success Criteria

This spec is successful if a prototype can:

- Run a command inside a virtual workspace.
- Snapshot before and after execution.
- Detect changed files.
- Record command provenance.
- Keep raw logs out of public projections.
- Deny network access by default.
- Block publication when command output leaks denied content.
