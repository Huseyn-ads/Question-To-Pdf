package user

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"qtp/internal/session"

	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repository        userStore
	sessionRepository sessionStore
	mailer            VerificationMailer
	publicOrigin      string
}

type userStore interface {
	CreateWithVerification(context.Context, *User, *EmailVerificationToken) (*User, error)
	IssueVerification(context.Context, string, *EmailVerificationToken) (*User, error)
	ConfirmEmail(context.Context, []byte) error
	FindByEmail(context.Context, string) (*User, error)
	FindByID(context.Context, string) (*User, error)
	FindByGoogleSub(context.Context, string) (*User, error)
	CreateGoogleUser(ctx context.Context, sub, email, name string) (*User, error)
}

type sessionStore interface {
	Create(context.Context, *session.Session) (*session.Session, error)
	FindActiveByTokenHash(context.Context, []byte) (*session.Session, error)
	DeleteByTokenHash(context.Context, []byte) error
}

var dummyPasswordHash = func() []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte("dummy password for missing accounts"), bcrypt.DefaultCost)
	if err != nil {
		panic("cannot initialize password verification")
	}
	return hash
}()

func NewUserService(ur userStore, sr sessionStore, mailer VerificationMailer, publicOrigin string) *UserService {
	return &UserService{
		repository:        ur,
		sessionRepository: sr,
		mailer:            mailer,
		publicOrigin:      strings.TrimRight(publicOrigin, "/"),
	}
}

func (s *UserService) Authenticate(ctx context.Context, request LoginRequest) (*User, error) {
	request.Email = strings.ToLower(strings.TrimSpace(request.Email))

	if !validEmail(request.Email) || request.Password == "" || len(request.Password) > 72 || !utf8.ValidString(request.Password) {
		return nil, ErrInvalidCredentials
	}

	foundUser, err := s.repository.FindByEmail(ctx, request.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(request.Password))
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	if foundUser.PasswordHash == "" {
		_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(request.Password))
		return nil, ErrInvalidCredentials
	}

	err = bcrypt.CompareHashAndPassword([]byte(foundUser.PasswordHash), []byte(request.Password))

	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return nil, ErrInvalidCredentials
		}

		return nil, fmt.Errorf("compare password hash: %w", err)
	}

	return foundUser, nil

}

func (s *UserService) Login(ctx context.Context, request LoginRequest) (*LoginResult, error) {
	newUser, err := s.Authenticate(ctx, request)

	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return nil, ErrInvalidCredentials
		}

		return nil, fmt.Errorf("authenticate user: %w", err)
	}

	return s.createSession(ctx, newUser)
}

func (s *UserService) loginWithGoogle(ctx context.Context, sub string, profile googleClaims) (*LoginResult, error) {
	currentUser, err := s.repository.FindByGoogleSub(ctx, sub)
	if errors.Is(err, ErrUserNotFound) {
		email := normalizeEmail(profile.Email)
		if !profile.EmailVerified || !validEmail(email) || !strings.HasSuffix(email, "@gmail.com") {
			return nil, ErrGoogleEmailRequired
		}
		name := strings.TrimSpace(profile.Name)
		if name == "" || utf8.RuneCountInString(name) > 255 {
			name = email
		}
		currentUser, err = s.repository.CreateGoogleUser(ctx, sub, email, name)
	}
	if err != nil {
		return nil, err
	}
	return s.createSession(ctx, currentUser)
}

func (s *UserService) createSession(ctx context.Context, currentUser *User) (*LoginResult, error) {
	if currentUser.EmailVerifiedAt == nil {
		return nil, ErrEmailNotVerified
	}

	token, hashedToken, err := session.GenerateToken()

	if err != nil {
		return nil, fmt.Errorf("token generate: %w", err)
	}

	csrfToken, _, err := session.GenerateToken()
	if err != nil {
		return nil, fmt.Errorf("generate CSRF token: %w", err)
	}

	newSession := session.Session{
		UserID:    currentUser.ID,
		TokenHash: hashedToken,
		ExpiresAt: time.Now().UTC().Add(session.Duration),
		CSRFToken: csrfToken,
	}

	createdSession, err := s.sessionRepository.Create(ctx, &newSession)

	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &LoginResult{
		User:      currentUser,
		Token:     token,
		ExpiresAt: createdSession.ExpiresAt,
	}, nil
}

func (s *UserService) Register(ctx context.Context, request RegisterRequest) (*RegisterResult, error) {
	request.Email = strings.ToLower(strings.TrimSpace(request.Email))
	request.Name = strings.TrimSpace(request.Name)

	if len(request.Name) == 0 {
		return nil, ErrNameRequired
	}

	if len(request.Email) == 0 {
		return nil, ErrEmailRequired
	}

	if !validEmail(request.Email) {
		return nil, ErrInvalidEmail
	}

	if len(request.Password) == 0 {
		return nil, ErrPasswordRequired
	}

	if !utf8.ValidString(request.Password) {
		return nil, ErrPasswordTooShort
	}
	symbolCount := utf8.RuneCountInString(request.Password)
	if symbolCount < 8 {
		return nil, ErrPasswordTooShort
	}

	if len(request.Password) > 72 {
		return nil, ErrPasswordTooLong
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)

	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	newUser := &User{
		Email:        request.Email,
		Name:         request.Name,
		PasswordHash: string(hashedPassword),
	}

	raw, verification, err := newVerificationToken()
	if err != nil {
		return nil, fmt.Errorf("generate email verification token: %w", err)
	}
	createdUser, err := s.repository.CreateWithVerification(ctx, newUser, verification)

	if err != nil {
		return nil, fmt.Errorf("register user: %w", err)
	}

	// User and token are committed before SMTP. A delivery failure does not
	// delete the account; the client can explicitly request another email.
	sent := s.sendVerification(ctx, createdUser, raw, verification.ExpiresAt)
	return &RegisterResult{User: createdUser, VerificationEmailSent: sent}, nil
}

func (s *UserService) CurrentUser(ctx context.Context, token string) (*User, error) {
	currentUser, _, err := s.CurrentSession(ctx, token)
	return currentUser, err
}

func (s *UserService) CurrentSession(ctx context.Context, token string) (*User, *session.Session, error) {
	if token == "" {
		return nil, nil, ErrUnauthorized
	}

	newToken := session.HashToken(token)

	foundSession, err := s.sessionRepository.FindActiveByTokenHash(ctx, newToken)

	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			return nil, nil, ErrUnauthorized
		}
		return nil, nil, fmt.Errorf("current user: %w", err)
	}
	if !foundSession.ExpiresAt.After(time.Now()) {
		return nil, nil, ErrUnauthorized
	}

	user, err := s.repository.FindByID(ctx, foundSession.UserID)

	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, nil, ErrUnauthorized
		}
		return nil, nil, fmt.Errorf("find current user: %w", err)
	}

	if user.EmailVerifiedAt == nil {
		return nil, nil, ErrUnauthorized
	}
	return user, foundSession, nil

}

func validEmail(email string) bool {
	if len(email) == 0 || len(email) > 254 {
		return false
	}
	parsed, err := mail.ParseAddress(email)
	return err == nil && parsed.Name == "" && parsed.Address == email && strings.Contains(email, "@")
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (s *UserService) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}

	tokenHash := session.HashToken(token)

	err := s.sessionRepository.DeleteByTokenHash(ctx, tokenHash)

	if err != nil {
		return fmt.Errorf("logout user: %w", err)
	}

	return nil
}
