# Verification Document: Mail Markdown Rendering and Cool-Palette Refresh

## Overview
**Feature**: mail-render-refresh / **SPEC.md**:
`feature-docs/mail-render-refresh/SPEC.md` / **IMPLEMENTATION.md**:
`feature-docs/mail-render-refresh/IMPLEMENTATION.md`

## Build Verification
- Command: `go build ./...`
- Expected: exit code 0, no errors

## Test Verification
- Command: `go test ./...`
- Coverage target: no numeric floor; every TS scenario below covered

### Test Scenarios from SPEC.md
| ID | Scenario | Expected Result | Test Type |
|----|----------|-----------------|-----------|
| TS-1 | Supported Markdown constructs | Heading/ul/ol/inline code/bold/link/fenced block each render to styled HTML; mixed documents render in order | Unit |
| TS-2 | Fallback | Unsupported constructs and Markdown-free text render as escaped plain paragraphs; empty body keeps the placeholder path | Unit |
| TS-3 | Injection safety | Raw HTML escaped; `javascript:` and other non-http(s) schemes never emitted as hrefs | Unit |
| TS-4 | Palette | Output contains current token values only; no legacy warm hexes (#C15F3C #F4F1EC #2B2620 #6E675E #E4DED4) | Unit |
| TS-5 | Prompt instruction | Assembled prompt contains the Markdown-preservation directive; delimiter assertions unchanged | Unit |
| TS-6 | Unclosed fence | Renders to end of body without error, content escaped verbatim | Unit |
| TS-7 | Dependency minimality | go.mod unchanged (stdlib + yaml.v3 only) | Manual (file inspection) |

## Code Quality Verification
- Format: `gofmt -w .` (expected: no diff after run) / Static analysis: none
  configured

## SPEC.md Compliance

### Success Criteria
| ID | Criterion | How to Verify |
|----|-----------|---------------|
| SC-1 | FR1–FR3 implemented and unit-tested | TS-1…TS-6 pass via `go test ./...` |
| SC-2 | Visual match with updated mockup | Manual comparison (below) |
| SC-3 | Code review completed | Review phase record |

### Functional Requirements Coverage
| Requirement | Tasks | Verification |
|-------------|-------|--------------|
| FR1 | task0002 | TS-1, TS-2, TS-6 |
| FR2 | task0002 | TS-4 |
| FR3 | task0001 | TS-5 |
| NFR1 | task0002 | TS-7 |
| NFR2 | task0002 | TS-3 |

## E2E Testing
None (unchanged policy).

## Manual Testing (E2E Not Possible)
- [ ] MT-1: モックとの目視照合 — render the built HTML and compare against
      `design/mockups/screen-email-digest.html` (states: rendered / plain /
      empty-body): sub-headings, lists, inline code, code blocks, links,
      cool palette throughout.
- [ ] MT-2: (Optional, post-merge, user-run) real delivery: next release
      mail arrives cool-toned with rendered Markdown.

## Performance / Security Verification (if applicable)
- NFR2: covered by TS-3.

## Verification Summary
| Category | Items | Automated | E2E | Manual |
|----------|-------|-----------|-----|--------|
| Build/Format | 2 | 2 | 0 | 0 |
| Test scenarios | 7 | 6 | 0 | 1 (TS-7) |
| Manual checks | 2 | 0 | 0 | 2 |
