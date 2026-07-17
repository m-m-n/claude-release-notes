package mail

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"fmt"
	"io"
	"math/big"
	"mime/quotedprintable"
	"net"
	"net/textproto"
	"strings"
	"testing"
	"time"
)

// fakeSMTPResult captures what the fake server observed while handling one
// Send call.
type fakeSMTPResult struct {
	authOK       bool
	authAccount  string
	authPassword string
	mailFrom     string
	rcptTo       string
	rawMessage   string
}

// startFakeSMTPServer starts a local TCP listener that speaks just enough
// SMTP (EHLO / STARTTLS / EHLO / AUTH PLAIN / MAIL / RCPT / DATA / QUIT) to
// exercise Sender.Send's real STARTTLS+AUTH code path end to end, including
// a genuine TLS handshake. When acceptAuth is false, the server rejects
// authentication and echoes authFailureDetail back verbatim in the SMTP
// error text — this proves Send scrubs the app password from returned
// errors even when the wire-level error text contains it.
func startFakeSMTPServer(t *testing.T, acceptAuth bool, authFailureDetail string) (addr string, resultCh <-chan fakeSMTPResult, trustedCAs *x509.CertPool) {
	t.Helper()

	cert, pool := generateTestCert(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	ch := make(chan fakeSMTPResult, 1)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		var result fakeSMTPResult
		tc := textproto.NewConn(conn)

		tc.PrintfLine("220 localhost ESMTP fake")
		tc.ReadLine() // EHLO
		tc.PrintfLine("250-localhost Hello")
		tc.PrintfLine("250 STARTTLS")
		tc.ReadLine() // STARTTLS
		tc.PrintfLine("220 Go ahead")

		tlsConn := tls.Server(conn, &tls.Config{Certificates: []tls.Certificate{cert}})
		if err := tlsConn.Handshake(); err != nil {
			return
		}
		tc = textproto.NewConn(tlsConn)

		tc.ReadLine() // EHLO (post-TLS)
		tc.PrintfLine("250-localhost Hello")
		tc.PrintfLine("250 AUTH PLAIN")

		authLine, _ := tc.ReadLine()
		const prefix = "AUTH PLAIN "
		if strings.HasPrefix(authLine, prefix) {
			decoded, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(authLine, prefix))
			parts := strings.SplitN(string(decoded), "\x00", 3)
			if len(parts) == 3 {
				result.authAccount = parts[1]
				result.authPassword = parts[2]
			}
		}

		if !acceptAuth {
			tc.PrintfLine("535 authentication failed: %s", authFailureDetail)
			tc.ReadLine() // "*" (client aborts the AUTH exchange)
			tc.PrintfLine("501 aborted")
			tc.ReadLine() // QUIT
			tc.PrintfLine("221 Bye")
			ch <- result
			return
		}

		result.authOK = true
		tc.PrintfLine("235 Authentication successful")

		result.mailFrom, _ = tc.ReadLine()
		tc.PrintfLine("250 OK")

		result.rcptTo, _ = tc.ReadLine()
		tc.PrintfLine("250 OK")

		tc.ReadLine() // DATA
		tc.PrintfLine("354 Send message, end with <CRLF>.<CRLF>")

		msgBytes, _ := tc.ReadDotBytes()
		result.rawMessage = string(msgBytes)
		tc.PrintfLine("250 OK: queued")

		tc.ReadLine() // QUIT
		tc.PrintfLine("221 Bye")

		ch <- result
	}()

	return ln.Addr().String(), ch, pool
}

// generateTestCert creates an ephemeral, self-signed certificate for
// "127.0.0.1" plus a CertPool trusting it, so the test client can perform a
// genuine, fully-verified TLS handshake against the local fake server
// without disabling certificate verification.
func generateTestCert(t *testing.T) (tls.Certificate, *x509.CertPool) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	parsed, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}

	pool := x509.NewCertPool()
	pool.AddCert(parsed)

	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, pool
}

func splitHostPort(t *testing.T, addr string) (string, int) {
	t.Helper()
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split host port: %v", err)
	}
	var port int
	if _, err := fmt.Sscanf(portStr, "%d", &port); err != nil {
		t.Fatalf("parse port: %v", err)
	}
	return host, port
}

// decodeQuotedPrintableBody extracts and decodes the message body.
// rawMessage comes from textproto.Reader.ReadDotBytes, which normalizes
// line endings to a bare "\n", so the header/body separator is "\n\n" here
// (rather than the "\r\n\r\n" the message uses on the wire).
func decodeQuotedPrintableBody(t *testing.T, rawMessage string) string {
	t.Helper()
	idx := strings.Index(rawMessage, "\n\n")
	if idx == -1 {
		t.Fatalf("message has no header/body separator: %q", rawMessage)
	}
	body := rawMessage[idx+len("\n\n"):]
	decoded, err := io.ReadAll(quotedprintable.NewReader(strings.NewReader(body)))
	if err != nil {
		t.Fatalf("decode quoted-printable body: %v", err)
	}
	return string(decoded)
}

// TestSend_STARTTLSAndAuthAgainstMockServer references AC-6 (Send performs
// STARTTLS and AUTH against a mocked/injected SMTP endpoint) and AC-5 (the
// assembled message has correct MIME headers, a UTF-8-encoded subject, and
// a body that decodes back to the HTML).
func TestSend_STARTTLSAndAuthAgainstMockServer(t *testing.T) {
	addr, resultCh, trustedCAs := startFakeSMTPServer(t, true, "")
	host, port := splitHostPort(t, addr)

	sender := NewSender(host, port, "sender@example.com", "app-password-1234")
	sender.tlsConfig = &tls.Config{RootCAs: trustedCAs, ServerName: host}

	subject, htmlBody := Build([]Item{{Version: "v1.0.0", PublishedAt: time.Now(), Body: "本文です"}})

	if err := sender.Send("recipient@example.com", subject, htmlBody); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	result := <-resultCh

	if !result.authOK {
		t.Fatalf("server did not observe a successful AUTH")
	}
	if result.authAccount != "sender@example.com" {
		t.Errorf("authAccount = %q, want %q", result.authAccount, "sender@example.com")
	}
	if result.authPassword != "app-password-1234" {
		t.Errorf("authPassword = %q, want %q", result.authPassword, "app-password-1234")
	}
	if !strings.Contains(result.mailFrom, "sender@example.com") {
		t.Errorf("MAIL FROM missing sender: %q", result.mailFrom)
	}
	if !strings.Contains(result.rcptTo, "recipient@example.com") {
		t.Errorf("RCPT TO missing recipient: %q", result.rcptTo)
	}

	if !strings.Contains(result.rawMessage, "MIME-Version: 1.0") {
		t.Errorf("message missing MIME-Version header: %q", result.rawMessage)
	}
	if !strings.Contains(result.rawMessage, "Content-Type: text/html; charset=UTF-8") {
		t.Errorf("message missing Content-Type header: %q", result.rawMessage)
	}
	if !strings.Contains(result.rawMessage, "Content-Transfer-Encoding: quoted-printable") {
		t.Errorf("message missing Content-Transfer-Encoding header: %q", result.rawMessage)
	}
	if !strings.Contains(result.rawMessage, "Subject: =?UTF-8?") {
		t.Errorf("message subject is not MIME-encoded: %q", result.rawMessage)
	}

	decoded := decodeQuotedPrintableBody(t, result.rawMessage)
	if !strings.Contains(decoded, "v1.0.0") || !strings.Contains(decoded, "本文です") {
		t.Errorf("decoded body does not match the original HTML: %q", decoded)
	}
}

// TestSend_ErrorsNeverContainAppPassword references AC-6's sentinel
// assertion: even when the underlying SMTP error text itself echoes the
// attempted password, the error Send returns must not contain it.
func TestSend_ErrorsNeverContainAppPassword(t *testing.T) {
	const appPassword = "super-secret-app-password"

	addr, resultCh, trustedCAs := startFakeSMTPServer(t, false, "attempted password was "+appPassword)
	host, port := splitHostPort(t, addr)

	sender := NewSender(host, port, "sender@example.com", appPassword)
	sender.tlsConfig = &tls.Config{RootCAs: trustedCAs, ServerName: host}

	err := sender.Send("recipient@example.com", "subject", "<p>body</p>")
	if err == nil {
		t.Fatalf("expected an error from failed AUTH, got nil")
	}
	if strings.Contains(err.Error(), appPassword) {
		t.Fatalf("error leaked the app password: %v", err)
	}

	<-resultCh
}
