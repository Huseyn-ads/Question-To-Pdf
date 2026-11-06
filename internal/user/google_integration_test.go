package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"qtp/internal/session"
)

const googleLoginMigration = "20261106120000_add_google_login.sql"

func TestGoogleLoginPostgres(t *testing.T) {
	pool := verificationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	repo := NewUserRepository(pool)
	mailer := &captureMailer{}
	service := NewUserService(repo, session.NewSessionRepository(pool), mailer, "https://quiz.test")
	passwordUser, err := service.Register(ctx, RegisterRequest{
		Email: "existing@gmail.com", Name: "Existing", Password: "test-password",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.VerifyEmail(ctx, emailToken(t, mailer.link)); err != nil {
		t.Fatal(err)
	}

	applyUserMigration(t, pool, "20260928120924_add_sessions_table.sql")
	applyUserMigration(t, pool, "20261102120000_add_session_csrf_token.sql")
	applyUserMigration(t, pool, googleLoginMigration)

	login, err := service.Login(ctx, LoginRequest{Email: passwordUser.User.Email, Password: "test-password"})
	if err != nil || login.User.ID != passwordUser.User.ID {
		t.Fatal("the existing password account could not log in after migration", err)
	}
	if _, err := service.Register(ctx, RegisterRequest{
		Email: "another@example.test", Name: "Another", Password: "test-password",
	}); err != nil {
		t.Fatal("registration without a Google subject failed", err)
	}
	if _, err := service.loginWithGoogle(ctx, "different-google-id", googleClaims{
		Email: passwordUser.User.Email, EmailVerified: true,
	}); !errors.Is(err, ErrEmailAlreadyExists) {
		t.Fatalf("Google login with an existing email: %v", err)
	}

	profile := googleClaims{Email: "student@gmail.com", Name: "Student", EmailVerified: true}
	const sub = "google-id"
	type outcome struct {
		login *LoginResult
		err   error
	}
	const concurrentLogins = 2
	results := make(chan outcome, concurrentLogins)
	start := make(chan struct{})
	for range concurrentLogins {
		go func() {
			<-start
			login, err := service.loginWithGoogle(ctx, sub, profile)
			results <- outcome{login, err}
		}()
	}
	close(start)
	var localID, lastToken string
	for range concurrentLogins {
		result := <-results
		if result.err != nil {
			t.Fatal(result.err)
		}
		u, currentSession, err := service.CurrentSession(ctx, result.login.Token)
		if err != nil || u.EmailVerifiedAt == nil || u.PasswordHash != "" || currentSession.CSRFToken == "" {
			t.Fatal("could not load Google session and user", err)
		}
		if localID != "" && (u.ID != localID || lastToken == result.login.Token) {
			t.Fatal("concurrent sign-ins should share a user and have distinct sessions")
		}
		localID, lastToken = u.ID, result.login.Token
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM users WHERE google_sub = $1", sub).Scan(&count); err != nil || count != 1 {
		t.Fatal("wrong number of users for the Google subject", count, err)
	}
	if _, err := service.Authenticate(ctx, LoginRequest{Email: profile.Email, Password: "test-password"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("password login for a Google-only account: %v", err)
	}
}

func TestGoogleMigrationAcceptsExistingColumn(t *testing.T) {
	pool := verificationDatabase(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		ALTER TABLE users
			ADD COLUMN google_sub VARCHAR(255) NOT NULL UNIQUE,
			ALTER COLUMN password_hash DROP NOT NULL;
		INSERT INTO users (email, name, google_sub, email_verified_at)
		VALUES ('earlier@gmail.com', 'Earlier', 'existing-google-id', NOW());
	`); err != nil {
		t.Fatal(err)
	}
	applyUserMigration(t, pool, googleLoginMigration)
	repo := NewUserRepository(pool)
	u, err := repo.FindByGoogleSub(ctx, "existing-google-id")
	if err != nil || u.Email != "earlier@gmail.com" {
		t.Fatal("migration should preserve the existing Google user", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO users (email, name, password_hash)
		VALUES ('password@example.test', 'Password user', 'hash')`); err != nil {
		t.Fatal("migration should permit users without a Google subject", err)
	}
}
