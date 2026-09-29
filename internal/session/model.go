package session

import "time"

type Session struct {
	ID        string
	UserID    string
	TokenHash []byte
	ExpiresAt time.Time
	CreatedAt time.Time
}
