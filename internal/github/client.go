// Package github fetches anthropics/claude-code releases from the GitHub
// REST API.
package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultBaseURL = "https://api.github.com"
	releasesPath   = "/repos/anthropics/claude-code/releases"
	perPage        = 30
	requestTimeout = 15 * time.Second
)

// Release is a single Claude Code release. Only non-draft, non-prerelease
// releases are ever returned by Client's methods.
type Release struct {
	TagName     string
	Body        string
	PublishedAt time.Time
}

// Client fetches releases of anthropics/claude-code from the GitHub REST
// API.
type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a Client authenticating with token. baseURL overrides
// the default GitHub API base URL; pass "" to use the default
// (https://api.github.com). Tests point baseURL at a local test server.
func NewClient(token, baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		token:      token,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: requestTimeout},
	}
}

// apiRelease mirrors the GitHub API release JSON shape.
type apiRelease struct {
	TagName     string    `json:"tag_name"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"published_at"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
}

// Latest returns the newest non-draft, non-prerelease release. It returns an
// error if pagination is exhausted without finding one.
func (c *Client) Latest() (Release, error) {
	page := 1
	for {
		releases, err := c.fetchPage(page)
		if err != nil {
			return Release{}, err
		}
		if len(releases) == 0 {
			return Release{}, fmt.Errorf("github: no releases found")
		}
		for _, r := range releases {
			if r.Draft || r.Prerelease {
				continue
			}
			return toRelease(r), nil
		}
		page++
	}
}

// Since returns all non-draft, non-prerelease releases newer than version,
// newest first. Collection walks pages newest to oldest and stops
// (exclusive) at the first release whose tag name equals version. If
// version is never found, all listed (filtered) releases are returned.
func (c *Client) Since(version string) ([]Release, error) {
	result := []Release{}
	page := 1
	for {
		releases, err := c.fetchPage(page)
		if err != nil {
			return nil, err
		}
		if len(releases) == 0 {
			return result, nil
		}
		for _, r := range releases {
			if r.TagName == version {
				return result, nil
			}
			if r.Draft || r.Prerelease {
				continue
			}
			result = append(result, toRelease(r))
		}
		page++
	}
}

func toRelease(r apiRelease) Release {
	return Release{TagName: r.TagName, Body: r.Body, PublishedAt: r.PublishedAt}
}

// fetchPage retrieves and decodes one page of the releases listing.
func (c *Client) fetchPage(page int) ([]apiRelease, error) {
	url := fmt.Sprintf("%s%s?per_page=%d&page=%d", c.baseURL, releasesPath, perPage, page)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("github: building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("github: unexpected response status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("github: reading response: %w", err)
	}

	var releases []apiRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, fmt.Errorf("github: decoding response: %w", err)
	}
	return releases, nil
}
