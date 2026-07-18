# Implementation Plan: Mail Markdown Rendering and Cool-Palette Refresh

## Overview

Render translated release-note Markdown as styled HTML in the digest email
and align the mail builder's inline styles with the current cool design
tokens. Follow-up to release-notes-fetcher; pipeline architecture unchanged.

## Technology Stack

- **Language**: Go (standard library only; the Markdown renderer is
  self-written — no Markdown library, NFR1)
- Existing module `claude-release-notes` (go.mod untouched)

## Layer Structure

Unchanged from release-notes-fetcher. Changes are confined to two leaf
packages with no new cross-package dependencies:

- `internal/mail` — Markdown renderer (new file) + palette/element styles
- `internal/translate` — prompt instruction wording only

## Shared Components

None new. The two tasks touch disjoint packages and share no contracts;
the renderer is `internal/mail`-private (its contract lives in the task
plan). The existing `mail.Build` / `translate.Translate` signatures from the
parent feature's IMPLEMENTATION.md remain unchanged.

## Conventions

- Escape-before-markup (NFR2): the renderer receives raw body text, escapes
  every text run first, and emits markup itself. `template.HTML`-style raw
  passthrough of source content is forbidden.
- All colors/sizes in generated HTML are literal values from
  `design-system/tokens.yaml` (cool palette + surface-muted / heading-body /
  code tokens) as inline `style=""` attributes — CSS custom properties and
  `<style>` blocks remain unreliable in mail clients (parent DESIGN.md
  constraint, restated in this feature's DESIGN.md).
- Error policy unchanged: the renderer never fails; unclassifiable input
  degrades to escaped plain paragraphs.

## Cross-task Design Decisions

### Renderer stays inside internal/mail
The Markdown renderer is a package-private component of `internal/mail`,
not a new shared package. Rationale: mail is its only consumer; a separate
package would create an import surface no other component needs. Affected:
task0002 only (recorded here because it constrains future features too).

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Renderer edge cases (unclosed fences, odd nesting) mis-render | medium | low | Plain-paragraph fallback guarantees no content loss; TS-2/TS-6 cover the known cases |
| Legacy warm hexes left behind in build.go | low | low | TS-4 asserts no legacy warm hex values appear in output |

## Open Questions

- [ ] None.
