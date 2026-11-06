package user

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Uses an isolated schema; never truncates or drops an application's tables.
// Run against a test database with TEST_DATABASE_URL configured.
func verificationDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; PostgreSQL integration test skipped")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "verification_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		_, err := admin.Exec(context.Background(), "DROP SCHEMA "+quoted+" CASCADE")
		if err != nil {
			t.Errorf("cleanup test schema: %v", err)
		}
		admin.Close()
	})
	for _, name := range []string{
		"20260923111601_add_users_table.sql",
		"20261104180000_add_email_verification.sql",
	} {
		body, err := os.ReadFile(filepath.Join("..", "..", "migrations", name))
		if err != nil {
			t.Fatal(err)
		}
		up := strings.Split(strings.Split(string(body), "-- +goose Up")[1], "-- +goose Down")[0]
		if _, err := pool.Exec(ctx, up); err != nil {
			t.Fatalf("apply %s: %v", name, err)
		}
	}
	return pool
}

func TestVerificationRepositoryAtomicRegistration(t *testing.T) {
	pool := verificationDatabase(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()
	_, err := repo.CreateWithVerification(ctx, &User{Email: "rollback@example.test", Name: "Rollback", PasswordHash: "hash"}, &EmailVerificationToken{TokenHash: []byte("too short")})
	if err == nil {
		t.Fatal("invalid token insert should fail")
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE email = 'rollback@example.test'").Scan(&count); err != nil || count != 0 {
		t.Fatal("registration was not rolled back atomically", count, err)
	}
}

func TestVerificationRepositoryConcurrencyAndResend(t *testing.T) {
	pool := verificationDatabase(t)
	ctx := context.Background()
	repo := NewUserRepository(pool)
	oldRaw, token, err := newVerificationToken()
	if err != nil {
		t.Fatal(err)
	}
	registered, err := repo.CreateWithVerification(ctx, &User{Email: "student@example.test", Name: "Student", PasswordHash: "hash"}, token)
	if err != nil {
		t.Fatal(err)
	}
	if registered.EmailVerifiedAt != nil || token.ExpiresAt.Sub(token.CreatedAt) < 23*time.Hour {
		t.Fatal("invalid initial verification state")
	}
	_, next, _ := newVerificationToken()
	if _, err := repo.IssueVerification(ctx, registered.Email, next); !errors.Is(err, ErrVerificationThrottled) {
		t.Fatal("resend cooldown bypass", err)
	}
	if _, err := pool.Exec(ctx, "UPDATE email_verification_tokens SET created_at = clock_timestamp() - INTERVAL '2 minutes' WHERE user_id = $1", registered.ID); err != nil {
		t.Fatal(err)
	}
	newRaw, next, _ := newVerificationToken()
	if _, err := repo.IssueVerification(ctx, registered.Email, next); err != nil {
		t.Fatal(err)
	}
	oldHash, _ := verificationTokenHash(oldRaw)
	if err := repo.ConfirmEmail(ctx, oldHash); !errors.Is(err, ErrInvalidVerificationToken) {
		t.Fatal("old token accepted", err)
	}
	newHash, _ := verificationTokenHash(newRaw)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Go(func() { results <- repo.ConfirmEmail(ctx, newHash) })
	}
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		} else if !errors.Is(err, ErrInvalidVerificationToken) {
			t.Fatal(err)
		}
	}
	if successes != 1 {
		t.Fatalf("confirmed %d times", successes)
	}
	if _, err := repo.IssueVerification(ctx, registered.Email, next); !errors.Is(err, ErrVerificationNotPending) {
		t.Fatal("resent for verified account", err)
	}
}

func TestVerificationExpiryCheckedAfterLock(t *testing.T) {
	pool := verificationDatabase(t)
	ctx := context.Background()
	repo := NewUserRepository(pool)
	_, token, _ := newVerificationToken()
	u, err := repo.CreateWithVerification(ctx, &User{Email: "expiry@example.test", Name: "Expiry", PasswordHash: "hash"}, token)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Rollback(ctx)
	if _, err := lock.Exec(ctx, "SELECT id FROM users WHERE id = $1 FOR UPDATE", u.ID); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() { result <- repo.ConfirmEmail(ctx, token.TokenHash) }()
	// Expire while confirmation is waiting for the user row lock.
	if _, err := lock.Exec(ctx, "UPDATE email_verification_tokens SET expires_at = clock_timestamp() WHERE user_id = $1", u.ID); err != nil {
		t.Fatal(err)
	}
	if err := lock.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if !errors.Is(err, ErrInvalidVerificationToken) {
			t.Fatal("expired token accepted after lock", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("confirmation blocked")
	}
}
