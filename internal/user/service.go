package user

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repository *UserRepository
}

func NewUserService(ur *UserRepository) *UserService {
	return &UserService{
		repository: ur,
	}
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
