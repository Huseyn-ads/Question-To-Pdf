package mailer

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/mail"
	"strconv"
	"strings"
	"testing"
	"time"

	"qtp/internal/config"
	"qtp/internal/locale"
)

func smtpTestServer(t *testing.T) (config.MailConfig, <-chan string) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	messages := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
		_, _ = io.WriteString(conn, "220 local test SMTP\r\n")
		reader := bufio.NewReader(conn)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			switch {
			case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
				_, _ = io.WriteString(conn, "250-localhost\r\n250 8BITMIME\r\n")
			case strings.HasPrefix(line, "MAIL FROM:"), strings.HasPrefix(line, "RCPT TO:"):
				_, _ = io.WriteString(conn, "250 ok\r\n")
			case strings.TrimSpace(line) == "DATA":
				_, _ = io.WriteString(conn, "354 send message\r\n")
				var message strings.Builder
				for {
					data, err := reader.ReadString('\n')
					if err != nil {
						return
					}
					if data == ".\r\n" {
						break
					}
					message.WriteString(strings.TrimPrefix(data, "."))
				}
				messages <- message.String()
				_, _ = io.WriteString(conn, "250 accepted\r\n")
			case strings.TrimSpace(line) == "QUIT":
				_, _ = io.WriteString(conn, "221 bye\r\n")
				return
			default:
				_, _ = io.WriteString(conn, "500 unsupported\r\n")
			}
		}
	}()
	host, portString, _ := net.SplitHostPort(listener.Addr().String())
	port, _ := strconv.Atoi(portString)
	return config.MailConfig{Host: host, Port: uint16(port), From: "sender@example.test", TLSMode: "none"}, messages
}

func TestSMTPDeliversMIMEVerificationMessage(t *testing.T) {
	cfg, received := smtpTestServer(t)
	link := "https://quiz.test/v1/auth/verify-email#token=example-only"
	expires := time.Now().UTC().Add(24 * time.Hour)
	if err := NewSMTP(cfg).SendVerification(context.Background(), "student@example.test", link, expires); err != nil {
		t.Fatal(err)
	}
	var raw string
	select {
	case raw = <-received:
	case <-time.After(time.Second):
		t.Fatal("SMTP did not receive message")
	}
	message, err := mail.ReadMessage(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	sender, err := mail.ParseAddress(message.Header.Get("From"))
	if message.Header.Get("To") != "student@example.test" || err != nil || sender.Address != cfg.From || sender.Name != brandName {
		t.Fatal("wrong SMTP message addresses")
	}
	mediaType, params, err := mime.ParseMediaType(message.Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/alternative" {
		t.Fatal("not a MIME message", err)
	}
	reader := multipart.NewReader(message.Body, params["boundary"])
	parts := 0
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(part)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(body, []byte(link)) || !bytes.Contains(body, []byte("24 часа")) {
			t.Fatalf("verification link/expiry missing from MIME part %d", parts)
		}
		parts++
	}
	if parts != 2 {
		t.Fatalf("got %d MIME parts, want text and HTML", parts)
	}
}

func TestSMTPRequiresSTARTTLSInsteadOfDowngrading(t *testing.T) {
	cfg, received := smtpTestServer(t)
	cfg.TLSMode = "starttls"
	if err := NewSMTP(cfg).SendVerification(context.Background(), "student@example.test",
		"https://quiz.test/#token=example", time.Now()); err == nil {
		t.Fatal("SMTP silently downgraded to plaintext")
	}
	select {
	case <-received:
		t.Fatal("sent mail without required TLS")
	default:
	}
}

func TestSMTPHonorsCancellation(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = io.Copy(io.Discard, conn)
	}()
	host, portString, _ := net.SplitHostPort(listener.Addr().String())
	port, _ := strconv.Atoi(portString)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	started := time.Now()
	err = NewSMTP(config.MailConfig{Host: host, Port: uint16(port), From: "sender@example.test", TLSMode: "none"}).
		SendVerification(ctx, "student@example.test", "https://quiz.test/#token=example", time.Now())
	if err == nil || time.Since(started) > time.Second {
		t.Fatal("SMTP did not honor context", err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("SMTP connection remained open")
	}
}

func TestBuildMessageEscapesHTMLAndRejectsRecipientInjection(t *testing.T) {
	cfg := config.MailConfig{Host: "localhost", Port: 1025, From: "sender@example.test", TLSMode: "none"}
	err := NewSMTP(cfg).SendVerification(context.Background(), "user@example.test\r\nBcc:other@example.test", "", time.Now())
	if err == nil {
		t.Fatal("accepted injected recipient header")
	}
	raw, err := buildMessage(cfg.From, "user@example.test", "https://quiz.test/#token=a&x=<bad>", time.Now(), "ru")
	if err != nil {
		t.Fatal(err)
	}
	message, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	_, params, err := mime.ParseMediaType(message.Header.Get("Content-Type"))
	if err != nil {
		t.Fatal(err)
	}
	reader := multipart.NewReader(message.Body, params["boundary"])
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(part)
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(part.Header.Get("Content-Type"), "text/html") && bytes.Contains(body, []byte("<bad>")) {
			t.Fatal("HTML template failed to escape link")
		}
	}
}

func TestVerificationEmailFollowsRequestLanguage(t *testing.T) {
	now := time.Now()
	for lang, want := range map[string][]string{
		"az": {"E-poçtunuzu təsdiqləyin", "24 saat", `lang="az"`},
		"ru": {"Подтвердите email", "24 часа", `lang="ru"`},
		"xx": {"Подтвердите email"},
	} {
		email, err := renderVerification("https://quiz.test/v1/auth/verify-email#token=t&lang="+lang, now.Add(24*time.Hour), now, lang)
		if err != nil {
			t.Fatal(err)
		}
		for _, text := range want {
			if !strings.Contains(email.HTML, text) {
				t.Errorf("%s email is missing %q", lang, text)
			}
		}
	}
	if got := russianHours(1) + russianHours(3) + russianHours(11) + russianHours(21); got != "часчасачасовчас" {
		t.Errorf("russian hour forms = %q", got)
	}
}

func TestSMTPUsesLanguageFromContext(t *testing.T) {
	cfg, received := smtpTestServer(t)
	ctx := locale.WithLanguage(context.Background(), locale.Azerbaijani)
	if err := NewSMTP(cfg).SendVerification(ctx, "student@example.test", "https://quiz.test/#token=x", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	raw := <-received
	message, err := mail.ReadMessage(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	subject, err := new(mime.WordDecoder).DecodeHeader(message.Header.Get("Subject"))
	if err != nil || !strings.HasPrefix(subject, "E-poçtunuzu") || message.Header.Get("Content-Language") != "az" {
		t.Fatalf("Azerbaijani email expected, got subject %q", subject)
	}
}
