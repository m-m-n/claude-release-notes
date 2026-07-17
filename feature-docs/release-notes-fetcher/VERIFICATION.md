# Verification Document: Claude Code Release Notes Fetcher

## Overview
**Feature**: release-notes-fetcher / **SPEC.md**:
`feature-docs/release-notes-fetcher/SPEC.md` / **IMPLEMENTATION.md**:
`feature-docs/release-notes-fetcher/IMPLEMENTATION.md`

## Build Verification
- Command: `go build ./...`
- Expected: exit code 0, no errors

## Test Verification
- Command: `go test ./...`
- Coverage target: no numeric floor; every TS scenario below covered

### Test Scenarios from SPEC.md
| ID | Scenario | Expected Result | Test Type |
|----|----------|-----------------|-----------|
| TS-1 | Stored version + mocked paged API listing | All newer releases returned, drafts/prereleases skipped, newest first | Unit |
| TS-2 | First run (no state file) | Only the latest release processed; state written atomically with dirs created | Unit |
| TS-3 | Config parsing and validation | Valid config parses; each missing/empty key errors naming the key, value never printed | Unit |
| TS-4 | Translation round trip | Prompt carries delimiters + all bodies; output split back per release; non-zero exit / empty / undelimitable output error | Unit |
| TS-5 | Digest build | Every release present, newest first, version + date + body; correct MIME headers and UTF-8 subject | Unit |
| TS-6 | Stage failure abort | Failure injected at fetch/translate/send → error exit path, state save never invoked | Unit |
| TS-7 | No new releases | No translation, no email, no state change, success exit | Unit |
| TS-8 | systemd unit content | Timer has `OnCalendar=*-*-* 06,12,18:00:00 Asia/Tokyo` and `Persistent=true`; service is oneshot and referenced by the timer | Manual (file inspection) |
| TS-9 | Secret non-leakage | Sentinel app password / token values never appear in any error or log output on failure paths | Unit |
| TS-10 | Dependency minimality | go.mod requires only gopkg.in/yaml.v3 beyond the standard library | Manual (file inspection) |
| TS-11 | Logging streams | Progress on stdout, errors on stderr; printed timestamps carry an explicit timezone | Unit |

## Code Quality Verification
- Format: `gofmt -w .` (expected: no diff after run) / Static analysis: none
  configured

## SPEC.md Compliance

### Success Criteria
| ID | Criterion | How to Verify |
|----|-----------|---------------|
| SC-1 | FR1–FR6 implemented and unit-tested | TS-1…TS-7 pass via `go test ./...` |
| SC-2 | No secret leakage in output | TS-9 |
| SC-3 | README documents config, Gmail app password, systemd install | Manual read of README.md |
| SC-4 | Digest visuals follow the design step | Manual comparison (below) |

### Functional Requirements Coverage
| Requirement | Tasks | Verification |
|-------------|-------|--------------|
| FR1 | task0003 | TS-1, TS-7 |
| FR2 | task0002, task0006 | TS-2, TS-7 |
| FR3 | task0001 | TS-3 |
| FR4 | task0004 | TS-4 |
| FR5 | task0005 | TS-5 |
| FR6 | task0007 | TS-8 |
| NFR1 | task0006 | TS-10 |
| NFR2 | task0001, task0005 | TS-9 |
| NFR3 | task0006 | TS-11 |
| NFR4 | task0006 | TS-6 |

## E2E Testing
None (no E2E infrastructure; external systems are mocked — test/README.md).

## Manual Testing (E2E Not Possible)
- [ ] MT-1: モックとの目視照合 — render the built HTML (from TS-5's build
      output or a sample invocation) and compare against
      `design/mockups/screen-email-digest.html` (states: default / single /
      empty-body): header band, per-release sections, divider rules, footer,
      colors and sizes per design tokens.
- [ ] MT-2: README walkthrough — following README.md alone, a user can
      configure the tool and install the timer (read-through check).
- [ ] MT-3: (Optional, post-merge, user-run) real end-to-end delivery with a
      live config: first run delivers the latest release and creates the
      state file.

## Performance / Security Verification (if applicable)
- NFR2: covered by TS-9 and the README chmod 600 instruction (task0007).

## Verification Summary
| Category | Items | Automated | E2E | Manual |
|----------|-------|-----------|-----|--------|
| Build/Format | 2 | 2 | 0 | 0 |
| Test scenarios | 11 | 9 | 0 | 2 (TS-8, TS-10) |
| Manual checks | 3 | 0 | 0 | 3 |
