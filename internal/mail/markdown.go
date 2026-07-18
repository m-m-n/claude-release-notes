package mail

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

// renderMarkdown converts one release body (plain text possibly containing
// the supported Markdown subset) into a styled, self-contained HTML
// fragment using only inline style="" attributes (SPEC FR1/FR2). It never
// returns an error: any line it cannot classify degrades to an escaped
// plain paragraph so no content is ever lost.
//
// Supported constructs: ATX headings (# / ## / ### — all rendered as the
// single sub-heading style), unordered lists (- / *), ordered lists
// (N. ), inline code (`), bold (**), links ([text](url), http/https
// schemes only) and fenced code blocks (```). Everything else accumulates
// into plain paragraphs.
func renderMarkdown(body string) string {
	lines := strings.Split(body, "\n")

	var out strings.Builder
	first := true

	var paragraph []string
	var list *markdownList

	flushParagraph := func() {
		if len(paragraph) == 0 {
			return
		}
		out.WriteString(renderParagraph(paragraph))
		paragraph = nil
		first = false
	}
	flushList := func() {
		if list == nil {
			return
		}
		out.WriteString(list.render())
		list = nil
		first = false
	}

	i := 0
	for i < len(lines) {
		line := lines[i]

		if isFenceLine(line) {
			flushParagraph()
			flushList()
			var code []string
			i++
			for i < len(lines) && !isFenceLine(lines[i]) {
				code = append(code, lines[i])
				i++
			}
			// A missing closing fence (i == len(lines)) is not an error:
			// the block simply runs to the end of the body (SPEC FR1/AC-4).
			out.WriteString(renderCodeBlock(code))
			first = false
			i++ // skip the closing fence line, if any
			continue
		}

		if text, ok := parseHeading(line); ok {
			flushParagraph()
			flushList()
			out.WriteString(renderHeading(text, first))
			first = false
			i++
			continue
		}

		if text, ordered, ok := parseListItem(line); ok {
			flushParagraph()
			if list == nil || list.ordered != ordered {
				flushList()
				list = &markdownList{ordered: ordered}
			}
			list.items = append(list.items, text)
			i++
			continue
		}

		if strings.TrimSpace(line) == "" {
			flushParagraph()
			flushList()
			i++
			continue
		}

		flushList()
		paragraph = append(paragraph, line)
		i++
	}

	flushParagraph()
	flushList()

	return out.String()
}

// isFenceLine reports whether line opens or closes a fenced code block.
// Trailing content after the fence marker (a language hint, e.g. "```go")
// is accepted and ignored.
func isFenceLine(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "```")
}

var headingRe = regexp.MustCompile(`^(#{1,3})\s+(.*)$`)

// parseHeading recognizes 1-3 leading '#' followed by whitespace (SPEC
// FR1). 4+ hashes, or hashes with no following whitespace, are not
// classified as a heading and fall through to plain-paragraph handling.
func parseHeading(line string) (text string, ok bool) {
	m := headingRe.FindStringSubmatch(line)
	if m == nil {
		return "", false
	}
	return m[2], true
}

var (
	unorderedListRe = regexp.MustCompile(`^[-*]\s+(.*)$`)
	orderedListRe   = regexp.MustCompile(`^\d+\.\s+(.*)$`)
)

// parseListItem recognizes one unordered ("- "/"* ") or ordered ("N. ")
// list-item line.
func parseListItem(line string) (text string, ordered bool, ok bool) {
	if m := unorderedListRe.FindStringSubmatch(line); m != nil {
		return m[1], false, true
	}
	if m := orderedListRe.FindStringSubmatch(line); m != nil {
		return m[1], true, true
	}
	return "", false, false
}

// markdownList accumulates one run of consecutive same-kind list items.
type markdownList struct {
	ordered bool
	items   []string
}

func (l *markdownList) render() string {
	tag := "ul"
	if l.ordered {
		tag = "ol"
	}
	var items strings.Builder
	for _, item := range l.items {
		items.WriteString(fmt.Sprintf(`<li style="margin:0 0 8px 0;">%s</li>`, renderInline(item)))
	}
	return fmt.Sprintf(`<%s style="padding-left:24px;margin:8px 0;">%s</%s>`, tag, items.String(), tag)
}

// renderHeading renders heading text at the mail's single sub-heading
// style. first controls the top margin: the very first rendered element in
// the body carries no top margin (design decision).
func renderHeading(text string, first bool) string {
	margin := "24px 0 8px 0"
	if first {
		margin = "0 0 8px 0"
	}
	return fmt.Sprintf(
		`<div style="font-size:16px;line-height:1.5;font-weight:700;color:#232A31;margin:%s;">%s</div>`,
		margin, renderInline(text),
	)
}

// renderParagraph renders one run of consecutive plain lines as a single
// paragraph, joining the original lines with <br> so no content is lost
// (SPEC FR1: "consecutive plain lines form paragraphs with line breaks").
func renderParagraph(lines []string) string {
	rendered := make([]string, len(lines))
	for i, l := range lines {
		rendered[i] = renderInline(l)
	}
	return fmt.Sprintf(
		`<p style="margin:8px 0;font-size:16px;line-height:1.8;color:#232A31;">%s</p>`,
		strings.Join(rendered, "<br>"),
	)
}

// renderCodeBlock renders a fenced block's lines verbatim (escaped, never
// inline-parsed) inside a <pre> element (AC-4).
func renderCodeBlock(lines []string) string {
	escaped := make([]string, len(lines))
	for i, l := range lines {
		escaped[i] = html.EscapeString(l)
	}
	return fmt.Sprintf(
		`<pre style="background-color:#EDF1F5;padding:16px;font-size:13px;line-height:1.6;font-family:ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;border-radius:4px;margin:8px 0;">%s</pre>`,
		strings.Join(escaped, "\n"),
	)
}

// renderInline applies the inline pass (code spans, then bold, then links,
// in that precedence order) to one line of heading/list-item/paragraph
// text. Every text run is HTML-escaped before any markup is added around it
// (escape-before-markup, NFR2); code-span content is escaped but never
// further inline-parsed.
func renderInline(text string) string {
	var out strings.Builder
	var plain strings.Builder

	flushPlain := func() {
		if plain.Len() == 0 {
			return
		}
		out.WriteString(html.EscapeString(plain.String()))
		plain.Reset()
	}

	i := 0
	for i < len(text) {
		if text[i] == '`' {
			if j := strings.IndexByte(text[i+1:], '`'); j >= 0 {
				flushPlain()
				out.WriteString(renderCodeSpan(text[i+1 : i+1+j]))
				i += 1 + j + 1
				continue
			}
		}

		if strings.HasPrefix(text[i:], "**") {
			if j := strings.Index(text[i+2:], "**"); j >= 0 {
				flushPlain()
				out.WriteString(renderBold(text[i+2 : i+2+j]))
				i += 2 + j + 2
				continue
			}
		}

		if text[i] == '[' {
			if rendered, consumed, ok := parseLink(text[i:]); ok {
				flushPlain()
				out.WriteString(rendered)
				i += consumed
				continue
			}
		}

		r, size := utf8.DecodeRuneInString(text[i:])
		plain.WriteRune(r)
		i += size
	}
	flushPlain()

	return out.String()
}

func renderCodeSpan(content string) string {
	return fmt.Sprintf(
		`<code style="font-family:ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;font-size:0.875em;background-color:#EDF1F5;padding:1px 4px;border-radius:3px;">%s</code>`,
		html.EscapeString(content),
	)
}

func renderBold(content string) string {
	return fmt.Sprintf(`<strong style="font-weight:700;">%s</strong>`, html.EscapeString(content))
}

// parseLink recognizes a "[text](href)" construct starting at s[0]=='['.
// consumed is the number of bytes of s the construct occupies. When href
// does not parse as an absolute http/https URL (NFR2), the construct is not
// treated as a link at all: it is rendered as escaped plain text instead of
// emitting an <a> element.
func parseLink(s string) (rendered string, consumed int, ok bool) {
	closeBracket := strings.IndexByte(s, ']')
	if closeBracket < 0 || closeBracket+1 >= len(s) || s[closeBracket+1] != '(' {
		return "", 0, false
	}
	closeParenRel := strings.IndexByte(s[closeBracket+2:], ')')
	if closeParenRel < 0 {
		return "", 0, false
	}
	closeParen := closeBracket + 2 + closeParenRel

	linkText := s[1:closeBracket]
	href := s[closeBracket+2 : closeParen]
	consumed = closeParen + 1

	if !isHTTPURL(href) {
		return html.EscapeString(s[:consumed]), consumed, true
	}

	rendered = fmt.Sprintf(
		`<a href="%s" style="color:#33658A;text-decoration:underline;">%s</a>`,
		html.EscapeString(href),
		html.EscapeString(linkText),
	)
	return rendered, consumed, true
}

// isHTTPURL reports whether href parses as an absolute URL with scheme
// http or https (NFR2). Any other scheme (javascript:, mailto:, relative
// paths, ...) is rejected.
func isHTTPURL(href string) bool {
	u, err := url.ParseRequestURI(href)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}
