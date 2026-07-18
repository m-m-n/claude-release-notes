package mail

import (
	"strings"
	"testing"
)

// TestRenderMarkdown_Headings references AC-1: ATX headings (1-3 #) all map
// to the single sub-heading style, and the first rendered element carries
// no top margin.
func TestRenderMarkdown_Headings(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"level 1", "# Title"},
		{"level 2", "## Title"},
		{"level 3", "### Title"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			html := renderMarkdown(tc.input)
			for _, want := range []string{"font-weight:700", "color:#232A31", "Title"} {
				if !strings.Contains(html, want) {
					t.Errorf("expected %q in heading html: %s", want, html)
				}
			}
			if !strings.Contains(html, "margin:0 0 8px 0") {
				t.Errorf("expected no-top-margin as first element in: %s", html)
			}
		})
	}
}

// TestRenderMarkdown_HeadingAfterOtherContentGetsTopMargin references AC-1:
// a heading that is not the first rendered element gets the 24px top
// margin.
func TestRenderMarkdown_HeadingAfterOtherContentGetsTopMargin(t *testing.T) {
	html := renderMarkdown("intro paragraph\n\n# Title")
	if !strings.Contains(html, "margin:24px 0 8px 0") {
		t.Errorf("expected 24px top margin for non-first heading in: %s", html)
	}
}

// TestRenderMarkdown_UnorderedList references AC-1: consecutive `- ` / `* `
// lines form one unordered list, rendered in source order.
func TestRenderMarkdown_UnorderedList(t *testing.T) {
	html := renderMarkdown("- first\n- second")
	if !strings.Contains(html, "<ul") || !strings.Contains(html, "</ul>") {
		t.Fatalf("expected <ul> element in: %s", html)
	}
	if !strings.Contains(html, "padding-left:24px") {
		t.Errorf("expected 24px left padding on list in: %s", html)
	}
	if strings.Index(html, "first") > strings.Index(html, "second") {
		t.Errorf("expected items in source order in: %s", html)
	}
	if got := strings.Count(html, "<li"); got != 2 {
		t.Errorf("expected 2 list items, got %d in html: %s", got, html)
	}
}

// TestRenderMarkdown_OrderedList references AC-1: consecutive `N. ` lines
// form one ordered list.
func TestRenderMarkdown_OrderedList(t *testing.T) {
	html := renderMarkdown("1. first\n2. second")
	if !strings.Contains(html, "<ol") || !strings.Contains(html, "</ol>") {
		t.Fatalf("expected <ol> element in: %s", html)
	}
	if strings.Index(html, "first") > strings.Index(html, "second") {
		t.Errorf("expected items in source order in: %s", html)
	}
}

// TestRenderMarkdown_InlineCode references AC-1: backtick spans render as
// styled <code> elements.
func TestRenderMarkdown_InlineCode(t *testing.T) {
	html := renderMarkdown("use `foo()` now")
	if !strings.Contains(html, "<code") {
		t.Fatalf("expected <code> element in: %s", html)
	}
	if !strings.Contains(html, "foo()") {
		t.Errorf("expected code content preserved in: %s", html)
	}
	if !strings.Contains(html, "background-color:#EDF1F5") {
		t.Errorf("expected surface-muted background on inline code in: %s", html)
	}
}

// TestRenderMarkdown_Bold references AC-1: `**text**` renders as a styled
// <strong> element.
func TestRenderMarkdown_Bold(t *testing.T) {
	html := renderMarkdown("this is **important** text")
	if !strings.Contains(html, "<strong") {
		t.Fatalf("expected <strong> element in: %s", html)
	}
	if !strings.Contains(html, "important") {
		t.Errorf("expected bold content preserved in: %s", html)
	}
	if !strings.Contains(html, "font-weight:700") {
		t.Errorf("expected bold weight style in: %s", html)
	}
}

// TestRenderMarkdown_HTTPLink references AC-1: `[text](url)` with an
// http/https scheme renders as a styled, underlined <a> element.
func TestRenderMarkdown_HTTPLink(t *testing.T) {
	html := renderMarkdown("see [docs](https://example.com/docs)")
	if !strings.Contains(html, `href="https://example.com/docs"`) {
		t.Fatalf("expected href attribute in: %s", html)
	}
	if !strings.Contains(html, "docs</a>") {
		t.Errorf("expected link text in: %s", html)
	}
	if !strings.Contains(html, "text-decoration:underline") || !strings.Contains(html, "color:#33658A") {
		t.Errorf("expected primary-colored underlined link style in: %s", html)
	}
}

// TestRenderMarkdown_FencedCodeBlock references AC-1: a closed fenced block
// renders as a styled <pre> element.
func TestRenderMarkdown_FencedCodeBlock(t *testing.T) {
	html := renderMarkdown("```\ncode line\n```")
	if !strings.Contains(html, "<pre") {
		t.Fatalf("expected <pre> element in: %s", html)
	}
	if !strings.Contains(html, "code line") {
		t.Errorf("expected code block content preserved in: %s", html)
	}
	if !strings.Contains(html, "background-color:#EDF1F5") {
		t.Errorf("expected surface-muted background on code block in: %s", html)
	}
}

// TestRenderMarkdown_MixedDocumentRendersInSourceOrder references AC-1: a
// document combining every construct renders each element in source order.
func TestRenderMarkdown_MixedDocumentRendersInSourceOrder(t *testing.T) {
	input := "# Heading\n\nSome paragraph text.\n\n- item one\n- item two\n\n```\nblock code\n```"
	html := renderMarkdown(input)

	positions := []int{
		strings.Index(html, "Heading"),
		strings.Index(html, "Some paragraph text."),
		strings.Index(html, "item one"),
		strings.Index(html, "block code"),
	}
	for i := 1; i < len(positions); i++ {
		if positions[i-1] < 0 || positions[i] < 0 || positions[i-1] > positions[i] {
			t.Fatalf("expected elements in source order, got positions %v in: %s", positions, html)
		}
	}
}

// TestRenderMarkdown_PlainTextNoMarkdown references AC-2: Markdown-free text
// renders as an escaped plain paragraph with no content loss.
func TestRenderMarkdown_PlainTextNoMarkdown(t *testing.T) {
	html := renderMarkdown("plain sentence with no markup")
	if !strings.Contains(html, "<p") {
		t.Fatalf("expected <p> element in: %s", html)
	}
	if !strings.Contains(html, "plain sentence with no markup") {
		t.Errorf("expected full text preserved in: %s", html)
	}
}

// TestRenderMarkdown_UnsupportedHeadingLevelDegradesToParagraph references
// AC-2: a 4+ hash heading is unsupported and degrades to an escaped plain
// paragraph, with the literal hashes preserved (no content loss).
func TestRenderMarkdown_UnsupportedHeadingLevelDegradesToParagraph(t *testing.T) {
	html := renderMarkdown("#### too deep")
	if strings.Contains(html, "font-weight:700;color:#232A31") {
		t.Errorf("expected no heading style applied, got: %s", html)
	}
	if !strings.Contains(html, "#### too deep") {
		t.Errorf("expected literal hashes preserved in plain paragraph: %s", html)
	}
	if !strings.Contains(html, "<p") {
		t.Errorf("expected fallback to plain paragraph in: %s", html)
	}
}

// TestRenderMarkdown_BlankLineSeparatesParagraphs references AC-2: a blank
// line separates two paragraphs into distinct elements.
func TestRenderMarkdown_BlankLineSeparatesParagraphs(t *testing.T) {
	html := renderMarkdown("para one\n\npara two")
	if got := strings.Count(html, "<p"); got != 2 {
		t.Fatalf("expected 2 separate paragraphs, got %d in html: %s", got, html)
	}
	if strings.Index(html, "para one") > strings.Index(html, "para two") {
		t.Errorf("expected paragraphs in source order in: %s", html)
	}
}

// TestRenderMarkdown_BlankLineTerminatesList references AC-2: a blank line
// terminates a list run; a following list-item line starts a new list
// rather than continuing the old one.
func TestRenderMarkdown_BlankLineTerminatesList(t *testing.T) {
	html := renderMarkdown("- item1\n\n- item2")
	if got := strings.Count(html, "<ul"); got != 2 {
		t.Fatalf("expected 2 separate <ul> lists, got %d in html: %s", got, html)
	}
}

// TestRenderMarkdown_ConsecutivePlainLinesJoinWithLineBreak references AC-2
// and FR1: consecutive plain lines accumulate into one paragraph joined by
// line breaks, losing no content.
func TestRenderMarkdown_ConsecutivePlainLinesJoinWithLineBreak(t *testing.T) {
	html := renderMarkdown("line1\nline2")
	if got := strings.Count(html, "<p"); got != 1 {
		t.Fatalf("expected a single paragraph wrapping both lines, got %d in html: %s", got, html)
	}
	if !strings.Contains(html, "line1<br>line2") {
		t.Errorf("expected line break between accumulated lines in: %s", html)
	}
}

// TestRenderMarkdown_RawHTMLIsEscaped references AC-3: raw HTML in the body
// appears escaped in the output, never as live markup.
func TestRenderMarkdown_RawHTMLIsEscaped(t *testing.T) {
	html := renderMarkdown("this has <b>raw html</b> inside")
	if strings.Contains(html, "<b>raw html</b>") {
		t.Fatalf("raw HTML leaked into output unescaped: %s", html)
	}
	if !strings.Contains(html, "&lt;b&gt;raw html&lt;/b&gt;") {
		t.Errorf("expected escaped HTML in: %s", html)
	}
}

// TestRenderMarkdown_NonHTTPLinkSchemesNeverEmitHref references AC-3: link
// hrefs must parse as http/https URLs; any other scheme renders as escaped
// plain text with no <a> element.
func TestRenderMarkdown_NonHTTPLinkSchemesNeverEmitHref(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"javascript scheme", "[click](javascript:alert(1))"},
		{"mailto scheme", "[mail](mailto:a@example.com)"},
		{"relative path", "[home](/path)"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			html := renderMarkdown(tc.input)
			if strings.Contains(html, "<a ") || strings.Contains(html, "href=") {
				t.Errorf("expected no <a> element for unsupported scheme, got: %s", html)
			}
		})
	}
}

// TestRenderMarkdown_UnclosedFencedBlockRunsToEndOfBody references AC-4: an
// unclosed fenced code block renders to the end of the body without error,
// and its content is never inline-parsed.
func TestRenderMarkdown_UnclosedFencedBlockRunsToEndOfBody(t *testing.T) {
	html := renderMarkdown("```\nline1\n**not bold**\nline2")

	if !strings.Contains(html, "<pre") {
		t.Fatalf("expected <pre> element in: %s", html)
	}
	if !strings.Contains(html, "line1") || !strings.Contains(html, "line2") {
		t.Errorf("expected all lines through end of body preserved in: %s", html)
	}
	if strings.Contains(html, "<strong") {
		t.Errorf("expected fenced content never inline-parsed, got: %s", html)
	}
	if !strings.Contains(html, "**not bold**") {
		t.Errorf("expected literal markdown markers preserved (escaped) in: %s", html)
	}
}

// TestRenderMarkdown_FencedContentNeverInlineParsed references AC-4: a
// closed fenced block's content is escaped verbatim, never inline-parsed
// (bold/link/code-span markers stay literal).
func TestRenderMarkdown_FencedContentNeverInlineParsed(t *testing.T) {
	html := renderMarkdown("```\n**bold** [link](https://x.com) `code`\n```")

	if strings.Contains(html, "<strong") || strings.Contains(html, "<a ") {
		t.Errorf("expected no inline parsing inside fenced block, got: %s", html)
	}
	if got := strings.Count(html, "<code"); got != 0 {
		t.Errorf("expected no additional inline <code> spans inside fenced block, got %d in: %s", got, html)
	}
	if !strings.Contains(html, "**bold**") || !strings.Contains(html, "`code`") {
		t.Errorf("expected literal markdown markers preserved in: %s", html)
	}
}

// TestRenderMarkdown_EmptyInputRendersEmpty locks in that the renderer
// itself produces no output for an empty body; the placeholder path is
// build.go's responsibility, not the renderer's.
func TestRenderMarkdown_EmptyInputRendersEmpty(t *testing.T) {
	if html := renderMarkdown(""); html != "" {
		t.Errorf("expected empty output for empty input, got: %s", html)
	}
}

// TestRenderMarkdown_CodeBlockDeclaresHorizontalOverflowScroll references
// AC-1: the rendered code block's inline style declares horizontal overflow
// scrolling, so a long line does not blow out the 800px mail column.
func TestRenderMarkdown_CodeBlockDeclaresHorizontalOverflowScroll(t *testing.T) {
	html := renderMarkdown("```\ncode line\n```")
	if !strings.Contains(html, "overflow-x:auto") {
		t.Errorf("expected overflow-x:auto on code block in: %s", html)
	}
}

// TestRenderMarkdown_ListItemsDeclareBodyTypography references AC-2: each
// list item declares the body typography (16px / 1.8 / on-surface #232A31)
// inline, since mail clients are unreliable about CSS inheritance.
func TestRenderMarkdown_ListItemsDeclareBodyTypography(t *testing.T) {
	html := renderMarkdown("- first\n- second")
	want := "font-size:16px;line-height:1.8;color:#232A31"
	if got := strings.Count(html, want); got != 2 {
		t.Errorf("expected each of the 2 list items to declare %q inline, got %d occurrences in: %s", want, got, html)
	}
}

// TestRenderMarkdown_CodeBlockDeclaresOnSurfaceTextColor references AC-2:
// the fenced code block declares the on-surface text color (#232A31)
// inline, matching the mockup.
func TestRenderMarkdown_CodeBlockDeclaresOnSurfaceTextColor(t *testing.T) {
	html := renderMarkdown("```\ncode line\n```")
	if !strings.Contains(html, "color:#232A31") {
		t.Errorf("expected on-surface text color declared on code block in: %s", html)
	}
}

// TestRenderMarkdown_ClosingFenceLineWithTrailingTextIsPreservedAsContent
// references AC-3: inside an open fenced block, a line that starts with the
// fence marker but carries trailing text does not close the block; its text
// is preserved as block content, not dropped.
func TestRenderMarkdown_ClosingFenceLineWithTrailingTextIsPreservedAsContent(t *testing.T) {
	html := renderMarkdown("```\n``` still open\nreal close below\n```")
	if !strings.Contains(html, "``` still open") {
		t.Errorf("expected fence-marker line with trailing text preserved as content in: %s", html)
	}
	if !strings.Contains(html, "real close below") {
		t.Errorf("expected content after the fence-like line preserved in: %s", html)
	}
	if got := strings.Count(html, "<pre"); got != 1 {
		t.Errorf("expected exactly one code block (the fence-like line must not close it), got %d in: %s", got, html)
	}
}

// TestRenderMarkdown_OpeningFenceInfoStringIsDiscarded references AC-3: an
// opening fence's info string (e.g. a language hint) is discarded, never
// rendered as block content.
func TestRenderMarkdown_OpeningFenceInfoStringIsDiscarded(t *testing.T) {
	html := renderMarkdown("```go\ncode line\n```")
	if strings.Contains(html, "go") {
		t.Errorf("expected opening fence info string discarded, got: %s", html)
	}
	if !strings.Contains(html, "code line") {
		t.Errorf("expected code block content preserved in: %s", html)
	}
}

// TestRenderMarkdown_FenceMarkerLineWithTrailingTextOutsideBlockOpensBlock
// references AC-3: outside any open block, a fence-marker line with
// trailing text (an info string) opens a code block, discarding the info
// string, exactly like a bare fence marker would.
func TestRenderMarkdown_FenceMarkerLineWithTrailingTextOutsideBlockOpensBlock(t *testing.T) {
	html := renderMarkdown("``` trailing info\ncode\n```")
	if !strings.Contains(html, "<pre") {
		t.Fatalf("expected a code block to open in: %s", html)
	}
	if strings.Contains(html, "trailing info") {
		t.Errorf("expected the opening fence's info string discarded, got: %s", html)
	}
	if !strings.Contains(html, "code") {
		t.Errorf("expected code block content preserved in: %s", html)
	}
}

// TestRenderMarkdown_FenceRoundTripLosesNoContent references AC-3: across
// an opening fence with an info string, a fence-marker line with trailing
// text inside the block, and a real closing fence, every piece of non-fence
// content survives (FR1 "never dropped").
func TestRenderMarkdown_FenceRoundTripLosesNoContent(t *testing.T) {
	html := renderMarkdown("```go\n``` inner marker line\nreal content\n```")
	for _, want := range []string{"inner marker line", "real content"} {
		if !strings.Contains(html, want) {
			t.Errorf("expected content %q preserved with zero loss, got: %s", want, html)
		}
	}
}

// TestRenderMarkdown_LinkWithBalancedParenthesesInURL references AC-4: a
// link destination containing balanced parentheses renders as a link with
// the full URL, rather than truncating at the first ')'.
func TestRenderMarkdown_LinkWithBalancedParenthesesInURL(t *testing.T) {
	html := renderMarkdown("see [docs](https://example.com/a(b))")
	if !strings.Contains(html, `href="https://example.com/a(b)"`) {
		t.Errorf("expected full URL with balanced parentheses in href, got: %s", html)
	}
	if !strings.Contains(html, "docs</a>") {
		t.Errorf("expected link text preserved in: %s", html)
	}
}

// TestRenderMarkdown_UnbalancedLinkDestinationFallsBackToEscapedPlainText
// references AC-4: when a link destination's parentheses cannot be
// balanced (no terminating ')'), the whole construct renders as escaped
// plain text instead of a broken/truncated link.
func TestRenderMarkdown_UnbalancedLinkDestinationFallsBackToEscapedPlainText(t *testing.T) {
	html := renderMarkdown("see [docs](https://example.com/a(b")
	if strings.Contains(html, "<a ") || strings.Contains(html, "href=") {
		t.Errorf("expected no <a> element for an unbalanced destination, got: %s", html)
	}
	if !strings.Contains(html, "https://example.com/a(b") {
		t.Errorf("expected the construct preserved as escaped plain text, got: %s", html)
	}
}
