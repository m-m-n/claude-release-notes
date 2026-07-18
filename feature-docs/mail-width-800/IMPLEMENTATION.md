# Implementation Plan: Mail Card Width 800px Reflection

## Overview

Reflect the approved design decision (digest card `max-width: 800px`) into
the mail template. Single-task feature; no cross-task coordination needed.

## Technology Stack

- **Language**: Go (stdlib only) — unchanged.

## Layer Structure

Unchanged. The only touched layer is the mail composition package
(`internal/mail`), which holds the digest template with literal inline
style values (mail clients do not support CSS custom properties).

## Shared Components

None (single task).

## Conventions

- Design-token literals are inlined in the template as `style=""` values;
  the width value comes from the design SSOT
  (`feature-docs/release-notes-fetcher/DESIGN.md`: 800px).
- Tests assert on the generated HTML string, following the existing
  `internal/mail` test style.

## Cross-task Design Decisions

None (single task).

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Wider card degrades narrow-client rendering | Low | Low | `max-width` semantics keep narrow clients reflowing; no fixed width introduced |

## Open Questions

- None.
