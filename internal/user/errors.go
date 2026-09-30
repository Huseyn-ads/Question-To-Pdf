package user

import "errors"

var (
	ErrNameRequired       = errors.New("name required")
	ErrEmailRequired      = errors.New("email required")
	ErrInvalidEmail       = errors.New("invalid email")
	ErrPasswordRequired   = errors.New("password required")
	ErrPasswordTooShort   = errors.New("password is too short")
	ErrPasswordTooLong    = errors.New("password is too long")
	ErrEmailAlreadyExists = errors.New("this email already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUnauthorized       = errors.New("unauthorized")
)
