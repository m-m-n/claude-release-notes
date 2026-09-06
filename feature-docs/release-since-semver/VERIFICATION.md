# Verification Document: release-since-semver

## Overview

**Feature**: release-since-semver
**SPEC.md**: `feature-docs/release-since-semver/SPEC.md`
**IMPLEMENTATION.md**: `feature-docs/release-since-semver/IMPLEMENTATION.md`

This document covers the INTEGRATED verification of the whole feature. Two
kinds of item appear here:

- **Confirmation items** (FR1–FR7, NFR1–NFR3): the behaviour and its tests
  already exist in the integration branch (commit `13d0ac2`,
  IMPLEMENTATION.md D1). Verification confirms that the existing
  implementation and the existing tests satisfy the requirement — it does not
  ask for new code or new tests.
- **Change items** (FR8): produced by task0001 in this feature.

Test scenario IDs TS-1–TS-9 are SPEC.md's. TS-10, TS-11 and TS-12 are defined
here to close the coverage of FR6, FR7 and NFR1, which SPEC.md leaves without
a scenario.

## Build Verification

- Command: `go build ./...`
- Expected: exit code 0, no errors

## Test Verification

- Command: `go test ./...`
- Expected: exit code 0, every package `ok` (no `FAIL`, no build failure)
- Coverage target: `workflow.yaml` defines no coverage threshold, so none is
  asserted. The coverage criterion for this feature is the Functional
  Requirements Coverage table below: every requirement maps to at least one
  passing verification item.

### Test Scenarios from SPEC.md

The "Executed by" column names the existing test that carries the scenario, so
a confirmation item is checked by locating that test and observing it pass —
not by writing a new one.

| ID | Scenario | Expected Result | Test Type | Executed by |
|----|----------|-----------------|-----------|-------------|
| TS-1 | Differential fetch against a mock listing containing drafts, prereleases and older entries | Only the semver-newer, non-draft, non-prerelease releases are returned, newest first | Unit | `internal/github` — `TestSince_NewerThanStored` |
| TS-2 | Stored version equals the head of the listing | Empty result, no error | Unit | `internal/github` — `TestSince_StoredVersionIsNewest` |
| TS-3 | Stored version absent from the listing | Only the versions comparing greater are returned; never the whole listing; no error | Unit | `internal/github` — `TestSince_StoredVersionDeletedFromListing` |
| TS-4 | Request boundedness of the differential fetch | The set of requested pages is exactly `[1]`, and the requested per-page value equals the code's upper-bound constant | Integration (mock HTTP server) | `internal/github` — `TestSince_ReadsOnlyFirstPage`, `TestSince_RequestsPerPageBound` |
| TS-5 | Listing contains unparseable tags (`nightly`, `v2.9`, `v2.5.0-rc1`) | Those entries are skipped, the remaining releases are still returned, no error | Unit | `internal/github` — `TestSince_UnparsableTagIsSkipped` |
| TS-6 | Table-driven parsing of tag names (boundaries: `v0.0.0`, signed components, empty components, wrong component count, missing leading `v`) | Each case parses or fails exactly as the parse rule states | Unit | `internal/github` — `TestParseVersion` |
| TS-7 | Table-driven comparison of two versions | Major outranks minor outranks patch; equal versions are not newer | Unit | `internal/github` — `TestIsNewer` |
| TS-8 | Error content for an unparseable stored version and for a non-2xx response | The first error names the offending value; the second carries the HTTP status and never the token value | Unit | `internal/github` — `TestSince_UnparsableStoredVersionIsError`, `TestNon2xxResponse_ErrorHasStatusNotToken` |
| TS-9 | The sixteen enumerated locations in `feature-docs/release-notes-fetcher/` after task0001 | None of the stale markers remain (`FetchNewer`, `per_page=30`, differential fetch described as paging, identity-match stop, absent-version-returns-everything, unbounded "all newer fetched"); each location conveys its canonical statement | Documentation | Textual check, see Manual Testing MT-1 / MT-2 |
| TS-10 | Draft/prerelease exclusion and ordering on the latest-release path | Drafts and prereleases never appear; the newest remaining entry is returned; a listing of only drafts/prereleases is an error | Unit | `internal/github` — `TestLatest_ReturnsNewestFiltered`, `TestLatest_NoNonFilteredReleaseIsError` |
| TS-11 | The caller-side doc comment for the differential fetch in `internal/app` | It states semver comparison, single-page boundedness, and that an empty result means "already up to date" | Documentation | Read-through, see Manual Testing MT-3 |
| TS-12 | Dependency boundary | The module's require block contains `gopkg.in/yaml.v3` and nothing else; no semver library was added | Documentation (file inspection) | Inspection of `go.mod` |

## Code Quality Verification

- Format: `gofmt -w .` — expected: running it leaves the working tree
  unchanged (no file is reformatted).
- Static analysis: `go vet ./...` — expected: exit code 0, no diagnostics.
  `workflow.yaml` defines no static-analysis command; this is the Go-standard
  check the feature's own record shows was run (IMPLEMENTATION.md Open
  Questions).

## SPEC.md Compliance

### Success Criteria

| ID | Criterion | How to Verify |
|----|-----------|---------------|
| SC-1 | All functional requirements (FR1–FR8) are implemented and tested | The Functional Requirements Coverage table below has no empty verification cell |
| SC-2 | All test scenarios (TS-1–TS-9) pass | `go test ./...` exits 0 for TS-1–TS-8; MT-1 / MT-2 pass for TS-9 |
| SC-3 | Request boundedness holds (1 request, page 1 only, per-page at the upper bound) | TS-4 |
| SC-4 | Security requirements are satisfied — no token value in any error message | TS-8 |
| SC-5 | The `internal/app` doc comment and the older `feature-docs/release-notes-fetcher/` documents match the current behaviour | TS-11 for the doc comment; TS-9 for the documents |
| SC-6 | Code review is completed | The review phase's record in `workflow.yaml` shows no residual critical/high finding |

### Functional Requirements Coverage

| Requirement | Tasks | Verification |
|-------------|-------|--------------|
| FR1 | (none — confirmation item, D1) | TS-1, TS-2, TS-3, TS-7 |
| FR2 | (none — confirmation item, D1) | TS-5, TS-6 |
| FR3 | (none — confirmation item, D1) | TS-5 |
| FR4 | (none — confirmation item, D1) | TS-8 |
| FR5 | (none — confirmation item, D1) | TS-4 |
| FR6 | (none — confirmation item, D1) | TS-1 (exclusion and order on the differential path), TS-2 (empty result is not an error), TS-10 (exclusion on the latest-release path) |
| FR7 | (none — confirmation item, D1) | TS-11 |
| FR8 | task0001 | TS-9 |
| NFR1 | (none — confirmation item, D1) | TS-12 |
| NFR2 | (none — confirmation item, D1) | TS-4 |
| NFR3 | (none — confirmation item, D1) | TS-8 |

An empty `Tasks` cell here is intentional and is not an uncovered-requirement
gap: those requirements were satisfied before this feature's implement phase
and are verified, not rebuilt (IMPLEMENTATION.md D1).

## Manual Testing (E2E Not Possible)

The project has no E2E test framework — `test/README.md` records None / N/A —
so there is no E2E section. The items below need either a textual search or
human judgement.

- [ ] MT-1 (TS-9): Run a case-insensitive search over the four modified files
      — `feature-docs/release-notes-fetcher/SPEC.md`, `REQUIREMENTS.md`,
      `tasks/task0003.md` — for the stale-marker set (`FetchNewer`,
      `per_page=30`, differential fetch described as following/walking
      pagination, stopping at an entry equal to the stored version, an absent
      stored version returning all listed releases, `全件取得` / "all fetched"
      / "paging as needed"). Expected: no hit.
- [ ] MT-2 (TS-9): Read each of the sixteen locations against task0001's
      target-location table and confirm it conveys the canonical statement
      listed for it, in the document's own language, with headings, section
      numbering and requirement IDs unchanged.
- [ ] MT-3 (TS-11): Read the release-fetcher doc comment in
      `internal/app/app.go` and confirm it states semver comparison,
      single-page boundedness, and that an empty result means "already up to
      date".
- [ ] MT-4: Confirm the statements that describe the latest-release call as
      walking pagination are unchanged (task0001 AC-5) — they are current
      behaviour, and "correcting" them would be a regression of the documents.
- [ ] MT-5: Confirm the change set contains only the four documentation files
      plus this feature's own `feature-docs/release-since-semver/**`. Any hit
      under `internal/`, `cmd/`, `test/`, `feature-docs/release-notes-fetcher/reviews/`
      or `.../phase-state/` is a failure.
- [ ] MT-6: Report — do not fix — any stale statement found outside the
      sixteen enumerated locations. The differential call's Shared Components
      row in `feature-docs/release-notes-fetcher/IMPLEMENTATION.md` is location
      #16 and is in scope, so it is not a candidate here.

## Performance / Security Verification

- NFR2 (bounded API usage): exactly one listing request per differential run —
  TS-4. No dedicated performance test exists; boundedness is the whole
  performance requirement.
- Per-run payload bound: at most the per-page cap of releases per run —
  covered by the per-page assertion in TS-4.
- NFR3 (no secret leakage): a non-2xx error carries the HTTP status and never
  the token value — TS-8, which asserts against a sentinel token value on both
  the differential and the latest-release paths.

## Verification Summary

| Category | Items | Automated | E2E | Manual |
|----------|-------|-----------|-----|--------|
| Build | 1 (`go build ./...`) | 1 | 0 | 0 |
| Test scenarios | 12 (TS-1–TS-12) | 9 (TS-1–TS-8, TS-10) | 0 | 3 (TS-9, TS-11, TS-12) |
| Code quality | 2 (`gofmt`, `go vet`) | 2 | 0 | 0 |
| Success criteria | 6 (SC-1–SC-6) | 4 (SC-1–SC-4) | 0 | 2 (SC-5, SC-6) |
| Manual checks | 6 (MT-1–MT-6) | 0 | 0 | 6 |
