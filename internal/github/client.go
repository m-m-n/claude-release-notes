// Placeholder: compile-only stub for internal/github, added by task0006 so
// internal/app and cmd/claude-release-notes can compile and be wired in this
// worktree. Real implementation is owned by task0003 and replaces this file
// on merge (parent-side adoption protocol).
package github

import "time"

// Release is this package's own minimal value type (no shared Release type
// per IMPLEMENTATION.md's cross-task decision).
type Release struct {
	TagName     string
	Body        string
	PublishedAt time.Time
}

// Client fetches releases from the GitHub Releases API.
type Client struct {
	token   string
	baseURL string
}

// NewClient constructs a Client with the given auth token and API base URL.
// An empty baseURL means "use the default GitHub API host".
func NewClient(token, baseURL string) *Client {
	return &Client{token: token, baseURL: baseURL}
}

// Latest returns the newest non-draft, non-prerelease release. Placeholder:
// always returns a zero Release and a nil error.
func (c *Client) Latest() (Release, error) {
	return Release{}, nil
}

// Since returns all non-draft, non-prerelease releases newer than version,
// newest first, following pagination; empty slice when up to date.
// Placeholder: always returns an empty slice and a nil error.
func (c *Client) Since(version string) ([]Release, error) {
	return nil, nil
}
