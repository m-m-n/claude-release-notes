# Design: release-notes-fetcher

## Decisions

- Email layout: single centered column, max-width 800px, on a `canvas`
  background with one `surface` card containing header / releases / footer.
  Mockup: design/mockups/screen-email-digest.html
- Header: a `primary` band with the mail title (`heading-mail`, `on-primary`)
  and a caption line "anthropics/claude-code · N 件の新しいリリース".
- Each release section (newest first, FR5): version tag as `heading-release`
  in `primary`, publish date as `caption` in `on-surface-muted`, translated
  body as `body` on `on-surface`. Sections separated by a 1px `divider` rule.
- Empty release body renders "（リリースノート記載なし）" in
  `on-surface-muted` (SPEC edge case), never dropped.
- Footer: caption line "claude-release-notes により自動送信 · 原文: GitHub
  Releases" in `on-surface-muted` above nothing else — no unsubscribe or
  marketing chrome (single-recipient personal tool).
- Subject line format: `Claude Code リリースノート: v1.4.0 ほか N 件`
  (single release: `Claude Code リリースノート: v1.4.0`).
- Tokens: created `design-system/tokens.yaml` (no prior design system);
  cool-neutral palette with a single steel-blue accent.
- Email-client constraint: CSS custom properties and `<style>` blocks are
  unreliable in Gmail — the implementation must inline literal token VALUES
  as `style=""` attributes (table-or-div single column, no external
  resources, no web fonts; `system-ui, sans-serif` stack). The mockup's CSS
  variables are a design spec notation only.

## Rationale

- 800px single column (widened from 600px per user direction after seeing
  the delivered mail): the digest targets a single Gmail recipient, so the
  classic 600px mail-safe convention is not binding; `max-width` still lets
  narrow/mobile clients reflow to a single readable column.
- Body 16px / line-height 1.8: the payload is long-form translated Japanese;
  reading comfort outranks density (REQUIREMENTS 6.1).
- One accent color only (steel blue; cool base per user direction after
  seeing the delivered mail) keeps the digest calm and makes version
  headings the only landmarks — the email is scanned by version, then read
  linearly.
- Light palette fixed (no dark variant): dark-mode rendering of HTML mail is
  client-controlled and unreliable; a light card with 4.5:1+ text contrast
  degrades most gracefully.
- Autonomous judgment calls (revisit via /em-workflow:design): accent hue
  within the cool palette, footer wording, subject line format.

## Open items

- None. Visual refinements after seeing a real delivered mail go through
  /em-workflow:design.
