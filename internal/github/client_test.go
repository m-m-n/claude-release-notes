package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

// testRelease mirrors the GitHub API release JSON shape used by the mock
// server in these tests.
type testRelease struct {
	TagName     string `json:"tag_name"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
	Draft       bool   `json:"draft"`
	Prerelease  bool   `json:"prerelease"`
}

// newMockServer serves canned, paged release JSON. pages[0] is page 1
// (newest), pages[1] is page 2, etc. Requests beyond len(pages) receive an
// empty array, signaling pagination end. assertReq, if non-nil, is invoked
// for every incoming request so callers can assert on headers/URL.
func newMockServer(t *testing.T, pages [][]testRelease, assertReq func(*http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if assertReq != nil {
			assertReq(r)
		}
		page, err := strconv.Atoi(r.URL.Query().Get("page"))
		if err != nil || page < 1 {
			page = 1
		}
		idx := page - 1
		body := []testRelease{}
		if idx < len(pages) {
			body = pages[idx]
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(body); err != nil {
			t.Fatalf("encoding mock response: %v", err)
		}
	}))
}

func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("parsing time %q: %v", s, err)
	}
	return tm
}

// mixedPages spans two pages and interleaves draft/prerelease entries with
// real releases, covering AC-1 (multi-page) and AC-2 (draft/prerelease
// filtering) together.
func mixedPages() [][]testRelease {
	return [][]testRelease{
		{
			{TagName: "v3.1.0-draft", Body: "draft body", PublishedAt: "2024-03-15T00:00:00Z", Draft: true},
			{TagName: "v3.0.0", Body: "b3", PublishedAt: "2024-03-01T00:00:00Z"},
			{TagName: "v2.9.0-beta", Body: "beta body", PublishedAt: "2024-02-20T00:00:00Z", Prerelease: true},
			{TagName: "v2.5.0", Body: "b2.5", PublishedAt: "2024-02-15T00:00:00Z"},
		},
		{
			{TagName: "v2.0.0", Body: "b2", PublishedAt: "2024-02-01T00:00:00Z"},
			{TagName: "v1.0.0", Body: "b1", PublishedAt: "2024-01-01T00:00:00Z"},
		},
	}
}

func assertReleasesEqual(t *testing.T, got []Release, wantTags []string) {
	t.Helper()
	if len(got) != len(wantTags) {
		t.Fatalf("got %d releases %v, want %d releases %v", len(got), tagsOf(got), len(wantTags), wantTags)
	}
	for i, tag := range wantTags {
		if got[i].TagName != tag {
			t.Errorf("release[%d].TagName = %q, want %q", i, got[i].TagName, tag)
		}
	}
}

func tagsOf(rs []Release) []string {
	tags := make([]string, len(rs))
	for i, r := range rs {
		tags[i] = r.TagName
	}
	return tags
}

// TestSince_MultiPageNewerThanStored references AC-1: Since returns exactly
// the releases newer than the stored version, newest first, across a
// multi-page mocked listing. It also references AC-2: draft/prerelease
// entries never appear.
func TestSince_MultiPageNewerThanStored(t *testing.T) {
	srv := newMockServer(t, mixedPages(), nil)
	defer srv.Close()

	client := NewClient("test-token", srv.URL)
	got, err := client.Since("v1.0.0")
	if err != nil {
		t.Fatalf("Since returned error: %v", err)
	}
	assertReleasesEqual(t, got, []string{"v3.0.0", "v2.5.0", "v2.0.0"})
}

// TestSince_StoredVersionIsNewest references AC-3: Since with the stored
// version as the newest listing entry returns an empty slice.
func TestSince_StoredVersionIsNewest(t *testing.T) {
	pages := [][]testRelease{
		{
			{TagName: "v3.0.0", Body: "b3", PublishedAt: "2024-03-01T00:00:00Z"},
			{TagName: "v2.5.0", Body: "b2.5", PublishedAt: "2024-02-15T00:00:00Z"},
		},
	}
	srv := newMockServer(t, pages, nil)
	defer srv.Close()

	client := NewClient("test-token", srv.URL)
	got, err := client.Since("v3.0.0")
	if err != nil {
		t.Fatalf("Since returned error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d releases %v, want empty slice", len(got), tagsOf(got))
	}
}

// TestSince_VersionAbsentFromListing references AC-4: Since with a version
// absent from the listing returns all listed (filtered) releases.
func TestSince_VersionAbsentFromListing(t *testing.T) {
	srv := newMockServer(t, mixedPages(), nil)
	defer srv.Close()

	client := NewClient("test-token", srv.URL)
	got, err := client.Since("v0.1.0")
	if err != nil {
		t.Fatalf("Since returned error: %v", err)
	}
	assertReleasesEqual(t, got, []string{"v3.0.0", "v2.5.0", "v2.0.0", "v1.0.0"})
}

// TestLatest_ReturnsNewestFiltered references AC-5: Latest returns the
// newest filtered release, and AC-2: draft/prerelease never appear.
func TestLatest_ReturnsNewestFiltered(t *testing.T) {
	srv := newMockServer(t, mixedPages(), nil)
	defer srv.Close()

	client := NewClient("test-token", srv.URL)
	got, err := client.Latest()
	if err != nil {
		t.Fatalf("Latest returned error: %v", err)
	}
	if got.TagName != "v3.0.0" {
		t.Errorf("Latest().TagName = %q, want %q", got.TagName, "v3.0.0")
	}
	wantPublished := mustParseTime(t, "2024-03-01T00:00:00Z")
	if !got.PublishedAt.Equal(wantPublished) {
		t.Errorf("Latest().PublishedAt = %v, want %v", got.PublishedAt, wantPublished)
	}
	if got.Body != "b3" {
		t.Errorf("Latest().Body = %q, want %q", got.Body, "b3")
	}
}

// TestLatest_NoNonFilteredReleaseIsError covers the "none found after
// exhausting pagination" error path from the task design.
func TestLatest_NoNonFilteredReleaseIsError(t *testing.T) {
	pages := [][]testRelease{
		{
			{TagName: "v1.0.0-draft", Body: "d", PublishedAt: "2024-01-01T00:00:00Z", Draft: true},
			{TagName: "v0.9.0-beta", Body: "d", PublishedAt: "2023-12-01T00:00:00Z", Prerelease: true},
		},
	}
	srv := newMockServer(t, pages, nil)
	defer srv.Close()

	client := NewClient("test-token", srv.URL)
	_, err := client.Latest()
	if err == nil {
		t.Fatal("Latest() = nil error, want error when no non-draft/non-prerelease release exists")
	}
}

// TestRequests_CarryAuthAndAcceptHeaders references AC-6: requests carry the
// Bearer token and Accept headers.
func TestRequests_CarryAuthAndAcceptHeaders(t *testing.T) {
	const token = "sentinel-token-abc123"
	var sawAuth, sawAccept string
	srv := newMockServer(t, [][]testRelease{{{TagName: "v1.0.0", Body: "b", PublishedAt: "2024-01-01T00:00:00Z"}}}, func(r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		sawAccept = r.Header.Get("Accept")
	})
	defer srv.Close()

	client := NewClient(token, srv.URL)
	if _, err := client.Latest(); err != nil {
		t.Fatalf("Latest returned error: %v", err)
	}

	if want := "Bearer " + token; sawAuth != want {
		t.Errorf("Authorization header = %q, want %q", sawAuth, want)
	}
	if sawAccept != "application/vnd.github+json" {
		t.Errorf("Accept header = %q, want %q", sawAccept, "application/vnd.github+json")
	}
}

// TestNon2xxResponse_ErrorHasStatusNotToken references AC-7: a non-2xx API
// response yields an error containing the status but not the token value.
func TestNon2xxResponse_ErrorHasStatusNotToken(t *testing.T) {
	const sentinelToken = "super-secret-token-xyz"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"Bad credentials"}`))
	}))
	defer srv.Close()

	client := NewClient(sentinelToken, srv.URL)

	_, latestErr := client.Latest()
	if latestErr == nil {
		t.Fatal("Latest() = nil error, want error on non-2xx response")
	}
	if !strings.Contains(latestErr.Error(), "403") {
		t.Errorf("Latest() error = %q, want it to contain status 403", latestErr.Error())
	}
	if strings.Contains(latestErr.Error(), sentinelToken) {
		t.Errorf("Latest() error = %q, must not contain the token", latestErr.Error())
	}

	_, sinceErr := client.Since("v1.0.0")
	if sinceErr == nil {
		t.Fatal("Since() = nil error, want error on non-2xx response")
	}
	if !strings.Contains(sinceErr.Error(), "403") {
		t.Errorf("Since() error = %q, want it to contain status 403", sinceErr.Error())
	}
	if strings.Contains(sinceErr.Error(), sentinelToken) {
		t.Errorf("Since() error = %q, must not contain the token", sinceErr.Error())
	}
}
