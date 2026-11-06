package user

import "time"

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResult struct {
	User      *User
	Token     string `json:"-"`
	ExpiresAt time.Time
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID              string     `json:"id"`
	Email           string     `json:"email"`
	Name            string     `json:"name"`
	CreatedAt       time.Time  `json:"created_at"`
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
}

type RegisterResult struct {
	User                  *User
	VerificationEmailSent bool
}

type RegisterResponse struct {
	UserResponse
	VerificationRequired  bool `json:"verification_required"`
	VerificationEmailSent bool `json:"verification_email_sent"`
}

type VerifyEmailRequest struct {
	Token string `json:"token"`
}

type ResendVerificationRequest struct {
	Email string `json:"email"`
}

type MessageResponse struct {
	Message string `json:"message"`
}
