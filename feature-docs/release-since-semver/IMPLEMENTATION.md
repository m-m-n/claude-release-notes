# Implementation Plan: release-since-semver

## Overview

The behaviour this feature specifies — semver-based "newer" decision and a
single-page bounded fetch in `internal/github`, plus the caller doc comment in
`internal/app` — already exists in the integration branch (commit `13d0ac2`,
recorded as SPEC.md assumption A1). The remaining work is FR8: bringing the
stale statements in the older `feature-docs/release-notes-fetcher/` documents
in line with that behaviour.

## Technology Stack

- **Language**: Go (existing code; not modified by this plan's tasks)
- **Key libraries**: Go standard library only for the version comparison
  (NFR1)
- **Artifacts changed by tasks**: Markdown documents under
  `feature-docs/release-notes-fetcher/`

### Dependency licenses

This feature introduces NO new dependency (NFR1 forbids one). The single
existing entry in the module's require block, `gopkg.in/yaml.v3`, is
dual-licensed MIT / Apache-2.0 by its own LICENSE file and is compatible with
this project's `project.license: MIT`. Nothing in this plan changes the
require block, so no license question arises.

## Layer Structure

No code layers change. The layers below are stated only because the documents
being rewritten describe them, and the rewritten text must stay consistent
with this structure:

| Layer | Responsibility | Allowed dependency direction |
|---|---|---|
| `internal/app` | pipeline orchestration (config → state → github → translate → mail → state save) | depends on the packages below; nothing depends on it |
| `internal/github` | release listing retrieval; `Since` / `Latest`; unexported version parsing and comparison helpers | depends on the standard library and the GitHub API only |
| GitHub Releases API | external source of the listing | — |

## Shared Components

The single shared contract of this feature is the **current-behaviour
reference** below. It is not a component any task builds; it is the already
implemented behaviour that every rewritten sentence and every verification
item must agree with. Both the documentation task and VERIFICATION.md consume
it, so it is pinned here rather than in either one.

| Component | Responsibility | Contract (pre/postcondition) | Used by |
|---|---|---|---|
| `github` client, `Since(version)` | return the releases newer than the stored version | **Pre**: the stored version parses as `vMAJOR.MINOR.PATCH` (leading `v` optional), otherwise the call fails with an error naming the offending value. **Post**: exactly one listing request is issued, for the first page with the per-page cap; drafts, prereleases, unparseable tags and versions not strictly greater than the stored version are excluded; the remainder is returned in listing order (newest first); no match yields an empty result, not an error. | task0001, VERIFICATION.md |
| `github` client, `Latest()` | return the newest deliverable release | **Post**: walks pagination until a non-draft, non-prerelease entry is found; none found after exhausting pagination is an error. **Unchanged by this feature.** | task0001 (as an out-of-scope boundary), VERIFICATION.md |
| `app` release-fetcher abstraction | the pipeline's view of the two calls above | **Post**: its doc comment states semver comparison, single-page boundedness, and that an empty result means "already up to date" (FR7 — already satisfied). | VERIFICATION.md |

### Canonical statements (CB1–CB8)

Every rewritten location must be reducible to these eight statements. They
exist so that three documents in two languages end up saying the same thing.

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

## Conventions

These apply to every document rewritten under FR8.

- **Minimal edit**: rewrite only the enumerated locations. Do not restructure
  sections, renumber requirement IDs (`FR1`, `F01`, `TS-1`, `AC-1`, …), reflow
  unrelated paragraphs, or reformat tables that are not part of a target
  location. A large diff is a defect here, not thoroughness.
- **Language preservation**: `feature-docs/release-notes-fetcher/SPEC.md` and
  `tasks/task0003.md` stay in English; `REQUIREMENTS.md` stays in Japanese.
  Rewritten text matches the surrounding document's language and register.
- **Terminology**: use the terms already used by the current SPEC.md of this
  feature — semver comparison / identity match (同一性一致) / first page only
  (先頭 1 ページのみ) / bounded (有界). Do not invent synonyms per location.
- **No implementation detail**: the older documents are design documents.
  Describe behaviour and bounds; do not paste helper names, structures, or
  code fragments into them.
- **Historical records are never rewritten**: `reviews/round1.yaml`,
  `phase-state/`, and any other record of what was true at the time it was
  written stay untouched, even where they describe the old behaviour.
- **Statements that are still true stay**: a statement about pagination is
  stale only when it describes the differential fetch (CB1). Where it
  describes the latest-release call, it is current (CB7).

## Cross-task Design Decisions

### D1: FR1–FR7 produce no implementation task

The behaviour those requirements describe is already implemented and tested in
the integration branch (commit `13d0ac2`); the build and the full test suite
pass. Planning a task for them would mean re-implementing working code, which
YAGNI forbids. They are therefore carried entirely by VERIFICATION.md, as
verification that the existing implementation and its existing tests satisfy
them. Their `tasks` arrays in `workflow.yaml` stay empty by intent — this is
not an uncovered-requirement gap.

### D2: FR8 is one task, not three

The fifteen locations live in three files, but they are fifteen restatements
of the same eight canonical statements (CB1–CB8) with a shared terminology
decision. Splitting per file would give three implementers three independent
chances to pick different wording for the same fact, and the resulting
inconsistency is exactly the defect this feature exists to remove. The work is
one implementer session's worth, and its acceptance criteria stay well inside
the size limit.

### D3: The documentation check is textual, not a test binary

The project has no documentation-test harness, and adding one to police
fifteen prose edits would be a new architectural element bought for a one-off
cleanup. FR8's acceptance is therefore a defined textual check — the absence of
a fixed set of stale markers in the three target files, plus the presence of
the canonical statements at each location. It is mechanical and repeatable
without new tooling.

### D4: The scope boundary is the enumerated fifteen locations

SPEC.md FR8 enumerates the locations; the declared change set is derived from
the task's `files`. Locations outside the enumeration are not silently added,
even when they look stale — they are reported (see Open Questions). This keeps
the observed change set contained in the declared one.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| The rewrite spills beyond the fifteen locations and mangles unrelated sections of the older documents | Medium | Medium | Minimal-edit convention; the task plan lists the target locations and an explicit out-of-scope list |
| A still-true pagination statement (the latest-release call) is "corrected" into a false one | Medium | Medium | CB7 is a canonical statement; the task plan names the location and its acceptance criteria require it unchanged |
| The three documents end up describing the same behaviour in inconsistent terms | Medium | Low | Single task (D2) plus the fixed terminology convention and CB1–CB8 |
| Historical records (past review round, phase state) get rewritten to match current behaviour | Low | High | Convention: historical records are never rewritten; named in the task's out-of-scope list |
| Stale statements outside the enumerated fifteen remain in the older documents after this feature | High | Low | Reported as an open question rather than absorbed silently (D4) |

## Open Questions

- [ ] `feature-docs/release-notes-fetcher/IMPLEMENTATION.md` describes the
      differential fetch as "following pagination" in its Shared Components
      table. It is stale by CB1/CB2, but it is NOT one of the fifteen
      locations FR8 enumerates, so it is out of scope here. Whether to fold it
      into this feature or leave it to a follow-up needs a decision.
- [ ] FR1–FR7 and NFR1–NFR3 have no implementing task by design (D1). If the
      workflow's traceability check treats an empty `tasks` array as a gap
      regardless of cause, that expectation needs restating for
      documentation-only features.
- [ ] `workflow.yaml` defines no static-analysis command. VERIFICATION.md uses
      the Go-standard vet check, which is what the feature's own record shows
      was run; confirm that this is the intended project command.
