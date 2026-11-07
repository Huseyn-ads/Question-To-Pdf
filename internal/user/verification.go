package user

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/url"
	"time"

	"qtp/internal/locale"
)

const VerificationLifetime = 24 * time.Hour

type EmailVerificationToken struct {
	ID        string
	UserID    string
	TokenHash []byte
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

type VerificationMailer interface {
	SendVerification(context.Context, string, string, time.Time) error
}

func newVerificationToken() (string, *EmailVerificationToken, error) {
	var random [32]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", nil, err
	}
	raw := base64.RawURLEncoding.EncodeToString(random[:])
	hash := sha256.Sum256([]byte(raw))
	return raw, &EmailVerificationToken{TokenHash: hash[:]}, nil
}

func verificationTokenHash(raw string) ([]byte, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil || len(raw) != 43 || len(decoded) != 32 ||
		base64.RawURLEncoding.EncodeToString(decoded) != raw {
		return nil, ErrInvalidVerificationToken
	}
	hash := sha256.Sum256([]byte(raw))
	return hash[:], nil
}

func (s *UserService) sendVerification(ctx context.Context, u *User, raw string, expires time.Time) bool {
	// A fragment is never sent in HTTP request URLs or Referer headers.
	// The language only picks the page copy; it stays in the fragment too.
	link := s.publicOrigin + "/v1/auth/verify-email#token=" + url.QueryEscape(raw) +
		"&lang=" + locale.FromContext(ctx)
	if err := s.mailer.SendVerification(ctx, u.Email, link, expires); err != nil {
		// SMTP servers can echo addresses or message data. Do not log their response.
		slog.Error("verification email delivery failed", "reason", safeMailFailureReason(err))
		return false
	}
	return true
}

// The SMTP sender returns fixed operation names. Only known messages may be
// logged; errors from other sender implementations can contain credentials,
// recipients or the verification link.
func safeMailFailureReason(err error) string {
	switch reason := err.Error(); reason {
	case "invalid verification recipient",
		"build verification message failed",
		"SMTP connection failed",
		"SMTP deadline failed",
		"SMTP TLS handshake failed",
		"SMTP greeting failed",
		"SMTP STARTTLS failed",
		"SMTP authentication requires TLS",
		"invalid SMTP TLS mode",
		"SMTP authentication failed",
		"SMTP sender rejected",
		"SMTP recipient rejected",
		"SMTP DATA failed",
		"SMTP message write failed",
		"SMTP message acceptance failed":
		return reason
	default:
		return "unclassified mail delivery error; details suppressed"
	}
}

func (s *UserService) VerifyEmail(ctx context.Context, raw string) error {
	hash, err := verificationTokenHash(raw)
	if err != nil {
		return err
	}
	return s.repository.ConfirmEmail(ctx, hash)
}

func (s *UserService) ResendVerification(ctx context.Context, email string) error {
	email = normalizeEmail(email)
	if !validEmail(email) {
		return ErrInvalidEmail
	}
	raw, token, err := newVerificationToken()
	if err != nil {
		return err
	}
	u, err := s.repository.IssueVerification(ctx, email, token)
	if err != nil {
		if errors.Is(err, ErrVerificationNotPending) || errors.Is(err, ErrVerificationThrottled) {
			return nil
		}
		return err
	}
	_ = s.sendVerification(ctx, u, raw, token.ExpiresAt)
	return nil
}
