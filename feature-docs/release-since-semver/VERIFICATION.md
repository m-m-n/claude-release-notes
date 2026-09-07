# Verification Document: release-since-semver

## Overview

**Feature**: release-since-semver
**SPEC.md**: `feature-docs/release-since-semver/SPEC.md`
**IMPLEMENTATION.md**: `feature-docs/release-since-semver/IMPLEMENTATION.md`

This document covers the INTEGRATED verification of the whole feature. Two
kinds of item appear here:

- **Confirmation items** (FR1–FR6, NFR1–NFR3): the behaviour and its tests
  already exist in the integration branch (commit `13d0ac2`,
  IMPLEMENTATION.md D1). Verification confirms that the existing
  implementation and the existing tests satisfy the requirement — it does not
  ask for new code or new tests.
- **Change items** (FR7, FR8, FR9, FR10, FR11, NFR4): produced by this
  feature's tasks. FR8 is already delivered by task0001 (merged); FR7, FR9,
  FR10 and NFR4 by task0002; FR11 by task0003.

FR7 was a confirmation item in the previous generation of this document and is
now a change item: the verify phase failed TS-11 because the caller-side doc
comment states single-page boundedness and the empty-result meaning but omits
the semver-comparison statement the requirement demands.

Test scenario IDs TS-1–TS-9 and TS-13–TS-16 are SPEC.md's. TS-10, TS-11 and
TS-12 were defined by this document to close the coverage of FR6, FR7 and
NFR1, which SPEC.md leaves without a scenario; they keep exactly the meanings
`workflow.yaml` already records for them.

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

The "Executed by" column names the test that carries the scenario. For a
confirmation item that is an existing test, checked by locating it and
observing it pass; for a change item it is the test the owning task writes,
whose name is the implementer's choice.

| ID | Scenario | Expected Result | Test Type | Executed by |
|----|----------|-----------------|-----------|-------------|
| TS-1 | Differential fetch against a mock listing containing drafts, prereleases and older entries | Only the semver-newer, non-draft, non-prerelease releases are returned, newest first | Unit | `internal/github` — `TestSince_NewerThanStored` (existing) |
| TS-2 | Stored version equals the head of the listing | Empty result, no error | Unit | `internal/github` — `TestSince_StoredVersionIsNewest` (existing) |
| TS-3 | Stored version absent from the listing | Only the versions comparing greater are returned; never the whole listing; no error | Unit | `internal/github` — `TestSince_StoredVersionDeletedFromListing` (existing) |
| TS-4 | Request boundedness of the differential fetch | The set of requested pages is exactly `[1]`, and the requested per-page value equals the code's upper-bound constant | Integration (mock HTTP server) | `internal/github` — `TestSince_ReadsOnlyFirstPage`, `TestSince_RequestsPerPageBound` (existing) |
| TS-5 | Listing contains unparseable tags (`nightly`, `v2.9`, `v2.5.0-rc1`) | Those entries are skipped, the remaining releases are still returned, no error | Unit | `internal/github` — `TestSince_UnparsableTagIsSkipped` (existing) |
| TS-6 | Table-driven parsing of tag names (boundaries: `v0.0.0`, signed components, empty components, wrong component count, missing leading `v`) | Each case parses or fails exactly as the parse rule states | Unit | `internal/github` — `TestParseVersion` (existing) |
| TS-7 | Table-driven comparison of two versions | Major outranks minor outranks patch; equal versions are not newer | Unit | `internal/github` — `TestIsNewer` (existing) |
| TS-8 | Error content for an unparseable stored version and for a non-2xx response | The first error names the offending value; the second carries the HTTP status and never the token value | Unit | `internal/github` — `TestSince_UnparsableStoredVersionIsError`, `TestNon2xxResponse_ErrorHasStatusNotToken` (existing) |
| TS-9 | The sixteen enumerated locations in `feature-docs/release-notes-fetcher/` after task0001 | None of the stale markers remain (`FetchNewer`, `per_page=30`, differential fetch described as paging, identity-match stop, absent-version-returns-everything, unbounded "all newer fetched"); each location conveys its canonical statement | Documentation | Textual check, see Manual Testing MT-1 / MT-2 |
| TS-10 | Draft/prerelease exclusion and ordering on the latest-release path | Drafts and prereleases never appear; the newest remaining entry is returned; a listing of only drafts/prereleases is an error | Unit | `internal/github` — `TestLatest_ReturnsNewestFiltered`, `TestLatest_NoNonFilteredReleaseIsError` (existing; still pass after FR10 narrows the filter) |
| TS-11 | The caller-side doc comment for the differential fetch in `internal/app` | It states semver comparison, single-page boundedness, and that an empty result means "already up to date" | Documentation | Read-through after task0002, see Manual Testing MT-3 |
| TS-12 | Dependency boundary | The module's require block contains `gopkg.in/yaml.v3` and nothing else; no semver library was added | Documentation (file inspection) | Inspection of `go.mod` |
| TS-13 | A pipeline run whose delivered set has a head that is not its semver maximum — the listing `v2.1.244`, `v2.2.0` in that order against a stored `v2.0.0` | The value handed to the state store is the `v2.2.0` tag name, and the items handed to the mail builder are still in the order the differential query returned | Integration (in-process mocks) | `internal/app/app_test.go` — new in task0002 (task0002 AC-3) |
| TS-14 | Latest-release selection over a listing whose head is a non-draft, non-prerelease release with an unparseable tag, and over a listing with no parseable candidate at all | The first case returns the next parseable release; the second returns the existing "no releases found" error | Unit | `internal/github/client_test.go` — new in task0002 (task0002 AC-2) |
| TS-15 | The twenty enumerated locations in `feature-docs/release-notes-fetcher/` after task0003 | No statement assuming "the stored value is the head of the listing" or "the latest-release query unconditionally selects the head" remains; each location conveys its canonical statement (CB9 / CB10) | Documentation | Textual check, see Manual Testing MT-7 / MT-8 |
| TS-16 | The exported semver-maximum selector, table-driven (maximum at the head, maximum at the tail, a single element, equal versions, an unparseable input ignored, an empty input), plus the absence of any duplicate parsing or comparison implementation in `internal/app` | Each table case selects exactly as the contract states and the empty case reports nothing found; `internal/app` holds no parsing or comparison of its own and exactly one call site into the selector | Unit + Documentation (textual) | `internal/github/client_test.go` — new in task0002 (task0002 AC-1); the textual half is Manual Testing MT-9 |

## Code Quality Verification

- Format: `gofmt -w .` — expected: running it leaves the working tree
  unchanged (no file is reformatted).
- Static analysis: `go vet ./...` — expected: exit code 0, no diagnostics.
  `workflow.yaml` defines no static-analysis command; this is the Go-standard
  check the feature's own record shows was run (IMPLEMENTATION.md Open
  Questions).

## SPEC.md Compliance

### Success Criteria

These follow SPEC.md's own Success Criteria list, in its order. The SC IDs are
local to this document.

| ID | Criterion | How to Verify |
|----|-----------|---------------|
| SC-1 | All functional requirements (FR1–FR11) are implemented and tested | The Functional Requirements Coverage table below has no empty verification cell |
| SC-2 | All test scenarios (TS-1–TS-16) pass | `go test ./...` exits 0 for TS-1–TS-8, TS-10, TS-13, TS-14 and TS-16's table half; the manual items pass for TS-9, TS-11, TS-12, TS-15 and TS-16's textual half |
| SC-3 | Request boundedness holds (1 request, page 1 only, per-page at the upper bound) | TS-4 |
| SC-4 | Security requirements are satisfied — no token value in any error message | TS-8 |
| SC-5 | The persisted `last_version` is the semver maximum of what was delivered, and every value this tool writes parses as `vMAJOR.MINOR.PATCH` | TS-13 for the differential path; TS-14 for the first-run path, which is the only other place a value reaches the store |
| SC-6 | The semver comparison rule has exactly one definition site, in `internal/github` | TS-16, both halves |
| SC-7 | The `internal/app` doc comment and the older `feature-docs/release-notes-fetcher/` documents match the current behaviour | TS-11 for the doc comment; TS-9 and TS-15 for the documents |
| SC-8 | Code review is completed | The review phase's record in `workflow.yaml` shows no residual critical/high finding |

### Functional Requirements Coverage

| Requirement | Tasks | Verification |
|-------------|-------|--------------|
| FR1 | (none — confirmation item, D1) | TS-1, TS-2, TS-3, TS-7 |
| FR2 | (none — confirmation item, D1) | TS-5, TS-6 |
| FR3 | (none — confirmation item, D1) | TS-5 |
| FR4 | (none — confirmation item, D1) | TS-8 |
| FR5 | (none — confirmation item, D1) | TS-4 |
| FR6 | (none — confirmation item, D1) | TS-1 (exclusion and order on the differential path), TS-2 (empty result is not an error), TS-10 (exclusion on the latest-release path) |
| FR7 | task0002 | TS-11 |
| FR8 | task0001 | TS-9 |
| FR9 | task0002 | TS-13 |
| FR10 | task0002 | TS-14 |
| FR11 | task0003 | TS-15 |
| NFR1 | (none — confirmation item, D1) | TS-12 |
| NFR2 | (none — confirmation item, D1) | TS-4 |
| NFR3 | (none — confirmation item, D1) | TS-8 |
| NFR4 | task0002 | TS-16 |

An empty `Tasks` cell here is intentional and is not an uncovered-requirement
gap: those requirements were satisfied before this feature's implement phase
and are verified, not rebuilt (IMPLEMENTATION.md D1).

## Manual Testing (E2E Not Possible)

The project has no E2E test framework — `test/README.md` records None / N/A —
so there is no E2E section. The items below need either a textual search or
human judgement.

- [ ] MT-1 (TS-9): Run a case-insensitive search over the four files task0001
      modified — `feature-docs/release-notes-fetcher/SPEC.md`,
      `REQUIREMENTS.md`, `tasks/task0003.md`, `IMPLEMENTATION.md` — for the
      stale-marker set (`FetchNewer`, `per_page=30`, differential fetch
      described as following/walking pagination, stopping at an entry equal to
      the stored version, an absent stored version returning all listed
      releases, `全件取得` / "all fetched" / "paging as needed"). Expected: no
      hit.
- [ ] MT-2 (TS-9): Read each of the sixteen locations against task0001's
      target-location table and confirm it conveys the canonical statement
      listed for it, in the document's own language, with headings, section
      numbering and requirement IDs unchanged.
- [ ] MT-3 (TS-11): Read the release-fetcher doc comment in
      `internal/app/app.go` and confirm it states all three of semver
      comparison, single-page boundedness, and that an empty result means
      "already up to date". This is the item the previous verify run failed;
      the comparison statement is the one that was missing.
- [ ] MT-4: Confirm the statements that describe the latest-release call as
      **walking pagination** are unchanged — the pagination claim only. Its
      **selection rule** is deliberately changed by FR10 in two places that
      task0001 had preserved (IMPLEMENTATION.md D8, task0003 L15 and L19); a
      diff there is expected, not a regression.
- [ ] MT-5: Confirm the change set contains only the four older documentation
      files, the four Go files task0002 declares, and this feature's own
      `feature-docs/release-since-semver/**`. Any hit under `cmd/`, `test/`,
      `feature-docs/release-notes-fetcher/reviews/` or `.../phase-state/` is a
      failure.
- [ ] MT-6: Report — do not fix — any stale statement found outside the
      sixteen FR8 locations and the twenty FR11 locations. Two are already
      known and recorded in IMPLEMENTATION.md Open Questions.
- [ ] MT-7 (TS-15): Run a search over the same four older documents for the
      FR9/FR10 stale-marker set (task0003 AC-5): the stored value described as
      the newest or first release of the listing or of the result; the
      Japanese phrasings 最新バージョン / 取得済み最新リリース used for the
      stored value without the semver-greatest qualification; the
      latest-release query described as returning the newest non-draft,
      non-prerelease release without the parseability qualification; the
      Japanese 最新の 1 リリース / 最新 1 件 without it; and the data-flow
      label that writes the newest version. Expected: no hit. The Japanese
      markers need reading in context — 最新 is legitimate elsewhere in those
      documents, for example in the mail-ordering requirement.
- [ ] MT-8 (TS-15): Read each of the twenty locations against task0003's
      target-location table and confirm it conveys CB9 or CB10 as listed, in
      the document's own language, with headings, section numbering and IDs
      unchanged.
- [ ] MT-9 (TS-16, textual half): Confirm `internal/app` contains no
      `vMAJOR.MINOR.PATCH` parsing and no version comparison of its own — in
      production code or in its test file — that it reaches the rule through
      exactly one call site into `internal/github`'s exported selector, and
      that the version type and the parse and compare helpers in
      `internal/github` are still unexported.

## Performance / Security Verification

- NFR2 (bounded API usage): exactly one listing request per differential run —
  TS-4. FR10 does not change the latest-release path's pagination walk, so
  that path remains outside NFR2's scope as before. No dedicated performance
  test exists; boundedness is the whole performance requirement.
- Per-run payload bound: at most the per-page cap of releases per run —
  covered by the per-page assertion in TS-4.
- NFR3 (no secret leakage): a non-2xx error carries the HTTP status and never
  the token value — TS-8, which asserts against a sentinel token value on both
  the differential and the latest-release paths.
- Input validation on the write path: FR10 means every value this tool
  persists satisfies FR2's parsing rule, so the tool never creates FR4's error
  state on its own — SC-5.

## Verification Summary

| Category | Items | Automated | E2E | Manual |
|----------|-------|-----------|-----|--------|
| Build | 1 (`go build ./...`) | 1 | 0 | 0 |
| Test scenarios | 16 (TS-1–TS-16) | 12 (TS-1–TS-8, TS-10, TS-13, TS-14, TS-16 table half) | 0 | 4 (TS-9, TS-11, TS-12, TS-15) plus TS-16's textual half |
| Code quality | 2 (`gofmt`, `go vet`) | 2 | 0 | 0 |
| Success criteria | 8 (SC-1–SC-8) | 3 (SC-3, SC-4, SC-5) | 0 | 5 (SC-1, SC-2, SC-6, SC-7, SC-8 — each resolves partly or wholly to a review or textual check) |
| Manual checks | 9 (MT-1–MT-9) | 0 | 0 | 9 |
