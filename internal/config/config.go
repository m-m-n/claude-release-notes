// Placeholder: compile-only stub for internal/config, added by task0006 so
// internal/app and cmd/claude-release-notes can compile and be wired in this
// worktree. Real implementation is owned by task0001 and replaces this file
// on merge (parent-side adoption protocol).
package config

// Config holds validated application configuration. Field names and the
// zero-value-means-XDG-default convention for Load's baseDir are pinned by
// IMPLEMENTATION.md's Shared Components table.
type Config struct {
	GmailAccount     string
	GmailAppPassword string
	MailTo           string
	GitHubToken      string
}

// Load reads and validates the YAML config file rooted at baseDir. An empty
// baseDir means "resolve the XDG default internally". Placeholder: always
// returns a zero Config and a nil error.
func Load(baseDir string) (Config, error) {
	return Config{}, nil
}
