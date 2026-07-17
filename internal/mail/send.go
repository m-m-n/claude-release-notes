package mail

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

// Gmail's SMTP submission endpoint; the Sender's default target.
const (
	defaultHost = "smtp.gmail.com"
	defaultPort = 587
)

// Sender delivers the digest email through SMTP with STARTTLS then AUTH.
type Sender struct {
	host        string
	port        int
	account     string
	appPassword string

	// tlsConfig overrides the STARTTLS configuration. Left nil in
	// production, where a config derived from host is used; tests in this
	// package set it to talk to a local fake SMTP server without a
	// publicly-trusted certificate.
	tlsConfig *tls.Config
}

// NewSender constructs a Sender for the given Gmail account and app
// password. An empty host or zero port falls back to Gmail's submission
// endpoint (smtp.gmail.com:587); tests override both to target a local
// server.
func NewSender(host string, port int, account, appPassword string) *Sender {
	if host == "" {
		host = defaultHost
	}
	if port == 0 {
		port = defaultPort
	}
	return &Sender{host: host, port: port, account: account, appPassword: appPassword}
}

// Send delivers one text/html message to "to" via STARTTLS SMTP AUTH.
func (s *Sender) Send(to, subject, htmlBody string) error {
	addr := net.JoinHostPort(s.host, strconv.Itoa(s.port))

	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return s.wrapErr("dial", err)
	}

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		conn.Close()
		return s.wrapErr("connect", err)
	}
	defer client.Close()

	tlsConfig := s.tlsConfig
	if tlsConfig == nil {
		tlsConfig = &tls.Config{ServerName: s.host}
	}
	if err := client.StartTLS(tlsConfig); err != nil {
		return s.wrapErr("starttls", err)
	}

	auth := smtp.PlainAuth("", s.account, s.appPassword, s.host)
	if err := client.Auth(auth); err != nil {
		return s.wrapErr("auth", err)
	}

	if err := client.Mail(s.account); err != nil {
		return s.wrapErr("mail from", err)
	}
	if err := client.Rcpt(to); err != nil {
		return s.wrapErr("rcpt to", err)
	}

	w, err := client.Data()
	if err != nil {
		return s.wrapErr("data", err)
	}

	msg, err := buildMessage(s.account, to, subject, htmlBody)
	if err != nil {
		return s.wrapErr("build message", err)
	}

	if _, err := w.Write(msg); err != nil {
		return s.wrapErr("write message", err)
	}
	if err := w.Close(); err != nil {
		return s.wrapErr("finalize message", err)
	}

	return s.wrapErr("quit", client.Quit())
}

// wrapErr adds a stage prefix and scrubs the app password from the
// underlying error text before it can reach a caller or a log line.
func (s *Sender) wrapErr(stage string, err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if s.appPassword != "" {
		msg = strings.ReplaceAll(msg, s.appPassword, "[REDACTED]")
	}
	return fmt.Errorf("mail: %s: %s", stage, msg)
}

// buildMessage assembles an RFC-compliant text/html message: MIME headers,
// a UTF-8 MIME-encoded subject, and a quoted-printable body safe for
// non-ASCII content.
func buildMessage(from, to, subject, htmlBody string) ([]byte, error) {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "From: %s\r\n", from)
	fmt.Fprintf(&buf, "To: %s\r\n", to)
	fmt.Fprintf(&buf, "Subject: %s\r\n", mime.QEncoding.Encode("UTF-8", subject))
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	fmt.Fprintf(&buf, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	buf.WriteString("\r\n")

	qw := quotedprintable.NewWriter(&buf)
	if _, err := qw.Write([]byte(htmlBody)); err != nil {
		return nil, err
	}
	if err := qw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
