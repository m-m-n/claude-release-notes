// Package app is the run pipeline: fetch -> translate -> build -> send ->
// save state. It declares its own consumer-side interfaces for the five
// leaf roles (config, state, github, translate, mail) so the leaf packages
// stay import-free of each other and unit tests mock at this one seam. The
// entry point (cmd/claude-release-notes) wires the concrete implementations.
package app

import (
	"fmt"
	"io"
	"time"

	"claude-release-notes/internal/config"
	"claude-release-notes/internal/github"
	"claude-release-notes/internal/mail"
)

// ConfigLoader loads and validates application configuration.
type ConfigLoader interface {
	Load() (config.Config, error)
}

// ConfigLoaderFunc adapts a function to the ConfigLoader interface, letting
// the entry point bind config.Load to a fixed base directory.
type ConfigLoaderFunc func() (config.Config, error)

// Load calls f.
func (f ConfigLoaderFunc) Load() (config.Config, error) { return f() }

// StateStore persists the last delivered release version.
type StateStore interface {
	// Load returns (version, found=true) or (found=false) when no state
	// exists yet; it never errors for mere absence.
	Load() (version string, found bool, err error)
	Save(version string) error
}

// ReleaseFetcher fetches releases from GitHub.
type ReleaseFetcher interface {
	// Latest returns the newest non-draft, non-prerelease release.
	Latest() (github.Release, error)
	// Since returns the non-draft, non-prerelease releases newer than
	// version — newer decided by semantic-version comparison of tag
	// names — newest first, bounded to one listing page; empty slice
	// when up to date.
	Since(version string) ([]github.Release, error)
}

// Translator translates release bodies to Japanese via claude-batch.
type Translator interface {
	// Translate takes one string per release body (order preserved) and
	// returns a same-length, same-order slice of translated texts.
	Translate(sections []string) ([]string, error)
}

// MailSender delivers the built digest by email.
type MailSender interface {
	Send(to, subject, htmlBody string) error
}

// BuildFunc builds the HTML digest from items ordered newest first. It
// mirrors mail.Build's signature so the concrete function can be assigned
// directly by the entry point.
type BuildFunc func(items []mail.Item) (subject, htmlBody string)

// Dependencies wires the concrete (or mock, in tests) implementations of the
// five leaf roles into the run pipeline. NewFetcher and NewSender are
// factories rather than ready instances because their construction needs
// secrets (GitHub token, Gmail credentials) that only become available once
// Config has been loaded as the pipeline's first step.
type Dependencies struct {
	Config     ConfigLoader
	State      StateStore
	NewFetcher func(token string) ReleaseFetcher
	Translator Translator
	Build      BuildFunc
	NewSender  func(account, appPassword string) MailSender
	Stdout     io.Writer
	Stderr     io.Writer
}

// Run executes the fetch -> translate -> build -> send -> save-state
// pipeline. The order is the contract (NFR4): state is saved only after a
// successful send. It returns nil on success (including the "no new
// releases" case); any failure is wrapped with its stage's prefix
// ("config: ", "state: ", "github: ", "translate: ", "mail: ") and written to
// Stderr before being returned.
func Run(deps Dependencies) error {
	logProgress(deps.Stdout, "config: loading configuration")
	cfg, err := deps.Config.Load()
	if err != nil {
		return stageError(deps.Stderr, "config", err)
	}

	logProgress(deps.Stdout, "state: loading last delivered version")
	lastVersion, found, err := deps.State.Load()
	if err != nil {
		return stageError(deps.Stderr, "state", err)
	}

	fetcher := deps.NewFetcher(cfg.GitHubToken)

	var releases []github.Release
	if found {
		releases, err = fetcher.Since(lastVersion)
	} else {
		var latest github.Release
		latest, err = fetcher.Latest()
		if err == nil {
			releases = []github.Release{latest}
		}
	}
	if err != nil {
		return stageError(deps.Stderr, "github", err)
	}

	if len(releases) == 0 {
		logProgress(deps.Stdout, "github: no new releases")
		return nil
	}
	logProgress(deps.Stdout, "github: fetched %d release(s)", len(releases))

	bodies := make([]string, len(releases))
	for i, r := range releases {
		bodies[i] = r.Body
	}
	translations, err := deps.Translator.Translate(bodies)
	if err != nil {
		return stageError(deps.Stderr, "translate", err)
	}
	logProgress(deps.Stdout, "translate: translated %d release(s)", len(translations))

	items := make([]mail.Item, len(releases))
	for i, r := range releases {
		body := ""
		if i < len(translations) {
			body = translations[i]
		}
		items[i] = mail.Item{Version: r.TagName, PublishedAt: r.PublishedAt, Body: body}
	}
	subject, htmlBody := deps.Build(items)

	sender := deps.NewSender(cfg.GmailAccount, cfg.GmailAppPassword)
	if err := sender.Send(cfg.MailTo, subject, htmlBody); err != nil {
		return stageError(deps.Stderr, "mail", err)
	}
	logProgress(deps.Stdout, "mail: sent digest to %s", cfg.MailTo)

	maxRelease, ok := github.MaxByVersion(releases)
	if !ok {
		return stageError(deps.Stderr, "state", fmt.Errorf("no semver-parseable release among %d delivered", len(releases)))
	}
	if err := deps.State.Save(maxRelease.TagName); err != nil {
		return stageError(deps.Stderr, "state", err)
	}
	logProgress(deps.Stdout, "state: saved version %s", maxRelease.TagName)

	return nil
}

// logProgress writes one timestamped progress line to w. The timestamp is
// RFC3339 (always carries an explicit timezone designation, per NFR3).
func logProgress(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, "%s %s\n", time.Now().Format(time.RFC3339), fmt.Sprintf(format, args...))
}

// stageError wraps err with the stage prefix (Conventions), writes it to
// stderr, and returns the wrapped error for the caller to propagate.
func stageError(stderr io.Writer, stage string, err error) error {
	wrapped := fmt.Errorf("%s: %w", stage, err)
	if stderr != nil {
		fmt.Fprintln(stderr, wrapped)
	}
	return wrapped
}
