# Feature: Mail Markdown Rendering and Cool-Palette Refresh

## Overview

Render the translated release-note Markdown as styled HTML in the digest
email (currently shown as raw text), and bring the mail builder's inline
styles in line with the current cool design tokens. Small follow-up feature
to release-notes-fetcher.

## Objectives

- Delivered mails show headings, lists, code, and links as formatted HTML
- Mail colors match `design-system/tokens.yaml` (cool palette)
- Dependency policy unchanged (stdlib + yaml.v3; renderer self-written)

## User Stories

### US1: Formatted digest
As the owner, I want release-note Markdown rendered as HTML, so that the
mail reads like a document instead of raw markup.

**Acceptance Criteria:**
- [ ] `#`–`###` headings, `-`/`*` bullets, `1.` ordered items, `` `code` ``,
      `**bold**`, `[text](url)`, and fenced code blocks are rendered as
      their HTML counterparts
- [ ] Unsupported or unparsable lines fall back to plain paragraphs — never
      dropped
- [ ] Mail colors and Markdown-element styles follow the current tokens /
      DESIGN.md

## Technical Requirements

### Functional Requirements
- **FR1:** The mail builder converts each translated release body from
  Markdown to HTML using a self-written minimal renderer supporting: ATX
  headings (`#`, `##`, `###` — all rendered as the mail's single sub-heading
  style), unordered lists (`- ` / `* `), ordered lists (`1. `), inline code
  (backticks), bold (`**`), links (`[text](url)`, `http`/`https` schemes
  only — other schemes render as plain text), and fenced code blocks
  (` ``` `). Consecutive plain lines form paragraphs with line breaks.
  Unsupported constructs render as escaped plain paragraphs. An unclosed
  fenced block is treated as running to the end of the body.
- **FR2:** All inline styles in the generated mail use the CURRENT
  `design-system/tokens.yaml` values (cool palette: primary #33658A, canvas
  #EDF1F5, on-surface #232A31, on-surface-muted #5C6B7A, divider #D9E1E8,
  surface/on-primary #FFFFFF), including the Markdown-element styles decided
  in this feature's design step (DESIGN.md).
- **FR3:** The claude-batch translation instruction explicitly tells the
  model to preserve Markdown syntax and structure unchanged while
  translating the text content.

### Non-Functional Requirements
- **NFR1 - Dependencies:** unchanged — Go standard library plus
  `gopkg.in/yaml.v3` only; no Markdown library.
- **NFR2 - Security:** all body text is HTML-escaped BEFORE markup wrapping
  (renderer emits markup around escaped text; raw HTML in the source is
  neutralized). Link hrefs restricted to http/https.

## Implementation Approach

### Architecture

Unchanged pipeline. Changes confined to:

```
internal/mail       : markdown renderer (new file) + build.go styling/palette
internal/translate  : promptInstruction wording (FR3)
```

The renderer is a pure function inside `internal/mail`: escaped-Markdown
text in → HTML fragment string out. `Build` calls it per release body in
place of the current escape+`<br>` conversion.

### Data Flow

Unchanged (fetch → translate → build → send → save).

### API Design

No external API changes. Internal contract: the renderer takes one release
body (plain text with Markdown) and returns a self-contained HTML fragment
using only inline styles.

### Database Schema

N/A.

### Dependencies

Unchanged: `gopkg.in/yaml.v3`, `claude-batch`, GitHub REST API, Gmail SMTP.

### File Structure

```
internal/mail/markdown.go        # FR1 renderer (+ markdown_test.go)
internal/mail/build.go           # FR2 palette + element styles; renderer wiring
internal/mail/build_test.go      # updated expectations
internal/translate/claudebatch.go      # FR3 instruction wording
internal/translate/claudebatch_test.go # updated prompt assertions
```

## Test Scenarios

### Unit Tests
- [ ] TS-1: FR1 — each supported construct converts to its HTML element;
      mixed documents (heading + list + code + paragraph) render in order
- [ ] TS-2: FR1 — unsupported constructs and unparsable lines render as
      plain paragraphs; empty body keeps the existing placeholder path
- [ ] TS-3: NFR2 — raw HTML in the body appears escaped; `javascript:` and
      other non-http(s) link schemes are not emitted as hrefs
- [ ] TS-4: FR2 — rendered mail contains only current-token color values
      (no legacy warm hexes) and the element styles from DESIGN.md
- [ ] TS-5: FR3 — the assembled prompt contains the Markdown-preservation
      instruction; existing delimiter assertions still hold
- [ ] TS-6: FR1 — unclosed fenced code block renders to end of body without
      error

### Integration Tests
None (unchanged policy).

### E2E Tests
**Existing E2E tests**: None
**Run command**: Not detected
- [ ] N/A

### Edge Cases
- [ ] Empty body → existing "（リリースノート記載なし）" placeholder
- [ ] Body with no Markdown at all → paragraphs identical in content to
      current behavior
- [ ] List runs interrupted by blank lines terminate the list cleanly

### Performance Tests
N/A.

## Security Considerations

- **Input Validation / XSS:** renderer works on escaped text only; markup is
  added by the renderer itself, so source-supplied HTML never reaches the
  mail as markup. Link URLs must parse and have scheme http or https.
- Other considerations unchanged from release-notes-fetcher.

## Error Handling

Renderer never fails: any line it cannot classify becomes an escaped plain
paragraph. No new error paths in the pipeline.

## Performance Optimization

N/A.

## Success Criteria

- [ ] FR1–FR3 implemented and unit-tested (TS-1…TS-6 pass)
- [ ] Visual comparison against the updated mockup (cool palette + rendered
      Markdown states) passes
- [ ] Code review completed

## Open Questions

None.

## References

- Parent feature: feature-docs/release-notes-fetcher/SPEC.md
- Design tokens: design-system/tokens.yaml
- Design decisions: feature-docs/mail-render-refresh/DESIGN.md (design step)
