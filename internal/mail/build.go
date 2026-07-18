// Package mail composes the Japanese release-notes digest as a
// self-contained HTML email and sends it through Gmail SMTP.
package mail

import (
	"fmt"
	"html/template"
	"strings"
	"time"
)

// jst is the explicit, fixed timezone used to render calendar dates in the
// digest. Using a fixed zone (rather than the host's configured system
// timezone) keeps the rendered day unambiguous and deterministic regardless
// of where the binary runs.
var jst = time.FixedZone("JST", 9*60*60)

// emptyBodyPlaceholder is rendered in place of a release's translated body
// when that body is empty (or whitespace-only).
const emptyBodyPlaceholder = "（リリースノート記載なし）"

// Item is one release entry in the digest. Callers order items newest
// first; Build preserves that order. Body is plain text (translated
// release-notes prose); Build HTML-escapes it.
type Item struct {
	Version     string
	PublishedAt time.Time
	Body        string
}

// Build composes the digest subject and self-contained HTML body from items
// (already ordered newest first, per the caller's responsibility).
func Build(items []Item) (subject, htmlBody string) {
	return buildSubject(items), buildHTML(items)
}

func buildSubject(items []Item) string {
	if len(items) == 0 {
		return "Claude Code リリースノート"
	}
	newest := items[0].Version
	if len(items) == 1 {
		return fmt.Sprintf("Claude Code リリースノート: %s", newest)
	}
	return fmt.Sprintf("Claude Code リリースノート: %s ほか %d 件", newest, len(items)-1)
}

// itemView is the per-release data shape fed to digestTemplate. BodyHTML is
// pre-escaped and marked safe so the template does not double-escape it.
type itemView struct {
	Version     string
	PublishedAt string
	BodyHTML    template.HTML
	Divider     bool
}

type digestData struct {
	ItemCount int
	Items     []itemView
}

func buildHTML(items []Item) string {
	views := make([]itemView, len(items))
	for i, it := range items {
		views[i] = itemView{
			Version:     it.Version,
			PublishedAt: formatPublishedDate(it.PublishedAt),
			BodyHTML:    bodyHTML(it.Body),
			Divider:     i > 0,
		}
	}

	var buf strings.Builder
	if err := digestTemplate.Execute(&buf, digestData{ItemCount: len(items), Items: views}); err != nil {
		// digestTemplate is a fixed, trusted string executed against a
		// fixed data shape; Execute can only fail on a template/programming
		// defect here, never on caller-supplied item content.
		panic(fmt.Sprintf("mail: digest template: %v", err))
	}
	return buf.String()
}

// formatPublishedDate renders t's calendar day in JST, e.g. "2026-07-16 公開".
func formatPublishedDate(t time.Time) string {
	return t.In(jst).Format("2006-01-02") + " 公開"
}

// bodyHTML renders one release's translated body as HTML by running it
// through the Markdown renderer (renderMarkdown). An empty (or
// whitespace-only) body renders the placeholder text instead of an empty
// section.
func bodyHTML(body string) template.HTML {
	if strings.TrimSpace(body) == "" {
		return template.HTML(fmt.Sprintf(
			`<p style="margin:0;font-size:16px;line-height:1.8;color:#5C6B7A;">%s</p>`,
			template.HTMLEscapeString(emptyBodyPlaceholder),
		))
	}

	return template.HTML(renderMarkdown(body))
}

// digestTemplate renders the self-contained digest HTML: every style is an
// inline style="" attribute using literal design-token values (mail clients
// do not support CSS custom properties or <style> blocks), system-ui/
// sans-serif font stack, no external resources.
var digestTemplate = template.Must(template.New("digest").Parse(`<!doctype html>
<html>
<body style="margin:0;padding:0;background-color:#EDF1F5;font-family:system-ui, sans-serif;">
<div style="max-width:600px;margin:0 auto;padding:16px;">
<div style="background-color:#FFFFFF;border:1px solid #D9E1E8;">
<div style="background-color:#33658A;color:#FFFFFF;padding:24px;">
<div style="font-size:20px;line-height:1.4;font-weight:700;">Claude Code リリースノート</div>
<div style="font-size:13px;line-height:1.5;margin-top:8px;">anthropics/claude-code &middot; {{.ItemCount}} 件の新しいリリース</div>
</div>
{{range .Items}}{{if .Divider}}<div style="height:1px;line-height:1px;font-size:0;background-color:#D9E1E8;">&nbsp;</div>
{{end}}<div style="padding:24px;">
<div style="font-size:18px;line-height:1.4;font-weight:700;color:#33658A;">{{.Version}}</div>
<div style="font-size:13px;line-height:1.5;color:#5C6B7A;margin-top:8px;">{{.PublishedAt}}</div>
<div style="margin-top:16px;">{{.BodyHTML}}</div>
</div>
{{end}}<div style="height:1px;line-height:1px;font-size:0;background-color:#D9E1E8;">&nbsp;</div>
<div style="padding:24px;">
<div style="font-size:13px;line-height:1.5;color:#5C6B7A;">claude-release-notes により自動送信 &middot; 原文: GitHub Releases</div>
</div>
</div>
</div>
</body>
</html>
`))
