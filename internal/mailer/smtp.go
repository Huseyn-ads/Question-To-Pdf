package mailer

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"time"

	"qtp/internal/config"
)

type SMTP struct{ cfg config.MailConfig }

func NewSMTP(cfg config.MailConfig) *SMTP {
	return &SMTP{cfg: cfg}
}

// SendVerification returns only operation names, never SMTP server replies:
// a server reply can contain recipients or message data.
func (s *SMTP) SendVerification(ctx context.Context, to, link string, expires time.Time) error {
	address, err := mail.ParseAddress(to)
	if err != nil || address.Name != "" || address.Address != to {
		return fmt.Errorf("invalid verification recipient")
	}
	message, err := buildMessage(s.cfg.From, to, link, expires)
	if err != nil {
		return fmt.Errorf("build verification message failed")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(s.cfg.Host, fmt.Sprint(s.cfg.Port)))
	if err != nil {
		return fmt.Errorf("SMTP connection failed")
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	deadline, _ := ctx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		return fmt.Errorf("SMTP deadline failed")
	}
	tlsConfig := &tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12}
	smtpConn := conn
	if s.cfg.TLSMode == "tls" {
		secure := tls.Client(conn, tlsConfig)
		if err := secure.HandshakeContext(ctx); err != nil {
			return fmt.Errorf("SMTP TLS handshake failed")
		}
		smtpConn = secure
	}
	client, err := smtp.NewClient(smtpConn, s.cfg.Host)
	if err != nil {
		return fmt.Errorf("SMTP greeting failed")
	}
	defer client.Close()
	switch s.cfg.TLSMode {
	case "starttls":
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("SMTP STARTTLS failed")
		}
	case "tls":
	case "none":
		if s.cfg.Username != "" || s.cfg.Password != "" {
			return fmt.Errorf("SMTP authentication requires TLS")
		}
	default:
		return fmt.Errorf("invalid SMTP TLS mode")
	}
	if s.cfg.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)); err != nil {
			return fmt.Errorf("SMTP authentication failed")
		}
	}
	if err := client.Mail(s.cfg.From); err != nil {
		return fmt.Errorf("SMTP sender rejected")
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP recipient rejected")
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA failed")
	}
	if _, err := writer.Write(message); err != nil {
		return fmt.Errorf("SMTP message write failed")
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("SMTP message acceptance failed")
	}
	// DATA acceptance is delivery to the relay. QUIT errors must not trigger
	// a duplicate send after the relay already accepted this message.
	_ = client.Quit()
	return nil
}

var verificationTemplate = template.Must(template.New("email").Parse(`
<!doctype html><html lang="ru"><body>
<h1>Подтвердите email</h1>
<p>Вы зарегистрировались в Question-To-Quiz.</p>
<p><a href="{{.Link}}">Подтвердить email</a></p>
<p>Ссылка действует до {{.Expires}} (UTC) и используется один раз.</p>
<p>Если вы не регистрировались, проигнорируйте это письмо.</p>
</body></html>`))

func buildMessage(from, to, link string, expires time.Time) ([]byte, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	plain := "Подтвердите email для Question-To-Quiz:\r\n" + link +
		"\r\nСсылка действует до " + expires.UTC().Format(time.RFC3339) +
		". Если вы не регистрировались, проигнорируйте письмо.\r\n"
	var html bytes.Buffer
	data := struct{ Link, Expires string }{link, expires.UTC().Format(time.RFC3339)}
	if err := verificationTemplate.Execute(&html, data); err != nil {
		return nil, err
	}
	for _, part := range []struct{ kind, text string }{
		{"text/plain; charset=utf-8", plain},
		{"text/html; charset=utf-8", html.String()},
	} {
		header := make(textproto.MIMEHeader)
		header.Set("Content-Type", part.kind)
		header.Set("Content-Transfer-Encoding", "quoted-printable")
		p, err := writer.CreatePart(header)
		if err != nil {
			return nil, err
		}
		encoded := quotedprintable.NewWriter(p)
		if _, err := io.WriteString(encoded, part.text); err != nil {
			return nil, err
		}
		if err := encoded.Close(); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return nil, err
	}
	headers := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMessage-ID: <%s@question-to-quiz.local>\r\nMIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=%q\r\n\r\n",
		from, to, mime.QEncoding.Encode("utf-8", "Подтвердите email — Question-To-Quiz"),
		time.Now().UTC().Format(time.RFC1123Z), hex.EncodeToString(id[:]), writer.Boundary())
	return append([]byte(headers), body.Bytes()...), nil
}
