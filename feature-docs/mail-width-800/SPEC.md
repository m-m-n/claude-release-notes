# Feature: Mail Card Width 800px Reflection

## Overview

Reflect the approved design decision (mail card `max-width: 800px`, decided
in a /em-workflow:design session and recorded in
`feature-docs/release-notes-fetcher/DESIGN.md`) into the digest email
template. The implementation currently renders the container at
`max-width:600px`.

## Objectives

- Bring the implementation in line with the design SSOT (800px).
- Change nothing else about the rendered mail.

## User Stories

### US1: Wider digest card
As the digest recipient, I want the mail card to use up to 800px of width,
so that long Japanese release notes read comfortably in a desktop pane.

**Acceptance Criteria:**
- [ ] Generated digest HTML contains `max-width:800px` on the outer container
- [ ] No `max-width:600px` remains in the generated digest HTML

## Technical Requirements

### Functional Requirements
- **FR1:** The digest HTML outer container div (`internal/mail/build.go`,
  `digestTemplate`) uses inline style `max-width:800px` instead of
  `max-width:600px`.

### Non-Functional Requirements
- **NFR1 - Style isolation:** No other inline style value (colors, font
  sizes, line heights, padding/margins) changes; no dependency changes.

## Implementation Approach

### Dependencies

**Internal Dependencies:**
- `internal/mail/build.go`: `digestTemplate` holds the literal width value.
- `internal/mail/markdown_test.go`: one comment references the
  "600px mail column" and should be updated to 800px for accuracy.

**External Dependencies:** none.

### File Structure

```
internal/
└── mail/
    ├── build.go            # digestTemplate max-width change
    ├── build_test.go       # width assertion (add/update)
    └── markdown_test.go    # comment wording only
```

## Test Scenarios

### Unit Tests
- [ ] TS-1: `Build` output contains `max-width:800px` — asserts FR1.
- [ ] TS-2: `Build` output does not contain `max-width:600px` — asserts FR1.
- [ ] TS-3: All existing `internal/mail` tests pass unchanged (subject,
  body composition, escaping) — asserts NFR1 regression safety.

### E2E Tests
**Existing E2E tests**: None
**Run command**: Not detected

## Success Criteria

- [ ] FR1 implemented and covered by tests
- [ ] `go build ./...` and `go test ./...` pass
- [ ] `gofmt` reports no diff

## Open Questions

- None.

## References

- feature-docs/release-notes-fetcher/DESIGN.md — width decision SSOT
- feature-docs/release-notes-fetcher/design/mockups/screen-email-digest.html — 800px mockup
