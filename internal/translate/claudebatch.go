// Placeholder: compile-only stub for internal/translate, added by task0006
// so internal/app and cmd/claude-release-notes can compile and be wired in
// this worktree. Real implementation is owned by task0004 and replaces this
// file on merge (parent-side adoption protocol).
package translate

// DefaultCommand is the claude-batch command name used when the caller does
// not override it.
const DefaultCommand = "claude-batch"

// Translator translates release bodies to Japanese via claude-batch.
type Translator struct {
	command string
}

// NewTranslator constructs a Translator invoking command (default
// DefaultCommand when command is empty).
func NewTranslator(command string) *Translator {
	if command == "" {
		command = DefaultCommand
	}
	return &Translator{command: command}
}

// Translate takes one string per release body (order preserved) and returns
// a same-length, same-order slice of translated texts, or an error when the
// subprocess fails, output is empty, or any section cannot be recovered.
// Placeholder: always returns a nil slice and a nil error.
func (t *Translator) Translate(sections []string) ([]string, error) {
	return nil, nil
}
