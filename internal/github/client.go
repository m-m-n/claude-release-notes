// Package github fetches anthropics/claude-code releases from the GitHub
// REST API.
package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://api.github.com"
	releasesPath   = "/repos/anthropics/claude-code/releases"
	// perPage bounds one listing page and, because Since reads only the
	// first page, also caps how many releases a single run can deliver.
	perPage        = 10
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

// Latest returns the newest non-draft, non-prerelease release whose tag name
// parses as vMAJOR.MINOR.PATCH. It returns an error if pagination is
// exhausted without finding one.
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
			if _, ok := parseVersion(r.TagName); !ok {
				continue
			}
			return toRelease(r), nil
		}
		page++
	}
}

// Since returns the non-draft, non-prerelease releases newer than version,
// newest first. It reads only the first listing page, so at most perPage
// releases are ever returned.
//
// "Newer" is a semantic-version comparison of tag names, not a search for
// version itself: upstream deletes releases, so the stored version is
// routinely absent from the listing, and an identity match would then find
// no stopping point. Tags that do not parse as vMAJOR.MINOR.PATCH are
// treated as not newer and skipped.
func (c *Client) Since(version string) ([]Release, error) {
	stored, ok := parseVersion(version)
	if !ok {
		return nil, fmt.Errorf("github: unparsable stored version %q", version)
	}

	releases, err := c.fetchPage(1)
	if err != nil {
		return nil, err
	}

	result := []Release{}
	for _, r := range releases {
		if r.Draft || r.Prerelease {
			continue
		}
		v, ok := parseVersion(r.TagName)
		if !ok {
			continue
		}
		if isNewer(v, stored) {
			result = append(result, toRelease(r))
		}
	}
	return result, nil
}

// MaxByVersion returns the release among releases whose tag name is the
// greatest parsed version, together with true. Releases whose tag does not
// parse as vMAJOR.MINOR.PATCH are ignored. On equal parsed versions the
// earliest such input wins, making the result deterministic for a listing
// that repeats a version. MaxByVersion performs no I/O and does not modify
// or reorder releases; an empty input, or one in which no tag parses,
// returns the zero Release value and false.
func MaxByVersion(releases []Release) (Release, bool) {
	var (
		best  Release
		bestV version
		found bool
	)
	for _, r := range releases {
		v, ok := parseVersion(r.TagName)
		if !ok {
			continue
		}
		if !found || isNewer(v, bestV) {
			best = r
			bestV = v
			found = true
		}
	}
	return best, found
}

// version is a parsed vMAJOR.MINOR.PATCH tag name.
type version struct {
	major, minor, patch int
}

// parseVersion parses a "vMAJOR.MINOR.PATCH" tag name, with the leading "v"
// optional. It reports false for every other shape, including tags carrying
// a pre-release or build suffix.
func parseVersion(tag string) (version, bool) {
	parts := strings.Split(strings.TrimPrefix(tag, "v"), ".")
	if len(parts) != 3 {
		return version{}, false
	}

	nums := make([]int, len(parts))
	for i, p := range parts {
		// Atoi accepts a leading sign, which is not a version component.
		if p == "" || (p[0] != '0' && !(p[0] >= '1' && p[0] <= '9')) {
			return version{}, false
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return version{}, false
		}
		nums[i] = n
	}
	return version{major: nums[0], minor: nums[1], patch: nums[2]}, true
}

// isNewer reports whether a is strictly newer than b.
func isNewer(a, b version) bool {
	if a.major != b.major {
		return a.major > b.major
	}
	if a.minor != b.minor {
		return a.minor > b.minor
	}
	return a.patch > b.patch
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
