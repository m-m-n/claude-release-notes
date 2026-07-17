package mail

import (
	"strings"
	"testing"
	"time"
)

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 3, 0, 0, 0, time.UTC)
}

// TestBuild_SubjectFormat references AC-1 (subject format for N items and a
// single item).
func TestBuild_SubjectFormat(t *testing.T) {
	cases := []struct {
		name  string
		items []Item
		want  string
	}{
		{
			name: "single item subject has no ほか suffix",
			items: []Item{
				{Version: "v1.2.3", PublishedAt: date(2026, 7, 16), Body: "本文"},
			},
			want: "Claude Code リリースノート: v1.2.3",
		},
		{
			name: "multiple items subject counts the remainder",
			items: []Item{
				{Version: "v1.2.3", PublishedAt: date(2026, 7, 16), Body: "本文1"},
				{Version: "v1.2.2", PublishedAt: date(2026, 7, 10), Body: "本文2"},
				{Version: "v1.2.1", PublishedAt: date(2026, 7, 1), Body: "本文3"},
			},
			want: "Claude Code リリースノート: v1.2.3 ほか 2 件",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			subject, _ := Build(tc.items)
			if subject != tc.want {
				t.Errorf("subject = %q, want %q", subject, tc.want)
			}
		})
	}
}

// TestBuild_RendersEveryItemInOrder references AC-1 (every item rendered,
// in given order, each with version/date/body).
func TestBuild_RendersEveryItemInOrder(t *testing.T) {
	items := []Item{
		{Version: "v2.0.0", PublishedAt: date(2026, 7, 16), Body: "新しい本文"},
		{Version: "v1.0.0", PublishedAt: date(2026, 6, 1), Body: "古い本文"},
	}

	_, html := Build(items)

	for _, it := range items {
		if !strings.Contains(html, it.Version) {
			t.Errorf("html missing version %q", it.Version)
		}
		if !strings.Contains(html, it.Body) {
			t.Errorf("html missing body %q", it.Body)
		}
	}

	if !strings.Contains(html, "2026-07-16 公開") {
		t.Errorf("html missing formatted publish date for newest item: %s", html)
	}
	if !strings.Contains(html, "2026-06-01 公開") {
		t.Errorf("html missing formatted publish date for older item: %s", html)
	}

	if strings.Index(html, "v2.0.0") > strings.Index(html, "v1.0.0") {
		t.Errorf("items rendered out of order (newest must come first)")
	}
}

// TestBuild_PublishedDateUsesExplicitJSTNotSystemLocal locks in the design
// decision that the calendar day is computed from a fixed, explicit
// timezone (JST) rather than the host's system timezone, so a UTC/JST day
// boundary does not shift the rendered day depending on where the binary
// runs.
func TestBuild_PublishedDateUsesExplicitJSTNotSystemLocal(t *testing.T) {
	items := []Item{
		// 2026-07-15T16:00:00Z == 2026-07-16 01:00 JST.
		{Version: "v1.0.0", PublishedAt: time.Date(2026, 7, 15, 16, 0, 0, 0, time.UTC), Body: "本文"},
	}

	_, html := Build(items)

	if !strings.Contains(html, "2026-07-16 公開") {
		t.Errorf("expected JST calendar day 2026-07-16, got html: %s", html)
	}
	if strings.Contains(html, "2026-07-15 公開") {
		t.Errorf("date rendered using the wrong timezone (UTC day instead of JST): %s", html)
	}
}

// TestBuild_EscapesHTMLMetacharacters references AC-2 (HTML metacharacters
// in translated text appear escaped; no raw injection is possible).
func TestBuild_EscapesHTMLMetacharacters(t *testing.T) {
	items := []Item{
		{
			Version:     "v1.0.0",
			PublishedAt: date(2026, 7, 16),
			Body:        `<script>alert(1)</script> & "quotes" 'apostrophe'`,
		},
	}

	_, html := Build(items)

	if strings.Contains(html, "<script>") {
		t.Fatalf("raw <script> tag leaked into output: %s", html)
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Errorf("expected escaped script tag, got: %s", html)
	}
	if !strings.Contains(html, "&amp;") {
		t.Errorf("expected escaped ampersand, got: %s", html)
	}
}

// TestBuild_EmptyBodyRendersPlaceholder references AC-3 (an empty body item
// renders the placeholder text instead of an empty section).
func TestBuild_EmptyBodyRendersPlaceholder(t *testing.T) {
	items := []Item{
		{Version: "v1.0.0", PublishedAt: date(2026, 7, 16), Body: ""},
		{Version: "v0.9.0", PublishedAt: date(2026, 7, 1), Body: "   "},
	}

	_, html := Build(items)

	if got := strings.Count(html, emptyBodyPlaceholder); got != 2 {
		t.Errorf("expected placeholder to appear twice (empty + whitespace-only body), got %d in: %s", got, html)
	}
}

// TestBuild_HTMLIsSelfContainedAndInlineStyled references AC-4 (only inline
// style attributes, no <style> block, no external URLs; header band,
// divider rules, and footer are present).
func TestBuild_HTMLIsSelfContainedAndInlineStyled(t *testing.T) {
	items := []Item{
		{Version: "v1.0.0", PublishedAt: date(2026, 7, 16), Body: "本文"},
		{Version: "v0.9.0", PublishedAt: date(2026, 7, 1), Body: "本文2"},
	}

	_, html := Build(items)

	if strings.Contains(html, "<style") {
		t.Errorf("expected no <style> block, found one in: %s", html)
	}
	if strings.Contains(html, "http://") || strings.Contains(html, "https://") {
		t.Errorf("expected no external resource URLs in: %s", html)
	}
	if !strings.Contains(html, "background-color:#C15F3C") {
		t.Errorf("expected header band primary color in: %s", html)
	}
	if !strings.Contains(html, "background-color:#E4DED4") {
		t.Errorf("expected divider color rule in: %s", html)
	}
	if !strings.Contains(html, "claude-release-notes により自動送信") {
		t.Errorf("expected footer text in: %s", html)
	}
	if !strings.Contains(html, "2 件の新しいリリース") {
		t.Errorf("expected header caption with item count in: %s", html)
	}
}
