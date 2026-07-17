# Implementation Plan: Claude Code Release Notes Fetcher

## Overview

A single-shot Go CLI that fetches new `anthropics/claude-code` releases,
translates them to Japanese via the local `claude-batch` command, and emails
them as one HTML digest through Gmail SMTP.

## Technology Stack

- **Language**: Go (standard library only, single static binary)
- **Third-party**: `gopkg.in/yaml.v3` — YAML config parsing (the ONLY external
  module; NFR1)
- **External at runtime**: GitHub REST API, `claude-batch` command, Gmail SMTP

## Layer Structure

```
cmd/claude-release-notes  → internal/app → internal/{config,state,github,translate,mail}
```

- `cmd/claude-release-notes`: entry point only — flag-free, wires concrete
  components into the app pipeline and maps errors to exit codes.
- `internal/app`: the run pipeline (fetch → translate → build → send → save
  state). Depends on the five leaf packages ONLY through interfaces it
  declares itself (consumer-side interfaces); this is what unit tests mock.
- Leaf packages (`config`, `state`, `github`, `translate`, `mail`): self-
  contained, no imports between each other, no shared types — `internal/app`
  converts between their value types.

Dependency direction: cmd → app → leaves. Leaves never import app or cmd.

## Module identity

- Module path: `claude-release-notes` (local tool, not published).
- `go.mod` may be created by ANY task whose worktree lacks it, with exactly:
  module path above, the project's Go toolchain version, and (only when the
  task itself imports it) `gopkg.in/yaml.v3`. Identical content keeps merges
  trivial; a conflict is resolved by the standard parent-side adoption
  protocol.

## Shared Components

Tasks run fully in parallel; every cross-task use below is pinned here so
both sides implement against the contract independently.

| Component | Responsibility | Contract (pre/postcondition) | Used by tasks |
|-----------|----------------|------------------------------|---------------|
| `config.Load(baseDir)` | Read + validate YAML config | Pre: baseDir is the config directory to read from (caller resolves XDG; empty string means "resolve XDG default internally"). Post: returns a Config value with fields GmailAccount, GmailAppPassword, MailTo, GitHubToken, all non-empty — or an error naming the missing/empty key WITHOUT printing its value. | task0001 (provides), task0006 (consumes) |
| `state.Store` | Persist last delivered version | Constructor takes the state directory (caller resolves XDG; empty string means "resolve XDG default internally"). `Load` post: returns (version, found=true) or (found=false) when no state exists — never an error for mere absence. `Save(version)` post: state persisted atomically (temp file + rename), directories created as needed. | task0002 (provides), task0006 (consumes) |
| `github.Client` | Fetch releases | Constructor takes token and API base URL (overridable for tests). `Latest()` post: the newest non-draft, non-prerelease release. `Since(version)` post: all non-draft, non-prerelease releases newer than `version`, newest first, following pagination; empty slice when up to date. Release value has fields TagName, Body, PublishedAt. | task0003 (provides), task0006 (consumes) |
| `translate.Translator` | Japanese translation via claude-batch | Constructor takes the command name (default `claude-batch`; overridable for tests). `Translate(sections)` pre: one string per release body, order preserved. Post: same-length, same-order slice of translated texts — or an error when the subprocess fails, output is empty, or any section cannot be recovered from the output. Invoked as the command with single argument `-`, prompt written to stdin (never the `-p` flag). | task0004 (provides), task0006 (consumes) |
| `mail.Build(items)` | Compose digest | Pre: items ordered newest first; each item has Version, PublishedAt, translated Body (plain text, HTML-escaped by Build). Post: returns (subject, htmlBody); subject format `Claude Code リリースノート: {newest} ほか N 件` (single item: without the ほか part); htmlBody is the self-contained inline-styled digest. | task0005 (provides), task0006 (consumes) |
| `mail.Sender` | SMTP send | Constructor takes host, port, account, app password (host/port overridable for tests). `Send(to, subject, htmlBody)` post: one MIME `text/html; charset=UTF-8` message delivered via STARTTLS SMTP AUTH; UTF-8 subject encoded per MIME. | task0005 (provides), task0006 (consumes) |

Placeholder rule: task0006 may create compile-only stubs of the five leaf
packages, each limited to the contract above and marked with a leading
placeholder comment. On merge conflict the real implementation (parent side)
is adopted per the standard protocol. task0006 itself owns the final wiring
of real components in the entry point (integration-wiring owner).

## Conventions

- Errors: wrap with a stage prefix ("config: …", "github: …") and propagate
  up; only the entry point prints and exits. Exit code 0 on success (including
  "no new releases"), 1 on any failure. Secrets never appear in error text or
  logs (NFR2).
- Logging: progress lines to stdout, errors to stderr, standard library only.
  Any timestamp printed includes an explicit timezone (NFR3).
- State is saved ONLY after a successful send (NFR4) — ordering enforced in
  `internal/app`.
- Tests: table-driven, colocated `_test.go`, mocks via the app-side
  interfaces; no real network/subprocess/SMTP (test/README.md).
- HTML mail: all styling inline (`style=""` attributes) with literal values
  from `design-system/tokens.yaml`; CSS custom properties and `<style>`
  blocks must not be relied on (mail-client constraint pinned in the design
  step).

## Cross-task Design Decisions

### Consumer-side interfaces in internal/app
`internal/app` declares its own small interfaces for the five leaf roles and
is wired with the concrete implementations by the entry point. Rationale: the
leaves stay import-free of each other, unit tests mock at one seam, and
parallel tasks never need a sibling package to compile their own tests.
Affected: all tasks.

### No shared Release type
`github`, `translate`, and `mail` each own their minimal value types;
`internal/app` maps between them. Rationale: removes the one package every
task would otherwise import (a parallel-merge hotspot). Affected: task0003,
task0004, task0005, task0006.

### Translation delimiter protocol is translate-internal
The prompt construction, per-release delimiters, and output re-splitting are
fully encapsulated in `translate`; consumers see only sections-in /
translations-out. Rationale: FR4's "one combined call" and its
delimiter-recovery edge case stay testable in one place. Affected: task0004,
task0006.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Parallel tasks create diverging go.mod | medium | low | Exact content pinned in "Module identity"; parent-side adoption on conflict |
| claude-batch output loses delimiters on long input | medium | medium | Treated as error (state untouched → retried next run); splitting logic unit-tested (TS-4) |
| Gmail SMTP specifics (STARTTLS, app password) untestable offline | high | low | Sender takes overridable host/port; unit tests cover message construction; real send verified manually in verify phase |

## Open Questions

- [ ] None.
