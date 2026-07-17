// Placeholder: compile-only stub for internal/mail's send role, added by
// task0006 so internal/app and cmd/claude-release-notes can compile and be
// wired in this worktree. Real implementation is owned by task0005 and
// replaces this file on merge (parent-side adoption protocol).
package mail

// Sender delivers a built digest over Gmail SMTP.
type Sender struct {
	host        string
	port        int
	account     string
	appPassword string
}

// NewSender constructs a Sender. host/port are overridable for tests.
func NewSender(host string, port int, account, appPassword string) *Sender {
	return &Sender{host: host, port: port, account: account, appPassword: appPassword}
}

// Send delivers one MIME text/html; charset=UTF-8 message via STARTTLS SMTP
// AUTH. Placeholder: always returns a nil error.
func (s *Sender) Send(to, subject, htmlBody string) error {
	return nil
}
