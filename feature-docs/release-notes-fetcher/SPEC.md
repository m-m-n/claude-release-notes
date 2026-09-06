# Feature: Claude Code Release Notes Fetcher (Japanese Email Digest)

## Overview

A single-shot Go CLI binary that fetches new releases of `anthropics/claude-code`
from the GitHub Releases API, translates the release notes to Japanese via the
local `claude-batch` command, and sends them as one HTML email through Gmail
SMTP. Scheduling is delegated to a systemd user timer (sample units shipped in
the repository).

## Objectives

- Never miss a Claude Code release; receive it passively in Japanese
- Keep the tool a dependency-light single binary (stdlib + YAML parser only)
- Be safely re-runnable: at-least-once delivery with state updated only after a
  successful send

## User Stories

### US1: Periodic digest
As the owner, I want new release notes fetched and emailed in Japanese three
times a day, so that I stay up to date without manual checking.

**Acceptance Criteria:**
- [ ] Releases newer than the stored version, decided by semantic-version
      comparison of tag names, are fetched from the first listing page only,
      so a single run delivers at most the per-page cap
- [ ] All pending releases are sent in ONE HTML email, newest version first
- [ ] The state file is updated only after the email is sent successfully
- [ ] When there is no new release, the run exits 0 without sending anything

### US2: First run
As the owner, I want the very first run to process only the latest release,
so that I do not receive a huge backlog email.

**Acceptance Criteria:**
- [ ] With no state file present, exactly the latest release is fetched,
      translated, sent, and recorded

### US3: Failure recovery
As the owner, I want any mid-run failure to leave the state untouched, so that
the next scheduled run retries automatically.

**Acceptance Criteria:**
- [ ] GitHub / claude-batch / SMTP failure → non-zero exit, state file unchanged
- [ ] Errors are written to stderr; secrets never appear in output

## Technical Requirements

### Functional Requirements
- **FR1:** Fetch releases from `GET /repos/anthropics/claude-code/releases`
  (GitHub REST API) with mandatory token authentication
  (`Authorization: Bearer <token>`), issuing exactly one request for the
  first listing page (bounded by the per-page cap) and never walking
  pagination. A release is collected when its `tag_name` is a newer
  semantic version than the stored version — not by locating the stored
  version in the listing by identity match — so a stored version absent
  from the listing is a normal case where only the releases that compare
  newer are collected, never the whole listing. Tags that do not parse as a
  version are skipped. Drafts and prereleases are excluded. Ordering relies
  on the API's reverse-chronological listing.
- **FR2:** Persist the last delivered release `tag_name` in
  `${XDG_STATE_HOME:-~/.local/state}/claude-release-notes/state.json`. When the
  state file is absent (first run), process only the latest release. Write the
  state file only after a successful email send (write-temp + rename).
- **FR3:** Load configuration from
  `${XDG_CONFIG_HOME:-~/.config}/claude-release-notes/config.yaml` with keys:
  `gmail.account`, `gmail.app_password`, `mail.to`, `github.token`. Missing or
  empty required keys → error exit with a message naming the key (value never
  printed).
- **FR4:** Translate by executing `claude-batch -`, writing to stdin a Japanese
  translation instruction followed by ALL pending release notes (one combined
  call), and reading the translated text from stdout. The prompt must instruct
  claude-batch to keep per-release delimiters so the output can be split back
  into per-release sections. Non-zero exit, or empty/undelimitable output →
  error exit.
- **FR5:** Build one HTML email containing every pending release, newest
  version first; each release section shows the version (tag name), publish
  date, and translated body. Send via Gmail SMTP (`smtp.gmail.com:587`,
  STARTTLS, app-password auth) as a MIME message with `Content-Type: text/html;
  charset=UTF-8` and a UTF-8 encoded subject. Visual design follows DESIGN.md
  (design step output).
- **FR6:** Ship sample systemd user units in the repository
  (`systemd/claude-release-notes.service`,
  `systemd/claude-release-notes.timer`) with
  `OnCalendar=*-*-* 06,12,18:00:00 Asia/Tokyo` and `Persistent=true`, plus
  install instructions in README.md.

### Non-Functional Requirements
- **NFR1 - Dependencies:** Go standard library only, except one YAML parser
  (`gopkg.in/yaml.v3`). Single static binary, no runtime dependencies beyond
  `claude-batch` on PATH.
- **NFR2 - Security:** Secrets (app password, GitHub token) live only in the
  config file and are never written to stdout/stderr or included in error
  messages. SMTP uses STARTTLS.
- **NFR3 - Observability:** Progress and results are logged to stdout, errors
  to stderr; no log files. Timestamps in output include an explicit timezone.
- **NFR4 - Reliability:** At-least-once delivery semantics; every failure path
  exits non-zero without touching the state file.

## Implementation Approach

### Architecture

```
main (cmd/claude-release-notes)
  └── run pipeline
        ├── config   : load + validate YAML config        (FR3)
        ├── state    : read/write last-version state file (FR2)
        ├── github   : Releases API client                (FR1)
        ├── translate: claude-batch subprocess wrapper    (FR4)
        └── mail     : HTML builder + Gmail SMTP sender   (FR5)
```

Each external integration (`github`, `translate`, `mail`) is defined as an
interface consumed by the pipeline so unit tests can substitute mocks.

### Data Flow

```
state.json ─┐
            ├→ github.Since(lastVersion) → []Release
config.yaml ┘         │
                      ▼
        translate.Translate(releases) → per-release Japanese text
                      │
                      ▼
        mail.Build(releases, translations) → HTML → mail.Send()
                      │ (success only)
                      ▼
              state.Write(newestVersion)
```

### API Design

External API only (no server):

```
GET https://api.github.com/repos/anthropics/claude-code/releases?per_page=10&page=1
Headers:
  Authorization: Bearer {github.token}
  Accept: application/vnd.github+json
```

Response fields used: `tag_name`, `body`, `published_at`, `draft`,
`prerelease`.

### Database Schema

N/A. Files only:

- `config.yaml` (user-authored):
  ```yaml
  gmail:
    account: user@gmail.com
    app_password: "xxxx xxxx xxxx xxxx"
  mail:
    to: recipient@example.com
  github:
    token: ghp_xxx
  ```
- `state.json` (tool-written): `{"last_version": "v1.2.3"}`

### Dependencies

**Internal Dependencies:** none (new repository).

**External Dependencies:**
- `gopkg.in/yaml.v3`: YAML config parsing (only third-party module)
- `claude-batch` (local command, `/usr/bin/claude-batch`): translation;
  invoked as `claude-batch -` with the prompt on stdin (the `-p` flag must NOT
  be used)
- GitHub REST API, Gmail SMTP: runtime services

### File Structure

```
cmd/claude-release-notes/main.go   # entry point, pipeline wiring
internal/config/config.go          # FR3 (+ config_test.go)
internal/state/state.go            # FR2 (+ state_test.go)
internal/github/client.go          # FR1 (+ client_test.go)
internal/translate/claudebatch.go  # FR4 (+ claudebatch_test.go)
internal/mail/build.go             # FR5 HTML build (+ build_test.go)
internal/mail/send.go              # FR5 SMTP send (+ send_test.go)
systemd/claude-release-notes.service   # FR6
systemd/claude-release-notes.timer     # FR6
README.md                          # install & setup instructions
```

## Test Scenarios

### Unit Tests
- [ ] TS-1: FR1 — given a stored version and a mocked single-page API
      response, the releases that compare newer are returned newest first,
      and drafts/prereleases are skipped
- [ ] TS-2: FR2 — first run (no state file) yields only the latest release;
      state write is atomic and creates directories as needed
- [ ] TS-3: FR3 — valid config parses; each missing required key produces an
      error naming the key without printing values
- [ ] TS-4: FR4 — prompt contains delimiters and all release bodies; output is
      split back per release; non-zero exit and empty output produce errors
- [ ] TS-5: FR5 — HTML contains every release, ordered newest first, with
      version, date, and translated body; message headers are correct MIME
- [ ] TS-6: NFR4 — pipeline aborts without state update when any stage fails
- [ ] TS-7: FR1/FR2 — no new releases → no translation, no email, exit 0

### Integration Tests
None (external systems mocked; see test/README.md).

### E2E Tests
**Existing E2E tests**: None
**Run command**: Not detected
- [ ] N/A

### Edge Cases
- [ ] State file exists but its version no longer appears in the API listing
      (e.g. very old): a normal success path, returning only the releases
      that compare newer, never the whole listing
- [ ] Release body is empty: section is rendered with an "(no notes)" marker,
      not dropped
- [ ] claude-batch output missing a delimiter for one release: treated as a
      translation failure (error exit)

### Performance Tests
N/A (lightweight I/O batch).

## Security Considerations

- **Authentication:** GitHub PAT (Bearer header); Gmail app password over
  STARTTLS SMTP AUTH
- **Input Validation:** config keys validated at startup; release bodies are
  HTML-escaped before insertion into the email template
- **Data Protection:** secrets confined to the config file; never logged.
  README instructs `chmod 600 config.yaml`
- **XSS Prevention:** release note text is escaped via `html/template`
  auto-escaping in the email builder
- **SQL Injection / CSRF:** N/A (no DB, no server)

## Error Handling

### Error Codes

Process exit codes (no HTTP surface):

| Code | Description | User Message |
|------|-------------|--------------|
| 0 | Success (including "no new releases") | — |
| 1 | Any failure: config, state read, GitHub API, claude-batch, SMTP | stderr message identifying the failed stage |

### Error Flow

```
Error occurs → wrap with stage context → print to stderr → exit 1
(state.json is written only on the success path, after SMTP send)
```

## Performance Optimization

N/A — lightweight I/O batch, runs three times a day.

## Success Criteria

- [ ] All functional requirements (FR1–FR6) are implemented and unit-tested
- [ ] All test scenarios (TS-1…TS-7) pass via `go test ./...`
- [ ] Security requirements are satisfied (no secret leakage in output)
- [ ] README documents config setup, Gmail app-password setup, and systemd
      timer installation
- [ ] Code review is completed

## Open Questions

None — all requirements are `status: ok`. HTML email visual design is resolved
by the design step (DESIGN.md), not an open spec question.

## References

- Discussion report: tmp/discussion-release-notes-fetcher.md
- Requirements (Japanese): feature-docs/release-notes-fetcher/REQUIREMENTS.md
- GitHub Releases API: https://docs.github.com/en/rest/releases/releases
