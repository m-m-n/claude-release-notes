// Placeholder: compile-only stub for internal/state, added by task0006 so
// internal/app and cmd/claude-release-notes can compile and be wired in this
// worktree. Real implementation is owned by task0002 and replaces this file
// on merge (parent-side adoption protocol).
package state

// Store persists the last delivered release version.
type Store struct {
	baseDir string
}

// NewStore constructs a Store rooted at baseDir. An empty baseDir means
// "resolve the XDG default internally".
func NewStore(baseDir string) *Store {
	return &Store{baseDir: baseDir}
}

// Load returns the stored version and found=true, or found=false when no
// state exists yet (never an error for mere absence). Placeholder: always
// reports "not found".
func (s *Store) Load() (version string, found bool, err error) {
	return "", false, nil
}

// Save persists version atomically (temp file + rename), creating
// directories as needed. Placeholder: no-op.
func (s *Store) Save(version string) error {
	return nil
}
