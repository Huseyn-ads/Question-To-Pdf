package user

import "time"

type User struct {
	ID              string
	Email           string
	Name            string
	PasswordHash    string `json:"-"`
	EmailVerifiedAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
