# Feature: release-since-semver

## Overview

`(*Client).Since(version string) ([]Release, error)` in `internal/github` decides which releases are "new" by comparing semantic versions of tag names against the saved version, instead of locating the saved version inside the listing by identity match. It reads only the first page of the release listing, so a single run is bounded regardless of how old the saved version is. The value written back to `state.json` after a successful send is the semver-greatest release delivered in that run, and the latest-release path refuses to select a tag that does not parse, so the persisted baseline always satisfies the parsing rule the read path requires. This specification also covers bringing the stale statements in the older `feature-docs/release-notes-fetcher/` documents in line with the current behaviour.

Requirements document: `feature-docs/release-since-semver/REQUIREMENTS.md`.

## Objectives

- Keep incremental delivery working even when the saved version has been removed from upstream, so that only genuinely newer releases are delivered.
- Bound the number of releases a single run handles, keeping the translation and mail payload bounded.
- Make `feature-docs/release-notes-fetcher/` agree with the current behaviour, removing the divergence between docs and implementation.
- Ensure an already-delivered version is never delivered again.
- Guarantee on the write path that the value stored in `state.json` always satisfies FR2's parsing rule.
- Keep the `vMAJOR.MINOR.PATCH` comparison rule to a single definition site in `internal/github`.

## User Stories

### US1: Bounded incremental delivery based on semver comparison
As a recipient of the release-notes mail, I want only releases newer than the last delivered one, so that I never receive an unbounded re-delivery when the saved version disappears from the upstream listing.

**Acceptance Criteria:**
- [ ] AC-1: Only releases newer than the saved version by semver comparison are returned, newest first.
- [ ] AC-2: Drafts and prereleases appear in neither `Since` nor `Latest` results.
- [ ] AC-3: When the saved version equals the head of the listing, an empty slice is returned.
- [ ] AC-4: When the saved version has been removed from the listing, the run is still a success path and only the versions that compare greater are returned (not the whole listing).
- [ ] AC-5: Tags that cannot be parsed are skipped and the retrieval of other releases continues.
- [ ] AC-6: However old the saved version is, the only page requested is `page=1`, and `per_page` equals the upper bound constant in the code.
- [ ] AC-7: An unparseable saved version produces an error whose message names the offending value.
- [ ] AC-8: A non-2xx response produces an error that carries the HTTP status and does not carry the token value.
- [ ] AC-10 (FR9): Even when the head of the result is not the semver maximum — a listing of `[v2.1.244, v2.2.0]` against a saved `v2.0.0` — the persisted `last_version` is `v2.2.0`, and the next run does not deliver `v2.2.0` again.
- [ ] AC-11 (FR10): Even when a non-draft, non-prerelease release with an unparseable tag sits at the head of the listing, `Latest` does not return it and returns the next parseable release instead. When no parseable delivery candidate exists, the existing "no releases found" error is returned.
- [ ] AC-13 (NFR4): `internal/app` contains no `vMAJOR.MINOR.PATCH` parsing or comparison code, and the semver-maximum selection happens at a single call site into `internal/github`'s exported entry point. `version`, `parseVersion` and `isNewer` remain unexported.

### US2: Documentation that matches the current behaviour
As a maintainer of this repository, I want the older `feature-docs/release-notes-fetcher/` documents to describe the current behaviour, so that reading them does not lead to the paging-based or identity-match-based understanding.

**Acceptance Criteria:**
- [ ] AC-9: The 16 identified locations in `feature-docs/release-notes-fetcher/` are updated to the current behaviour, with no remaining statements assuming paging, identity match, `FetchNewer`, or `per_page=30`.
- [ ] AC-12 (FR11): The affected locations in the older `feature-docs/` agree with the behaviour established by FR9 and FR10.

## Technical Requirements

### Functional Requirements

- **FR1 — Newness decided by semantic version comparison:** `Since(version)` decides "newer" by comparing the semantic version of tag names, not by locating the saved version inside the listing by identity match. Comparison proceeds major → minor → patch, and only strictly greater versions count as newer. A version equal to the saved version is not included.
- **FR2 — Tag parsing rules:** The parse target is `vMAJOR.MINOR.PATCH`. The leading `v` is optional. The value must have exactly three dot-separated components, each an unsigned decimal integer. Values with a prerelease or build suffix (e.g. `v2.5.0-rc1`) and values with a different component count (`v2.9`, `v2.1.2.3`) are unparseable.
- **FR3 — Unparseable tags are skipped:** Unparseable tags in the listing (e.g. `nightly`, `v2.9`, `v2.5.0-rc1`) are treated as "not newer" and skipped; the run is not aborted.
- **FR4 — An unparseable saved version is an error:** When the saved version in `state.json` is unparseable, there is no basis for comparison, so `Since` returns an error. The error message names the offending value. There is no fallback to delivering everything. This read-side contract is matched on the write side by FR10, which keeps every value this tool persists within FR2's parsing rule.
- **FR5 — Bounded to a single page:** `Since` fetches only the first page (`per_page=10`) and does not walk pagination. However old the saved version is, a single run returns at most 10 releases. Newer releases sitting on page 2 and beyond are left behind; this is accepted as a permanent design decision favouring boundedness.
- **FR6 — Draft/prerelease exclusion and ordering:** Drafts and prereleases are included in neither the `Since` nor the `Latest` result. `Latest` additionally excludes releases whose tag does not parse (FR10). The result preserves listing order (newest first). When nothing matches, an empty slice is returned rather than an error.
- **FR7 — Caller doc comment consistency:** The doc comment on `ReleaseFetcher.Since` in `internal/app` correctly states semver comparison, single-page boundedness, and that an empty slice means "already up to date".
- **FR8 — Update the older `feature-docs/release-notes-fetcher/` documents:** Rewrite the 16 locations identified by the reference-impact scan to the current behaviour. The targets are: in `SPEC.md`, the US1 acceptance criterion ("paging as needed"), FR1, Data Flow (`FetchNewer`), API Design (`per_page`), TS-1, and Edge Cases; in `REQUIREMENTS.md`, the UC01 basic flow, F01, 8.2, the 10.1 risk mitigation, and 11.1; in `tasks/task0003.md`, Design 5, AC-1, AC-4, and Test Notes; in `IMPLEMENTATION.md`, the Shared Components row for the differential call. `Latest` still walks pagination, so the "pagination" statement under `tasks/task0003.md`'s Files to Create is not stale and is out of scope.
- **FR9 — The persisted baseline is the semver maximum of what was delivered:** The `last_version` written back to `state.json` after a successful send is the semver-greatest tag name among the releases delivered in that run — not the head of the result array (listing order). When nothing was delivered, nothing is persisted; that existing contract is unchanged. `internal/app` does not decide "semver-greatest" itself: it calls the exported entry point `internal/github` provides under NFR4.
- **FR10 — Unparseable tags are excluded from latest-release selection:** In addition to drafts and prereleases, `Latest` excludes releases whose tag name does not parse as `vMAJOR.MINOR.PATCH` — the same rule the differential path already applies under FR3. A listing with no parseable delivery candidate falls through to the existing "no releases found" error. As a result, the value a first run writes to `state.json` always satisfies FR2's parsing rule, so this tool never creates FR4's error state without manual intervention.
- **FR11 — Update the older `feature-docs/` statements that FR9 and FR10 make stale:** Statements in the older `feature-docs/release-notes-fetcher/` documents that disagree with the behaviour established by FR9 and FR10 are updated, following the already-settled reconciliation policy of updating only the stale parts. Two kinds of statement are in scope: those assuming the persisted value is the head of the listing, and those assuming `Latest` unconditionally selects the head. This is a set disjoint from FR8's 16 locations, identified by the reference-impact scan; FR8's locations are already complete and are not rewritten again.

### Non-Functional Requirements

- **NFR1 - Dependency boundary:** The semver comparison is implemented in-house using only the Go standard library. No external semver library is added (`go.mod`'s require stays `gopkg.in/yaml.v3` only).
- **NFR2 - Bounded GitHub API usage:** The `Since` path issues exactly one GitHub API request per run. FR10 does not change `Latest`'s pagination walk, so `Latest` remains outside NFR2's scope as before.
- **NFR3 - Errors that do not leak secrets:** An error for a non-2xx response carries the HTTP status and does not carry the token value.
- **NFR4 - A single definition site for the semver comparison rule:** The rules for parsing and comparing `vMAJOR.MINOR.PATCH` exist only in `internal/github`. `internal/github` gains a newly exported entry point that takes a set of releases and returns the semver-greatest one, and `internal/app` (FR9) calls it. `version`, `parseVersion` and `isNewer` stay unexported behind that entry point. `internal/app` neither re-parses tag names nor re-states the comparison rule.

## Implementation Approach

### Architecture

**System Architecture:**
```
┌─────────────────────────────────────┐
│  internal/app (Run)                 │  pipeline orchestration
├─────────────────────────────────────┤
│  internal/github (Client)           │  Since / Latest
│    exported semver-max selector     │  the single definition site (NFR4)
│    version, parseVersion, isNewer   │  unexported helpers behind it
├─────────────────────────────────────┤
│  GitHub Releases API                │  external
└─────────────────────────────────────┘
        ↑ state.json ($XDG_STATE_HOME/claude-release-notes/)
```

**Component Diagram:**
```
Run (internal/app)
  ├─ config     : loads configuration
  ├─ state      : writes {"last_version": <semver-max of delivered>}   (FR9)
  ├─ github     : (*Client).Since(version) -> []Release
  │                 exported semver-max selector([]Release) -> Release (NFR4)
  │                 parseVersion(tag) (version, bool)      [unexported]
  │                 isNewer(a, b version) bool             [unexported]
  │               (*Client).Latest() -> walks pagination (unchanged),
  │                 skipping draft / prerelease / unparseable tags (FR10)
  ├─ translate  : translates the selected releases
  └─ mail       : sends the digest mail
```

`version` is a struct of major/minor/patch. `parseVersion` and `isNewer` remain unexported and use only `strconv` and `strings` from the Go standard library (NFR1); the comparison rule they implement is reachable from `internal/app` only through the exported entry point `internal/github` provides (NFR4), never by re-parsing tag names outside the package.

Notes on that exported entry point:

- It is a package-level function in `internal/github` taking `[]Release` — a pure function, not a method on `*Client`, since the selection needs no HTTP. The exact name and signature are left to create-plan.
- `internal/app` already imports `internal/github` for the `github.Release` type, so calling the entry point adds no new import edge.
- The entry point is not called when nothing was delivered, because `Run` returns early on an empty release set; how it behaves for an empty input is a create-plan design choice.

### Data Flow

```
config → state (last_version) → github.Since(last_version) → translate → mail
       → state save (semver-max of the delivered releases, FR9)
```

```
Since(version)
  parseVersion(version) ──not ok──▶ error naming the offending value (FR4)
        │ ok
        ▼
  GET /releases?per_page=10&page=1        (one request only, FR5/NFR2)
        │
        ▼
  for each release in listing order (newest first)
        ├─ draft or prerelease  ──▶ skip                    (FR6)
        ├─ parseVersion(tag) not ok ──▶ skip                (FR3)
        ├─ not isNewer(tag, saved)  ──▶ skip                (FR1)
        └─ otherwise            ──▶ append
        │
        ▼
  []Release in listing order, empty slice when nothing matches (FR6)
```

```
Latest()
  for each release in listing order (newest first)
        ├─ draft or prerelease      ──▶ skip                (FR6)
        ├─ parseVersion(tag) not ok ──▶ skip                (FR10)
        └─ otherwise                ──▶ return it
        │
        ▼
  no parseable candidate ──▶ existing "no releases found" error (FR10)
```

The state save happens only after the mail is sent successfully; that is an existing contract of the pipeline and is unchanged by this feature. What the save writes does change: it is the semver-greatest delivered tag name, obtained from `internal/github`'s exported entry point (FR9, NFR4).

### API Design

#### Endpoint 1: List releases (first page only)

**Request:**
```
Method: GET
Path: https://api.github.com/repos/anthropics/claude-code/releases?per_page=10&page=1
Headers:
  - Authorization: Bearer {token}
  - Accept: application/vnd.github+json
Body: none
```

`Since` issues this request exactly once and never requests `page=2` or beyond (FR5, NFR2). `Latest` still walks pagination and is not changed by this feature; only its selection rule changes (FR10).

**Response:**
```json
[
  {
    "tag_name": "v2.1.245",
    "draft": false,
    "prerelease": false,
    "body": "..."
  }
]
```

**Error Response:**

A non-2xx response is surfaced as a Go `error` carrying the HTTP status; the token value is never included in the message (NFR3).

### Database Schema

No database. The only persisted state is a file.

#### State file: `$XDG_STATE_HOME/claude-release-notes/state.json`

| Field | Type | Null | Default | Description |
|--------|------|------|---------|-------------|
| last_version | string | NO | - | The semver-greatest tag name among the releases delivered in the run (FR9); must be `vMAJOR.MINOR.PATCH` |

```json
{"last_version": "v2.1.243"}
```

The file holds exactly one value and is overwritten after each successful send.

### Dependencies

**Internal Dependencies:**
- `internal/app`: calls `(*Client).Since` in its `Run` pipeline (config → state → github → translate → mail → state save), calls `internal/github`'s exported semver-max entry point to decide the value it persists (FR9, NFR4), and carries the doc comment covered by FR7.
- `internal/github`: owns `Since`, `Latest`, the exported semver-max selector that is the single definition site of the comparison rule (NFR4), and the unexported `version` / `parseVersion` / `isNewer` helpers behind it.

**External Dependencies:**
- Go standard library (`strconv`, `strings`): version parsing and comparison.
- `gopkg.in/yaml.v3`: the only entry in `go.mod`'s require block; no semver library is added (NFR1).
- GitHub Releases API: source of the release listing.

### File Structure

```
internal/
├── github/
│   ├── client.go            # Since, Latest, exported semver-max selector (NFR4),
│   │                        #   unexported version, parseVersion, isNewer
│   └── client_test.go       # TS-1..TS-8, TS-14, TS-16
└── app/
    └── app.go               # Run pipeline; persists the semver-max delivered tag (FR9);
                             #   ReleaseFetcher.Since doc comment (FR7)

feature-docs/
└── release-notes-fetcher/   # FR8 and FR11 update targets
    ├── SPEC.md
    ├── REQUIREMENTS.md
    ├── IMPLEMENTATION.md
    └── tasks/task0003.md
```

## Declared Change Set

This section states the create-plan derivation instead of a hand-authored
list: the feature-specific paths above are derived at create-plan from
every task's `files` entries in `workflow.yaml`
(`references/phases/create-plan-phase.md`).

Every SPEC declares, by default, the following two workflow-generated
entries in addition to the feature-specific paths above:

- `feature-docs/release-since-semver/**`
- `test-docs/release-since-semver/**`

`feature-docs/release-since-semver/**` covers `REQUIREMENTS.md`, `SPEC.md`,
`IMPLEMENTATION.md`, `workflow.yaml`, `phase-state/`, `tasks/`,
`reviews/roundN.yaml`, `VERIFICATION.md`, `retrospect.yaml`, and the design
artifacts the design step produces. These are generated and owned by the
phase documents and by `references/phase-state.md`; this section cites them
and restates none of their rules.

`test-docs/release-since-semver/**` covers
`test-docs/release-since-semver/{T}.tests.yaml`, the per-task test record.
It is generated and owned by `implement-phase.md`; this section cites it and
restates none of its rules.

These two default entries are part of the declaration unless the SPEC
author explicitly removes them; their absence is never assumed by
silence — removal is a deliberate, explicit narrowing.

This declaration is a SUPERSET assertion: the actual change set observed
at verification time must be CONTAINED IN the declared set, not equal to
it. A feature that produces no implement tasks generates no
`test-docs/release-since-semver/` directory at all; the declared
`test-docs/release-since-semver/**` entry is still correct in that case — a
declared path that never materializes is not a violation.

## Test Scenarios

### Unit Tests
- [ ] TS-1 (FR1, AC-1): Against a mock listing, only the semver-newer releases are returned, newest first.
- [ ] TS-2 (FR1, AC-3): When the saved version is the newest, an empty slice is returned.
- [ ] TS-3 (FR1, AC-4): When the saved version is absent from the listing, only the versions that compare greater are returned.
- [ ] TS-5 (FR2, FR3, AC-5): `nightly`, `v2.9`, `v2.5.0-rc1` and similar tags are skipped.
- [ ] TS-6 (FR2): Table-driven test of `parseVersion` (boundaries: `v0.0.0`, signed components, empty components, wrong component count).
- [ ] TS-7 (FR1): Table-driven test of `isNewer` (major > minor > patch precedence; equal versions yield false).
- [ ] TS-8 (FR4, NFR3, AC-7, AC-8): Error content for an unparseable saved version and for a non-2xx response.
- [ ] TS-14 (FR10, AC-11): `Latest` skips unparseable tags, and a listing with no parseable delivery candidate produces the "no releases found" error.
- [ ] TS-16 (NFR4, AC-13): Table-driven test of `internal/github`'s new exported entry point (maximum at the head, maximum at the tail, a single element, mixed equal values), together with a check that no duplicate parsing or comparison implementation exists in `internal/app`.

### Integration Tests
- [ ] TS-4 (FR5, AC-6): The set of requested pages is exactly `[1]`, and `per_page` equals the upper bound constant.
- [ ] TS-13 (FR9, AC-10): Against `internal/app`'s `Run`, for a case where `Since` returns an ordering whose head is not the semver maximum, the value handed to `State.Save` is the semver maximum.

### E2E Tests
**Existing E2E tests**: None
**Run command**: Not detected
- [ ] Existing E2E tests pass without regression

### Documentation Tests
- [ ] TS-9 (FR8, AC-9): None of the 16 locations in the older `feature-docs/release-notes-fetcher/` retain the previous behaviour's statements.
- [ ] TS-15 (FR11, AC-12): No statement assuming "the persisted value is the head of the listing" or "`Latest` unconditionally selects the head" remains in the older `feature-docs/`.

New scenarios are numbered from TS-13 onward: TS-10, TS-11 and TS-12 are already assigned in `workflow.yaml` to FR6, FR7 and NFR1.

### Edge Cases
- [ ] EC-1 (FR1, AC-4): The saved version has been removed from the listing — a success path; only versions that compare greater are returned, never the whole listing.
- [ ] EC-2 (FR6, AC-3): Nothing matches (the saved version equals the head, or every entry is a draft/prerelease) — an empty slice is returned, not an error.
- [ ] EC-3 (FR2, FR3, AC-5): The listing contains unparseable tags — they are skipped and the run continues.
- [ ] EC-4 (FR4, AC-7): The saved version is unparseable — `Since` returns an error naming the offending value, with no fallback to delivering everything.
- [ ] EC-5 (FR4): `state.json` is corrupt — recovery is not automated; the documented manual routes are (1) edit `last_version` in `state.json` to a valid `vMAJOR.MINOR.PATCH` by hand, or (2) delete `state.json` to return to first-run handling, in which only the newest parseable release is delivered.
- [ ] EC-6 (FR10, AC-11): The listing's head is a non-draft, non-prerelease release with an unparseable tag — `Latest` skips it and returns the next parseable release; if no parseable candidate exists at all, the existing "no releases found" error is returned.
- [ ] EC-7 (FR9, AC-10): The delivered set's head is not its semver maximum — the persisted `last_version` is the semver maximum, while the delivered result itself keeps listing order.

### Performance Tests
No dedicated performance test. Request boundedness is covered by TS-4 (`page=1` only, `per_page` at the upper bound).

## Security Considerations

- **Authentication:** `Authorization: Bearer {token}` on the GitHub API request.
- **Authorization:** Read-only access to the public `anthropics/claude-code` release listing.
- **Input Validation:** Tag names and the saved version are validated by `parseVersion` (exactly three unsigned decimal components, optional leading `v`, no suffix). Unparseable listing tags are skipped on the differential path (FR3) and excluded from latest-release selection (FR10); an unparseable saved version is an error (FR4).
- **Data Protection:** Error messages for non-2xx responses carry the HTTP status and never the token value (NFR3).
- **XSS Prevention:** Not applicable; this feature produces no markup.
- **SQL Injection Prevention:** Not applicable; no database.
- **CSRF Protection:** Not applicable; no browser-facing endpoint.

## Error Handling

### Error Codes

| Code | Description | HTTP Status | User Message |
|------|-------------|-------------|--------------|
| ERR_UNPARSEABLE_SAVED_VERSION | The saved version in `state.json` cannot be parsed as `vMAJOR.MINOR.PATCH`, so there is no basis for comparison (FR4, AC-7) | - | The error names the offending value; delivery stops and nothing is sent |
| ERR_LISTING_NON_2XX | The release listing request returned a non-2xx response (NFR3, AC-8) | non-2xx | The error carries the HTTP status and never the token value |
| ERR_NO_RELEASES_FOUND | The listing holds no parseable, non-draft, non-prerelease release for `Latest` to select (FR10, AC-11) | - | The existing "no releases found" error; delivery stops and nothing is sent |

Unparseable tags inside the listing are not errors: they are skipped (FR3, AC-5). A result with no matching release is not an error either: an empty slice is returned (FR6, AC-3).

### Error Flow

```
Error occurs → return error from Since → Run aborts before mail send → state.json is NOT updated
```

Because the state save happens only after a successful send, any error on the `Since` path leaves `last_version` untouched, and the next run starts from the same saved version.

### Manual recovery for a corrupt `state.json` (EC-5)

Recovery is deliberately not automated. Two documented routes:

1. Edit `last_version` in `$XDG_STATE_HOME/claude-release-notes/state.json` by hand to a valid `vMAJOR.MINOR.PATCH` value.
2. Delete `$XDG_STATE_HOME/claude-release-notes/state.json` to return to first-run handling; only the newest parseable release is delivered (FR10).

Under FR10 this tool never writes an unparseable value itself, so a corrupt `state.json` can only arise from outside the tool.

## Performance Optimization

### Performance Goals
- GitHub API requests on the `Since` path: exactly 1 per run (NFR2).
- Releases handled per run: at most 10 (`per_page=10`, FR5).

### Optimization Strategies
- Single-page fetch: `Since` never walks pagination, which fixes both the request count and the payload size handed to translation and mail.

### Caching Strategy
No cache. `state.json` holds only the semver-greatest delivered version and is not a cache of listing content.

## Success Criteria

- [ ] All functional requirements (FR1–FR11) are implemented and tested
- [ ] All test scenarios (TS-1–TS-16) pass
- [ ] Request boundedness (1 request, `page=1`, `per_page` at the upper bound) holds
- [ ] Security requirements are satisfied (no token value in any error message)
- [ ] The persisted `last_version` is the semver maximum of what was delivered, and every value this tool writes parses as `vMAJOR.MINOR.PATCH`
- [ ] The semver comparison rule has exactly one definition site, in `internal/github` (NFR4)
- [ ] `internal/app`'s `ReleaseFetcher.Since` doc comment and the older `feature-docs/release-notes-fetcher/` documents match the current behaviour
- [ ] Code review is completed

## Open Questions

> **Note**: 未解決の要件は workflow.yaml で `status: tbd` として管理されています。
> plan フェーズの実行前に解決してください。

None. Every requirement in this specification has `status: ok`.

## References

- Requirements document: `feature-docs/release-since-semver/REQUIREMENTS.md`
- Implementation: `internal/github/client.go` (lines 85–159)
- Tests: `internal/github/client_test.go` (lines 96–284)
- Caller: `internal/app/app.go`
- FR8 update targets: `feature-docs/release-notes-fetcher/SPEC.md`, `feature-docs/release-notes-fetcher/REQUIREMENTS.md`, `feature-docs/release-notes-fetcher/tasks/task0003.md`, `feature-docs/release-notes-fetcher/IMPLEMENTATION.md`
