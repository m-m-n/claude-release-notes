# Design: mail-render-refresh

## Decisions

- Base layout, header, footer, and per-release structure follow the parent
  feature's design (release-notes-fetcher DESIGN.md) with the CURRENT cool
  tokens. Mockup: design/mockups/screen-email-digest.html (states:
  rendered / plain / empty-body).
- Markdown sub-headings (any `#`–`###`) render as ONE style: `heading-body`
  token (16px/1.5, 700) in `on-surface` — NOT primary; version headings
  remain the mail's only primary-colored landmarks. Spacing: 24px above /
  8px below (first element: no top margin).
- Paragraphs: 8px vertical margins within the body.
- Lists (ul/ol): 24px left padding, 8px between items — unchanged from the
  parent design's list style.
- Inline code: monospace stack (`ui-monospace, SFMono-Regular, Menlo,
  Consolas, monospace`), 0.875em, `surface-muted` background, 1px 4px
  padding, 3px radius.
- Code blocks: `surface-muted` background, 16px padding, `code` token
  (13px/1.6 monospace), 4px radius, horizontal scroll when overflowing.
- Links: `primary` color, underlined.
- Bold: weight 700 only (no color change).
- Tokens extended (extend-don't-fork): `color.surface-muted` (#EDF1F5),
  `typography.heading-body`, `typography.code` in design-system/tokens.yaml.
- Email-client constraint carried over: implementation inlines literal token
  VALUES as `style=""` attributes; the mockup's CSS variables are notation
  only.

## Rationale

- Sub-headings in on-surface (not primary): release bodies can contain many
  headings; coloring them all primary would bury the version landmarks
  (parent rationale: the mail is scanned by version).
- One heading level: GitHub release notes rarely use meaningful heading
  hierarchy inside a single release; flattening avoids a typographic scale
  competing with version headings in a 600px column.
- `surface-muted` shares canvas's value but is a separate role: code
  backgrounds must be able to diverge from the page background in a future
  theme without re-tokenizing.
- Input evidence: design/input/delivered-mail-raw-markdown.png (delivered
  mail showing raw `##` / `-` / backticks — the defect this feature fixes).

## Open items

- None.
