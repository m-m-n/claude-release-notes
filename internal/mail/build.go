// Placeholder: compile-only stub for internal/mail's build role, added by
// task0006 so internal/app and cmd/claude-release-notes can compile and be
// wired in this worktree. Real implementation is owned by task0005 and
// replaces this file on merge (parent-side adoption protocol).
package mail

import "time"

// Item is this package's own minimal value type for one digest entry (no
// shared Release type per IMPLEMENTATION.md's cross-task decision).
type Item struct {
	Version     string
	PublishedAt time.Time
	Body        string
}

// Build composes the digest from items (ordered newest first) and returns
// the mail subject and the self-contained inline-styled HTML body.
// Placeholder: always returns empty strings.
func Build(items []Item) (subject string, htmlBody string) {
	return "", ""
}
