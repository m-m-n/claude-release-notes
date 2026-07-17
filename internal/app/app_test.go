package app

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"claude-release-notes/internal/config"
	"claude-release-notes/internal/github"
	"claude-release-notes/internal/mail"
)

// --- mocks recording invocations at the app-level seam ---

type mockConfigLoader struct {
	cfg config.Config
	err error
}

func (m mockConfigLoader) Load() (config.Config, error) { return m.cfg, m.err }

type mockStateStore struct {
	version string
	found   bool
	loadErr error
	saveErr error

	saveCalled   bool
	savedVersion string
}

func (m *mockStateStore) Load() (string, bool, error) { return m.version, m.found, m.loadErr }

func (m *mockStateStore) Save(version string) error {
	m.saveCalled = true
	m.savedVersion = version
	return m.saveErr
}

type mockFetcher struct {
	latest    github.Release
	latestErr error

	since    []github.Release
	sinceErr error

	latestCalled    bool
	sinceCalled     bool
	sinceCalledWith string
}

func (m *mockFetcher) Latest() (github.Release, error) {
	m.latestCalled = true
	return m.latest, m.latestErr
}

func (m *mockFetcher) Since(version string) ([]github.Release, error) {
	m.sinceCalled = true
	m.sinceCalledWith = version
	return m.since, m.sinceErr
}

type mockTranslator struct {
	out []string
	err error

	called      bool
	gotSections []string
}

func (m *mockTranslator) Translate(sections []string) ([]string, error) {
	m.called = true
	m.gotSections = sections
	return m.out, m.err
}

type mockSender struct {
	err error

	called  bool
	to      string
	subject string
	body    string
}

func (m *mockSender) Send(to, subject, htmlBody string) error {
	m.called = true
	m.to, m.subject, m.body = to, subject, htmlBody
	return m.err
}

// newTestDeps builds a Dependencies value wired to the given mocks, with
// buffers for stdout/stderr and a build func that records its call.
func newTestDeps(cfg config.Config, state *mockStateStore, fetcher *mockFetcher, translator *mockTranslator, sender *mockSender, stdout, stderr *bytes.Buffer) (Dependencies, *bool, *[]mail.Item) {
	buildCalled := false
	var buildItems []mail.Item
	deps := Dependencies{
		Config: mockConfigLoader{cfg: cfg},
		State:  state,
		NewFetcher: func(token string) ReleaseFetcher {
			return fetcher
		},
		Translator: translator,
		Build: func(items []mail.Item) (string, string) {
			buildCalled = true
			buildItems = items
			return "subject", "htmlBody"
		},
		NewSender: func(account, appPassword string) MailSender {
			return sender
		},
		Stdout: stdout,
		Stderr: stderr,
	}
	return deps, &buildCalled, &buildItems
}

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("parsing time %q: %v", s, err)
	}
	return tm
}

// AC-1: first run (state not found) processes exactly the latest release and
// saves its version after a successful send.
func TestRun_FirstRunProcessesOnlyLatestAndSaves(t *testing.T) {
	cfg := config.Config{GmailAccount: "a@example.com", GmailAppPassword: "pw", MailTo: "to@example.com", GitHubToken: "tok"}
	state := &mockStateStore{found: false}
	release := github.Release{TagName: "v1.0.0", Body: "body1", PublishedAt: mustTime(t, "2026-01-01T00:00:00Z")}
	fetcher := &mockFetcher{latest: release}
	translator := &mockTranslator{out: []string{"訳1"}}
	sender := &mockSender{}
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	deps, _, _ := newTestDeps(cfg, state, fetcher, translator, sender, stdout, stderr)

	if err := Run(deps); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if !fetcher.latestCalled {
		t.Error("Latest() was not called on first run")
	}
	if fetcher.sinceCalled {
		t.Error("Since() must not be called on first run")
	}
	if !state.saveCalled {
		t.Fatal("state.Save was not called")
	}
	if state.savedVersion != "v1.0.0" {
		t.Errorf("saved version = %q, want %q", state.savedVersion, "v1.0.0")
	}
}

// AC-2: subsequent run fetches only releases newer than the stored version
// and saves the newest delivered version after sending.
func TestRun_SubsequentRunFetchesSinceStoredVersion(t *testing.T) {
	cfg := config.Config{GmailAccount: "a@example.com", GmailAppPassword: "pw", MailTo: "to@example.com", GitHubToken: "tok"}
	state := &mockStateStore{found: true, version: "v1.0.0"}
	releases := []github.Release{
		{TagName: "v1.2.0", Body: "body3", PublishedAt: mustTime(t, "2026-03-01T00:00:00Z")},
		{TagName: "v1.1.0", Body: "body2", PublishedAt: mustTime(t, "2026-02-01T00:00:00Z")},
	}
	fetcher := &mockFetcher{since: releases}
	translator := &mockTranslator{out: []string{"訳3", "訳2"}}
	sender := &mockSender{}
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	deps, _, _ := newTestDeps(cfg, state, fetcher, translator, sender, stdout, stderr)

	if err := Run(deps); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if fetcher.latestCalled {
		t.Error("Latest() must not be called when state was found")
	}
	if !fetcher.sinceCalled || fetcher.sinceCalledWith != "v1.0.0" {
		t.Errorf("Since() called with %q (called=%v), want %q", fetcher.sinceCalledWith, fetcher.sinceCalled, "v1.0.0")
	}
	if !state.saveCalled || state.savedVersion != "v1.2.0" {
		t.Errorf("saved version = %q (called=%v), want %q", state.savedVersion, state.saveCalled, "v1.2.0")
	}
}

// AC-3: zero new releases -> success, and neither translator, mail, nor
// state save is invoked.
func TestRun_NoNewReleases_SkipsDownstreamStages(t *testing.T) {
	cfg := config.Config{GmailAccount: "a@example.com", GmailAppPassword: "pw", MailTo: "to@example.com", GitHubToken: "tok"}
	state := &mockStateStore{found: true, version: "v1.0.0"}
	fetcher := &mockFetcher{since: nil}
	translator := &mockTranslator{}
	sender := &mockSender{}
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	deps, buildCalled, _ := newTestDeps(cfg, state, fetcher, translator, sender, stdout, stderr)

	if err := Run(deps); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if translator.called {
		t.Error("Translate must not be called when there are no new releases")
	}
	if *buildCalled {
		t.Error("Build must not be called when there are no new releases")
	}
	if sender.called {
		t.Error("Send must not be called when there are no new releases")
	}
	if state.saveCalled {
		t.Error("Save must not be called when there are no new releases")
	}
	if !strings.Contains(stdout.String(), "no new releases") {
		t.Errorf("stdout = %q, want it to mention no new releases", stdout.String())
	}
}

// AC-4: failure injected at each stage (fetch / translate / send) -> the
// pipeline returns an error carrying that stage's prefix and state save is
// never invoked.
func TestRun_StageFailures_WrapWithStagePrefixAndSkipSave(t *testing.T) {
	sentinel := errors.New("boom")
	cfg := config.Config{GmailAccount: "a@example.com", GmailAppPassword: "pw", MailTo: "to@example.com", GitHubToken: "tok"}
	release := github.Release{TagName: "v1.0.0", Body: "body1", PublishedAt: mustTime(t, "2026-01-01T00:00:00Z")}

	tests := []struct {
		name       string
		wantPrefix string
		build      func() (*mockStateStore, *mockFetcher, *mockTranslator, *mockSender)
	}{
		{
			name:       "fetch failure",
			wantPrefix: "github: ",
			build: func() (*mockStateStore, *mockFetcher, *mockTranslator, *mockSender) {
				return &mockStateStore{found: false}, &mockFetcher{latestErr: sentinel}, &mockTranslator{}, &mockSender{}
			},
		},
		{
			name:       "translate failure",
			wantPrefix: "translate: ",
			build: func() (*mockStateStore, *mockFetcher, *mockTranslator, *mockSender) {
				return &mockStateStore{found: false}, &mockFetcher{latest: release}, &mockTranslator{err: sentinel}, &mockSender{}
			},
		},
		{
			name:       "send failure",
			wantPrefix: "mail: ",
			build: func() (*mockStateStore, *mockFetcher, *mockTranslator, *mockSender) {
				return &mockStateStore{found: false}, &mockFetcher{latest: release}, &mockTranslator{out: []string{"訳1"}}, &mockSender{err: sentinel}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			state, fetcher, translator, sender := tc.build()
			stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
			deps, _, _ := newTestDeps(cfg, state, fetcher, translator, sender, stdout, stderr)

			err := Run(deps)
			if err == nil {
				t.Fatal("Run() error = nil, want error")
			}
			if !strings.HasPrefix(err.Error(), tc.wantPrefix) {
				t.Errorf("error = %q, want prefix %q", err.Error(), tc.wantPrefix)
			}
			if !errors.Is(err, sentinel) {
				t.Errorf("error chain does not wrap the underlying failure: %v", err)
			}
			if state.saveCalled {
				t.Error("state.Save must never be invoked on stage failure")
			}
			if !strings.Contains(stderr.String(), tc.wantPrefix) {
				t.Errorf("stderr = %q, want it to contain %q", stderr.String(), tc.wantPrefix)
			}
		})
	}
}

// AC-5: releases flow to the mail builder newest first with translations
// aligned to their releases.
func TestRun_BuildReceivesReleasesNewestFirstWithAlignedTranslations(t *testing.T) {
	cfg := config.Config{GmailAccount: "a@example.com", GmailAppPassword: "pw", MailTo: "to@example.com", GitHubToken: "tok"}
	state := &mockStateStore{found: true, version: "v1.0.0"}
	t2 := mustTime(t, "2026-03-01T00:00:00Z")
	t1 := mustTime(t, "2026-02-01T00:00:00Z")
	releases := []github.Release{
		{TagName: "v1.2.0", Body: "body-new", PublishedAt: t2},
		{TagName: "v1.1.0", Body: "body-old", PublishedAt: t1},
	}
	fetcher := &mockFetcher{since: releases}
	translator := &mockTranslator{out: []string{"訳-new", "訳-old"}}
	sender := &mockSender{}
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	deps, buildCalled, buildItems := newTestDeps(cfg, state, fetcher, translator, sender, stdout, stderr)

	if err := Run(deps); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if !*buildCalled {
		t.Fatal("Build was not called")
	}
	items := *buildItems
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].Version != "v1.2.0" || items[0].Body != "訳-new" || !items[0].PublishedAt.Equal(t2) {
		t.Errorf("items[0] = %+v, want version v1.2.0 body 訳-new", items[0])
	}
	if items[1].Version != "v1.1.0" || items[1].Body != "訳-old" || !items[1].PublishedAt.Equal(t1) {
		t.Errorf("items[1] = %+v, want version v1.1.0 body 訳-old", items[1])
	}
	if translator.gotSections[0] != "body-new" || translator.gotSections[1] != "body-old" {
		t.Errorf("translator input = %v, want order preserved from releases", translator.gotSections)
	}
	if sender.to != "to@example.com" {
		t.Errorf("sender.to = %q, want %q", sender.to, "to@example.com")
	}
}

// AC-6: progress logging goes to stdout and error output to stderr (via the
// injected writers); any timestamp in output includes a timezone
// designation.
func TestRun_LogsProgressToStdoutAndErrorsToStderrWithTimezone(t *testing.T) {
	cfg := config.Config{GmailAccount: "a@example.com", GmailAppPassword: "pw", MailTo: "to@example.com", GitHubToken: "tok"}
	state := &mockStateStore{found: false}
	release := github.Release{TagName: "v1.0.0", Body: "body1", PublishedAt: mustTime(t, "2026-01-01T00:00:00Z")}
	fetcher := &mockFetcher{latest: release}
	translator := &mockTranslator{out: []string{"訳1"}}
	sender := &mockSender{}
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	deps, _, _ := newTestDeps(cfg, state, fetcher, translator, sender, stdout, stderr)

	if err := Run(deps); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if stdout.Len() == 0 {
		t.Fatal("expected progress lines on stdout, got none")
	}
	if stderr.Len() != 0 {
		t.Errorf("expected no stderr output on success, got %q", stderr.String())
	}

	tzTimestamp := regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(Z|[+-]\d{2}:\d{2})`)
	found := false
	for _, line := range strings.Split(stdout.String(), "\n") {
		if line == "" {
			continue
		}
		match := tzTimestamp.FindString(line)
		if match == "" {
			continue
		}
		found = true
		if _, err := time.Parse(time.RFC3339, match); err != nil {
			t.Errorf("timestamp %q in line %q is not a valid RFC3339 timestamp with timezone: %v", match, line, err)
		}
	}
	if !found {
		t.Errorf("no timestamp with an explicit timezone designation found in stdout: %q", stdout.String())
	}

	// Now force a failure to assert error output lands on stderr.
	stdout2, stderr2 := &bytes.Buffer{}, &bytes.Buffer{}
	failFetcher := &mockFetcher{latestErr: errors.New("network down")}
	deps2, _, _ := newTestDeps(cfg, &mockStateStore{found: false}, failFetcher, translator, sender, stdout2, stderr2)
	if err := Run(deps2); err == nil {
		t.Fatal("Run() error = nil, want error")
	}
	if stderr2.Len() == 0 {
		t.Error("expected error output on stderr, got none")
	}
}

// AC-7: go.mod requires no third-party module other than the YAML parser
// (NFR1), inspected directly from the worktree's module file.
func TestGoMod_RequiresOnlyYAMLThirdPartyModule(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "go.mod"))
	if err != nil {
		t.Fatalf("reading go.mod: %v", err)
	}

	const allowed = "gopkg.in/yaml.v3"
	inRequireBlock := false
	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		switch {
		case line == "":
			continue
		case strings.HasPrefix(line, "module "), strings.HasPrefix(line, "go "), strings.HasPrefix(line, "toolchain "):
			continue
		case line == "require (":
			inRequireBlock = true
		case inRequireBlock && line == ")":
			inRequireBlock = false
		case inRequireBlock, strings.HasPrefix(line, "require "):
			fields := strings.Fields(strings.TrimPrefix(line, "require "))
			if len(fields) == 0 {
				t.Errorf("unparseable require line in go.mod: %q", rawLine)
				continue
			}
			if fields[0] != allowed {
				t.Errorf("go.mod requires third-party module %q, want only %q", fields[0], allowed)
			}
		default:
			t.Errorf("unexpected line in go.mod: %q", rawLine)
		}
	}
}
