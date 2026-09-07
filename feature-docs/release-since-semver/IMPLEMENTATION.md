# Implementation Plan: release-since-semver

## Overview

Part of what this feature specifies already exists in the integration branch
(commit `13d0ac2`): the semver-based "newer" decision and the single-page
bounded differential fetch in `internal/github`. That part is confirmed, not
rebuilt. The remaining work is the write side and its documentation — the
caller doc comment (FR7), a semver-greatest persisted baseline (FR9), a
latest-release selection that refuses unparseable tags (FR10), a single
exported definition site for the comparison rule (NFR4), and the older
`feature-docs/release-notes-fetcher/` statements those three make stale
(FR11). FR8 is already complete (task0001, merged).

## Technology Stack

- **Language**: Go
- **Key libraries**: Go standard library only for version parsing and
  comparison (NFR1)
- **Artifacts changed by tasks**: `internal/github`, `internal/app`, and
  Markdown documents under `feature-docs/release-notes-fetcher/`

### Dependency licenses

This feature introduces NO new dependency (NFR1 forbids one). The single
existing entry in the module's require block, `gopkg.in/yaml.v3`, is
dual-licensed MIT / Apache-2.0 by its own LICENSE file and is compatible with
this project's `project.license: MIT`. Nothing in this plan changes the
require block, so no license question arises.

## Layer Structure

| Layer | Responsibility | Allowed dependency direction |
|---|---|---|
| `internal/app` | pipeline orchestration (config → state → github → translate → mail → state save), and the decision of which value the save receives | depends on the packages below; nothing depends on it |
| `internal/github` | release listing retrieval; the differential and latest-release queries; the exported semver-maximum selector; the unexported parsing and comparison helpers behind it | depends on the standard library and the GitHub API only |
| GitHub Releases API | external source of the listing | — |

One constraint this feature adds to the structure: the `vMAJOR.MINOR.PATCH`
parsing and comparison rule exists in `internal/github` and nowhere else
(NFR4). `internal/app` reaches it only by calling the exported selector below;
it never re-parses a tag name and never restates the comparison rule, not even
inside a test helper.

## Shared Components

Tasks run fully in parallel, so every component one task builds and another
task or the verification plan depends on is pinned here.

| Component | Responsibility | Contract (pre/postcondition) | Used by |
|---|---|---|---|
| `github` client, differential query `Since(version)` | return the releases newer than the stored version | **Pre**: the stored version parses as `vMAJOR.MINOR.PATCH` (leading `v` optional), otherwise the call fails with an error naming the offending value. **Post**: exactly one listing request is issued, for the first page with the per-page cap; drafts, prereleases, unparseable tags and versions not strictly greater than the stored version are excluded; the remainder is returned in listing order (newest first); no match yields an empty result, not an error. **Unchanged by this feature.** | task0002 (as an unchanged boundary), task0003, VERIFICATION.md |
| `github` client, latest-release query `Latest()` | return the newest deliverable release | **Post**: walks pagination and returns the first entry, in listing order, that is not a draft, not a prerelease, and whose tag name parses as `vMAJOR.MINOR.PATCH`; exhausting pagination without such an entry yields the existing "no releases found" error. **Revised by FR10** — this row supersedes the previous plan generation's "Unchanged by this feature"; the pagination half is what stays unchanged, the selection rule is what narrows. | task0002 (provides), task0003, VERIFICATION.md |
| `github` semver-maximum selector, `MaxByVersion(releases []Release) (Release, bool)` | the single reachable definition site of the comparison rule (NFR4) | **Pre**: none — a nil or empty input is permitted. **Post**: a package-level pure function, not a method on the client; performs no I/O and does not modify or reorder its input. Returns the release whose tag name is the greatest parsed version among the inputs that parse, with `true`. Inputs whose tag does not parse are ignored. On equal parsed versions the earliest such input wins, so the result is deterministic. An empty input, or one in which no tag parses, returns the zero release value and `false`. | task0002 (provides and consumes), task0003, VERIFICATION.md |
| `app` release-fetcher abstraction, doc comment on the differential method | the pipeline's stated view of the differential query | **Post**: the doc comment states all three of — newness decided by semantic-version comparison of tag names, boundedness to a single listing page, and that an empty result means "already up to date". **Not yet satisfied**: the current comment carries the last two and omits the comparison statement (FR7, verify TS-11). | task0002 (provides), VERIFICATION.md |
| `app` run pipeline, persisted-baseline decision | choose the value handed to the state store after a successful send | **Pre**: the send succeeded and the delivered set is non-empty (the pipeline already returns early on an empty set). **Post**: the value handed to the store is the tag name of the release the selector above returns for the delivered set — never the head of that set. The delivered set's own order (listing order, newest first) is untouched, so the mail's ordering is unaffected. A `false` from the selector is unreachable given the two query contracts above; should it occur, it is surfaced the same way a store failure already is — a state-stage error after the send, leaving the stored value untouched. | task0002 (provides), task0003, VERIFICATION.md |
| `state` store, save | persist the baseline | **Post**: writes the value it is handed, atomically, only after a successful send. **Unchanged by this feature** — the timing and the mechanism are existing contracts; only the value the caller chooses changes (FR9). | task0002, task0003, VERIFICATION.md |

### Canonical statements (CB1–CB10)

Every rewritten location in the older documents must be reducible to these
statements. They exist so that four documents in two languages end up saying
the same thing. CB1–CB8 are carried forward unchanged from the generation that
drove task0001 (FR8); CB9 and CB10 are added for FR9 and FR10.

| ID | Statement |
|---|---|
| CB1 | The differential fetch reads only the first listing page (per-page cap 10, page 1) and never walks pagination. |
| CB2 | "Newer" is decided by comparing the semantic version of tag names — major, then minor, then patch, strictly greater — not by locating the stored version in the listing by identity match. |
| CB3 | A stored version absent from the listing is a normal success path: only the versions comparing greater are returned, never the whole listing. |
| CB4 | An unparseable stored version is an error naming the offending value; there is no fallback to delivering everything. |
| CB5 | Unparseable tags inside the listing are skipped and the run continues. |
| CB6 | Drafts and prereleases are excluded from both the differential and the latest-release results; listing order (newest first) is preserved; no match yields an empty result, not an error. |
| CB7 | The latest-release call still walks pagination — statements describing IT as paging are current, not stale. |
| CB8 | The differential call is named `Since`; no method named `FetchNewer` exists. |
| CB9 | The value written to the state file after a successful send is the semver-greatest tag name among the releases delivered in that run — not the head of the listing and not the head of the delivered result. Nothing is written when nothing was delivered. |
| CB10 | The latest-release call selects the first listing entry that is not a draft, not a prerelease, and whose tag parses as `vMAJOR.MINOR.PATCH`; exhausting pagination without such an entry yields the existing "no releases found" error. CB7 is unaffected — the call still walks pagination; only its selection rule narrows. |

## Conventions

### Documentation rewrites (FR8, and now FR11)

- **Minimal edit**: rewrite only the enumerated locations. Do not restructure
  sections, renumber requirement IDs (`FR1`, `F01`, `TS-1`, `AC-1`, …), reflow
  unrelated paragraphs, or reformat tables that are not part of a target
  location. A large diff is a defect here, not thoroughness.
- **Language preservation**: `feature-docs/release-notes-fetcher/SPEC.md`,
  `IMPLEMENTATION.md` and `tasks/task0003.md` stay in English;
  `REQUIREMENTS.md` stays in Japanese. Rewritten text matches the surrounding
  document's language and register.
- **Terminology**: use the terms already used by this feature's SPEC.md —
  semver comparison / identity match (同一性一致) / first page only
  (先頭 1 ページのみ) / bounded (有界) / semver-greatest (semver 上で最大).
  Do not invent synonyms per location.
- **No implementation detail**: the older documents are design documents.
  Describe behaviour and bounds; do not paste helper names, structures, or
  code fragments into them. The exported selector's name is an internal
  structural decision and does not belong in them either.
- **Historical records are never rewritten**: `reviews/round1.yaml`,
  `phase-state/`, and any other record of what was true at the time it was
  written stay untouched, even where they describe the old behaviour.
- **Statements that are still true stay**: a statement about pagination is
  stale only when it describes the differential fetch (CB1). Where it
  describes the latest-release call, it is current (CB7).

### Go changes

- **No new exported identifier** beyond the single selector NFR4 requires.
  `version`, `parseVersion` and `isNewer` stay unexported behind it.
- **Error wrapping** keeps the existing stage-prefix convention
  (`"github: …"`, `"state: …"`); no error message ever carries a token value
  (NFR3).
- **Tests** are table-driven where the scenario enumerates cases, colocated in
  `_test.go` beside the code, and use the consumer-side interfaces
  `internal/app` already declares as the mocking seam. No real network, no
  real SMTP, no real subprocess.
- **The delivered result's order is not a free variable**: the mail is built
  from the delivered slice in its existing order. Any change that reorders it
  to make the persisted value easier to pick is out of bounds.

## Cross-task Design Decisions

### D1: FR1–FR6 and NFR1–NFR3 produce no implementation task

The behaviour those requirements describe is already implemented and tested in
the integration branch (commit `13d0ac2`); the build and the full test suite
pass. Planning a task for them would mean re-implementing working code, which
YAGNI forbids. They are therefore carried entirely by VERIFICATION.md, as
confirmation that the existing implementation and its existing tests satisfy
them. Their `tasks` arrays in `workflow.yaml` stay empty by intent — this is
not an uncovered-requirement gap.

FR7 was inside this set in the previous plan generation and has left it: the
verify phase (TS-11) found the doc comment states single-page boundedness and
the empty-result meaning but omits the semver-comparison statement the
requirement demands. It is now a change item.

### D2: FR8 was one task, not four (historical)

The sixteen locations lived in four files but were sixteen restatements of the
same canonical statements with a shared terminology decision; splitting per
file would have given four implementers four independent chances to word the
same fact differently. That task (task0001) is merged. The reasoning is kept
here because D5 and D8 below reuse it.

### D3: Documentation checks are textual, not a test binary

The project has no documentation-test harness, and adding one to police prose
edits would be a new architectural element bought for a one-off cleanup. TS-9,
TS-11, TS-15, and the second half of TS-16 are therefore defined textual
checks — the absence of a fixed set of stale markers in named files, plus the
presence of the canonical statements at each location. They are mechanical and
repeatable without new tooling.

### D4: The scope boundary is the enumerated location set

FR8 enumerated sixteen locations; FR11's twenty are enumerated in task0003's
plan. Locations outside an enumeration are not silently added, even when they
look stale — they are reported (see Open Questions). This keeps the observed
change set contained in the declared one.

### D5: Two tasks — one Go-code task, one documentation task

The split is by artifact kind, not by requirement. The Go work (FR7, FR9,
FR10, NFR4) is a handful of small changes across two packages plus their
tests; the documentation work (FR11) is prose in four Markdown files under a
different tree. They share no file, so they can be merged in either order, and
each is one implementer session's worth. Splitting the Go work further is
rejected by D7; splitting the documentation work per file is rejected for D2's
reason.

### D6: The shape of NFR4's exported entry point

SPEC.md fixes that the entry point is a package-level pure function in
`internal/github` taking a set of releases, and leaves the name, the signature
and the empty-input behaviour to this plan. The decisions:

- **Name and signature**: `MaxByVersion(releases []Release) (Release, bool)`.
  The name says what the selection is (maximum by version), which is the fact
  the old code got wrong by taking the head. Returning the release rather than
  the tag string keeps the caller from having to know that the tag name is the
  version.
- **Empty or all-unparseable input**: the zero release value with `false`,
  rather than an error. The condition is not a failure of the function — it is
  the caller asking for a maximum of nothing — and the two query contracts
  above make it unreachable from the pipeline. Returning an error would put an
  error path into a pure function that no caller can trigger.
- **Ties**: equal parsed versions resolve to the earliest input, which falls
  out of a strictly-greater comparison and makes the result deterministic for
  a listing that repeats a version.

### D7: FR9 and NFR4 stay in the same task

FR9's call site cannot compile until the entry point exists, and the entry
point lives in a package the calling side does not own. Splitting them would
require the consumer to compile against a placeholder it would have to add to
`internal/github` itself — a duplicate-definition merge hazard, and one that
buys nothing, because the whole Go change is one session's work (D5). The
contract is still pinned in Shared Components above, because the documentation
task and VERIFICATION.md both depend on it without building it.

### D8: FR11's location set, and the two task0001 boundaries it supersedes

FR11's set is disjoint from FR8's sixteen at **statement** granularity: no
sentence rewritten by task0001 is rewritten again. Two of task0001's
preserved-statement boundaries are, however, deliberately superseded, because
FR10 made the statements they protected stale after that task completed:

- task0001 AC-7 required the latest-release half of the Shared Components cell
  in `feature-docs/release-notes-fetcher/IMPLEMENTATION.md` to stay
  byte-identical. FR10 changes the selection rule that half states, so it is
  now an FR11 target.
- task0001 AC-5 required `feature-docs/release-notes-fetcher/tasks/task0003.md`
  Design item 4 to stay byte-identical, for its pagination claim (CB7). The
  pagination claim still stands; the selection rule in the same item does not,
  and is an FR11 target.

Everything else task0001 preserved stays preserved. VERIFICATION.md's manual
item covering that preservation is narrowed to the pagination claim
accordingly.

### D9: TS-16's duplicate-implementation half is a textual check

The table-driven half of TS-16 is an ordinary unit test of the exported
selector. Its second half — that no `vMAJOR.MINOR.PATCH` parsing or comparison
exists in `internal/app` — is a property of source text, not of a running
program; expressing it as a Go test would mean a test that reads and pattern-
matches its own repository's source. It is therefore a textual check under D3,
and the same check confirms that the unexported helpers stayed unexported.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| The persisted-baseline change is made by reordering the delivered set (sorting it) instead of selecting from it, silently changing the mail's ordering | Medium | High | The Shared Components row and the Go conventions both state that the delivered order is untouched; the mail-ordering assertion is part of the task's acceptance criteria |
| `internal/app` grows its own tag parsing or comparison while implementing FR9, defeating NFR4 | Medium | Medium | The selector contract is pinned; TS-16's textual half checks for it; the task's acceptance criteria require exactly one call site |
| The latest-release change also alters pagination, breaking CB7 and the still-true statements that depend on it | Low | Medium | The revised `Latest()` contract states the pagination half is unchanged; VERIFICATION.md keeps a manual item on it |
| The FR11 rewrite spills into statements task0001 already rewrote, producing a churn diff or a contradiction | Medium | Medium | D8's statement-granularity disjointness; task0003's table names the location and what it currently says; the minimal-edit convention |
| The two task0001 boundaries D8 supersedes get "restored" by a reviewer reading task0001's acceptance criteria as still binding | Medium | Medium | D8 states the supersession explicitly, and VERIFICATION.md's manual item is narrowed rather than left contradicting it |
| The new pipeline test needs mocks for all five leaf roles and drifts into re-testing translation and mail | Medium | Low | The task plan scopes the new test to the persisted value and the delivered ordering; other stages are stubbed to succeed |
| Stale statements outside the enumerated sets remain in the older documents after this feature | High | Low | Reported as an open question rather than absorbed silently (D4) |

## Open Questions

- [ ] FR1–FR6 and NFR1–NFR3 have no implementing task by design (D1). If the
      workflow's traceability check treats an empty `tasks` array as a gap
      regardless of cause, that expectation needs restating for confirmation
      requirements.
- [ ] `workflow.yaml` defines no static-analysis command. VERIFICATION.md uses
      the Go-standard vet check, which is what the feature's own record shows
      was run; confirm that this is the intended project command.
- [ ] Two statements in the older documents are stale for FR8-era reasons
      rather than FR9/FR10 reasons, and so fall outside both enumerations:
      the goal line of `feature-docs/release-notes-fetcher/tasks/task0003.md`
      (which describes the differential query as returning all newer
      releases) and the expected-effect line of that feature's REQUIREMENTS.md
      section 2.3 (which claims releases are never missed, now bounded by the
      single-page cap). The first is inside an FR11 target line and is
      corrected as a side effect of that rewrite; the second is reported, not
      fixed.
- [ ] The FR11 scan covers the four documents FR8's reference-impact scan
      identified. Other files under `feature-docs/release-notes-fetcher/tasks/`
      were not scanned for FR9/FR10 staleness; if any exists that describes
      the pipeline's state save, it is outside this plan's enumeration.
