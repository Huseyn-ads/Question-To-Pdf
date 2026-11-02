package session

import "time"

const Duration = 7 * 24 * time.Hour

type Session struct {
	ID        string
	UserID    string
	TokenHash []byte `json:"-"`
	CSRFToken string `json:"-"`
	ExpiresAt time.Time
	CreatedAt time.Time
}
