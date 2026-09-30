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
	repository        *UserRepository
	sessionRepository *session.SessionRepository
}

func NewUserService(ur *UserRepository, sr *session.SessionRepository) *UserService {
	return &UserService{
		repository:        ur,
		sessionRepository: sr,
	}
}

func (s *UserService) Authenticate(ctx context.Context, request LoginRequest) (*User, error) {
	request.Email = strings.ToLower(strings.TrimSpace(request.Email))

	if request.Email == "" || request.Password == "" {
		return nil, ErrInvalidCredentials
	}

	foundUser, err := s.repository.FindByEmail(ctx, request.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("find user by email: %w", err)
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

	token, hashedToken, err := session.GenerateToken()

	if err != nil {
		return nil, fmt.Errorf("token generate: %w", err)
	}

	const sessionDuration = 7 * 24 * time.Hour
	newSession := session.Session{
		UserID:    newUser.ID,
		TokenHash: hashedToken,
		ExpiresAt: time.Now().UTC().Add(sessionDuration),
	}

	createdSession, err := s.sessionRepository.Create(ctx, &newSession)

	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &LoginResult{
		User:      newUser,
		Token:     token,
		ExpiresAt: createdSession.ExpiresAt,
	}, nil
}

func (s *UserService) Register(ctx context.Context, request RegisterRequest) (*User, error) {
	request.Email = strings.ToLower(strings.TrimSpace(request.Email))
	request.Name = strings.TrimSpace(request.Name)

	if len(request.Name) == 0 {
		return nil, ErrNameRequired
	}

	if len(request.Email) == 0 {
		return nil, ErrEmailRequired
	}

	parsedEmail, err := mail.ParseAddress(request.Email)

	if err != nil {
		return nil, ErrInvalidEmail
	}

	if parsedEmail.Address != request.Email && parsedEmail.Name != "" {
		return nil, ErrInvalidEmail
	}

	if len(request.Password) == 0 {
		return nil, ErrPasswordRequired
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

	createdUser, err := s.repository.Create(ctx, newUser)

	if err != nil {
		return nil, fmt.Errorf("register user: %w", err)
	}

	return createdUser, nil
}

func (s *UserService) CurrentUser(ctx context.Context, token string) (*User, error) {
	if token == "" {
		return nil, ErrUnauthorized
	}

	newToken := session.HashToken(token)

	foundSession, err := s.sessionRepository.FindActiveByTokenHash(ctx, newToken)

	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, fmt.Errorf("current user: %w", err)
	}

	user, err := s.repository.FindByID(ctx, foundSession.UserID)

	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, fmt.Errorf("find current user: %w", err)
	}

	return user, nil

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
