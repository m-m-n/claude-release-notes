package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
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

// TestSince_NewerThanStored references AC-1: Since returns exactly the
// releases newer than the stored version, newest first. It also references
// AC-2: draft/prerelease entries never appear. Only page 1 is read, so the
// second page of the mocked listing is not part of the expectation.
func TestSince_NewerThanStored(t *testing.T) {
	srv := newMockServer(t, mixedPages(), nil)
	defer srv.Close()

	client := NewClient("test-token", srv.URL)
	got, err := client.Since("v1.0.0")
	if err != nil {
		t.Fatalf("Since returned error: %v", err)
	}
	assertReleasesEqual(t, got, []string{"v3.0.0", "v2.5.0"})
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

// TestSince_StoredVersionDeletedFromListing references AC-4: the stored
// version having been deleted upstream is the normal case, not an error.
// Comparison still orders the listing against it, so only genuinely newer
// releases come back - never the whole listing.
func TestSince_StoredVersionDeletedFromListing(t *testing.T) {
	pages := [][]testRelease{
		{
			{TagName: "v3.0.0", Body: "b3", PublishedAt: "2024-03-01T00:00:00Z"},
			{TagName: "v2.5.0", Body: "b2.5", PublishedAt: "2024-02-15T00:00:00Z"},
			{TagName: "v2.0.0", Body: "b2", PublishedAt: "2024-02-01T00:00:00Z"},
		},
	}
	srv := newMockServer(t, pages, nil)
	defer srv.Close()

	client := NewClient("test-token", srv.URL)
	// v2.4.0 never appears in the listing; v2.5.0 and v3.0.0 outrank it.
	got, err := client.Since("v2.4.0")
	if err != nil {
		t.Fatalf("Since returned error: %v", err)
	}
	assertReleasesEqual(t, got, []string{"v3.0.0", "v2.5.0"})
}

// TestSince_ReadsOnlyFirstPage pins the bound that keeps one run's payload
// small: Since must never walk pagination, however far behind the stored
// version is.
func TestSince_ReadsOnlyFirstPage(t *testing.T) {
	var pagesRequested []string
	srv := newMockServer(t, mixedPages(), func(r *http.Request) {
		pagesRequested = append(pagesRequested, r.URL.Query().Get("page"))
	})
	defer srv.Close()

	client := NewClient("test-token", srv.URL)
	if _, err := client.Since("v0.0.1"); err != nil {
		t.Fatalf("Since returned error: %v", err)
	}
	if len(pagesRequested) != 1 || pagesRequested[0] != "1" {
		t.Errorf("pages requested = %v, want exactly [1]", pagesRequested)
	}
}

// TestSince_RequestsPerPageBound asserts the listing request carries the
// per-page cap, which is what actually limits one run's payload.
func TestSince_RequestsPerPageBound(t *testing.T) {
	var perPageRequested string
	srv := newMockServer(t, mixedPages(), func(r *http.Request) {
		perPageRequested = r.URL.Query().Get("per_page")
	})
	defer srv.Close()

	client := NewClient("test-token", srv.URL)
	if _, err := client.Since("v0.0.1"); err != nil {
		t.Fatalf("Since returned error: %v", err)
	}
	if want := strconv.Itoa(perPage); perPageRequested != want {
		t.Errorf("per_page = %q, want %q", perPageRequested, want)
	}
}

// TestSince_UnparsableTagIsSkipped covers tags that are not
// vMAJOR.MINOR.PATCH: they are treated as not newer and dropped, rather
// than aborting the run.
func TestSince_UnparsableTagIsSkipped(t *testing.T) {
	pages := [][]testRelease{
		{
			{TagName: "nightly", Body: "n", PublishedAt: "2024-03-02T00:00:00Z"},
			{TagName: "v3.0.0", Body: "b3", PublishedAt: "2024-03-01T00:00:00Z"},
			{TagName: "v2.9", Body: "short", PublishedAt: "2024-02-20T00:00:00Z"},
			{TagName: "v2.5.0-rc1", Body: "suffix", PublishedAt: "2024-02-16T00:00:00Z"},
			{TagName: "v2.5.0", Body: "b2.5", PublishedAt: "2024-02-15T00:00:00Z"},
		},
	}
	srv := newMockServer(t, pages, nil)
	defer srv.Close()

	client := NewClient("test-token", srv.URL)
	got, err := client.Since("v2.0.0")
	if err != nil {
		t.Fatalf("Since returned error: %v", err)
	}
	assertReleasesEqual(t, got, []string{"v3.0.0", "v2.5.0"})
}

// TestSince_UnparsableStoredVersionIsError covers a corrupted state file:
// with no parsable baseline there is no safe answer, so Since fails loudly
// instead of falling back to delivering the whole listing.
func TestSince_UnparsableStoredVersionIsError(t *testing.T) {
	srv := newMockServer(t, mixedPages(), nil)
	defer srv.Close()

	client := NewClient("test-token", srv.URL)
	_, err := client.Since("not-a-version")
	if err == nil {
		t.Fatal("Since() = nil error, want error on unparsable stored version")
	}
	if !strings.Contains(err.Error(), "not-a-version") {
		t.Errorf("Since() error = %q, want it to name the offending value", err.Error())
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		tag  string
		want version
		ok   bool
	}{
		{tag: "v2.1.243", want: version{2, 1, 243}, ok: true},
		{tag: "2.1.243", want: version{2, 1, 243}, ok: true},
		{tag: "v0.0.0", want: version{0, 0, 0}, ok: true},
		{tag: "v2.1", ok: false},
		{tag: "v2.1.2.3", ok: false},
		{tag: "v2.1.243-rc1", ok: false},
		{tag: "nightly", ok: false},
		{tag: "v2..3", ok: false},
		{tag: "v2.1.+3", ok: false},
		{tag: "v2.1.-3", ok: false},
		{tag: "", ok: false},
	}
	for _, tt := range tests {
		got, ok := parseVersion(tt.tag)
		if ok != tt.ok {
			t.Errorf("parseVersion(%q) ok = %v, want %v", tt.tag, ok, tt.ok)
			continue
		}
		if ok && got != tt.want {
			t.Errorf("parseVersion(%q) = %+v, want %+v", tt.tag, got, tt.want)
		}
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		name string
		a, b version
		want bool
	}{
		{name: "patch greater", a: version{2, 1, 244}, b: version{2, 1, 243}, want: true},
		{name: "patch lesser", a: version{2, 1, 242}, b: version{2, 1, 243}, want: false},
		{name: "equal", a: version{2, 1, 243}, b: version{2, 1, 243}, want: false},
		{name: "minor outranks patch", a: version{2, 2, 0}, b: version{2, 1, 999}, want: true},
		{name: "major outranks minor", a: version{3, 0, 0}, b: version{2, 9, 9}, want: true},
		{name: "older major", a: version{1, 9, 9}, b: version{2, 0, 0}, want: false},
	}
	for _, tt := range tests {
		if got := isNewer(tt.a, tt.b); got != tt.want {
			t.Errorf("%s: isNewer(%+v, %+v) = %v, want %v", tt.name, tt.a, tt.b, got, tt.want)
		}
	}
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

// TestMaxByVersion references AC-1 (task0002): MaxByVersion selects the
// release whose tag name is the greatest parsed version among the inputs
// that parse, resolves ties to the earliest such input, ignores tags that do
// not parse, and reports false on an empty or all-unparsable input. It must
// not reorder or mutate the slice it is given.
func TestMaxByVersion(t *testing.T) {
	tests := []struct {
		name     string
		releases []Release
		wantTag  string
		wantBody string // checked only when non-empty, to distinguish ties by identity
		wantOK   bool
	}{
		{
			name: "maximum at head",
			releases: []Release{
				{TagName: "v3.0.0"},
				{TagName: "v1.0.0"},
				{TagName: "v2.0.0"},
			},
			wantTag: "v3.0.0",
			wantOK:  true,
		},
		{
			name: "maximum at tail",
			releases: []Release{
				{TagName: "v1.0.0"},
				{TagName: "v2.0.0"},
				{TagName: "v3.0.0"},
			},
			wantTag: "v3.0.0",
			wantOK:  true,
		},
		{
			name:     "single element",
			releases: []Release{{TagName: "v1.2.3"}},
			wantTag:  "v1.2.3",
			wantOK:   true,
		},
		{
			name: "equal versions - earlier input wins",
			releases: []Release{
				{TagName: "v2.0.0", Body: "earlier"},
				{TagName: "v2.0.0", Body: "later"},
			},
			wantTag:  "v2.0.0",
			wantBody: "earlier",
			wantOK:   true,
		},
		{
			name: "unparsable tag is ignored",
			releases: []Release{
				{TagName: "nightly"},
				{TagName: "v1.0.0"},
			},
			wantTag: "v1.0.0",
			wantOK:  true,
		},
		{
			name:     "empty input reports nothing found",
			releases: nil,
			wantOK:   false,
		},
		{
			name: "all-unparsable input reports nothing found",
			releases: []Release{
				{TagName: "nightly"},
				{TagName: "v1.2"},
			},
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := append([]Release(nil), tt.releases...)

			got, ok := MaxByVersion(tt.releases)

			if ok != tt.wantOK {
				t.Fatalf("MaxByVersion() ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && got.TagName != tt.wantTag {
				t.Errorf("MaxByVersion().TagName = %q, want %q", got.TagName, tt.wantTag)
			}
			if ok && tt.wantBody != "" && got.Body != tt.wantBody {
				t.Errorf("MaxByVersion().Body = %q, want %q (tie should resolve to earliest input)", got.Body, tt.wantBody)
			}
			if !ok && got != (Release{}) {
				t.Errorf("MaxByVersion() = %+v on not-found, want zero value", got)
			}
			if !reflect.DeepEqual(tt.releases, original) {
				t.Errorf("MaxByVersion reordered or mutated its input: got %v, want %v", tt.releases, original)
			}
		})
	}
}

// TestLatest_SkipsUnparsableTagAtHead references AC-2 (task0002): a
// non-draft, non-prerelease release with an unparsable tag at the head of
// the listing is skipped in favor of the next parseable release.
func TestLatest_SkipsUnparsableTagAtHead(t *testing.T) {
	pages := [][]testRelease{
		{
			{TagName: "nightly", Body: "n", PublishedAt: "2024-03-02T00:00:00Z"},
			{TagName: "v3.0.0", Body: "b3", PublishedAt: "2024-03-01T00:00:00Z"},
		},
	}
	srv := newMockServer(t, pages, nil)
	defer srv.Close()

	client := NewClient("test-token", srv.URL)
	got, err := client.Latest()
	if err != nil {
		t.Fatalf("Latest returned error: %v", err)
	}
	if got.TagName != "v3.0.0" {
		t.Errorf("Latest().TagName = %q, want %q", got.TagName, "v3.0.0")
	}
}

// TestLatest_NoParseableCandidateIsError references AC-2 (task0002): when no
// non-draft, non-prerelease, parseable-tag entry exists in the listing -
// including when the only parseable tags belong to a draft and a prerelease
// - Latest returns the existing "no releases found" error instead of a
// filtered entry.
func TestLatest_NoParseableCandidateIsError(t *testing.T) {
	pages := [][]testRelease{
		{
			{TagName: "nightly", Body: "n", PublishedAt: "2024-03-02T00:00:00Z"},
			{TagName: "v3.0.0", Body: "d", PublishedAt: "2024-03-01T00:00:00Z", Draft: true},
			{TagName: "v2.9.0", Body: "b", PublishedAt: "2024-02-20T00:00:00Z", Prerelease: true},
		},
	}
	srv := newMockServer(t, pages, nil)
	defer srv.Close()

	client := NewClient("test-token", srv.URL)
	_, err := client.Latest()
	if err == nil {
		t.Fatal("Latest() = nil error, want error when no non-draft, non-prerelease, parseable-tag release exists")
	}
	if !strings.Contains(err.Error(), "no releases found") {
		t.Errorf("Latest() error = %q, want the existing \"no releases found\" error", err.Error())
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
