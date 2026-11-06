package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (repo *UserRepository) CreateWithVerification(ctx context.Context, user *User, token *EmailVerificationToken) (*User, error) {
	tx, err := repo.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin registration: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	createdUser := &User{}
	const query = "INSERT INTO users (email, name, password_hash) VALUES ($1, $2, $3) RETURNING id, email, name, password_hash, created_at, updated_at, email_verified_at;"
	row := tx.QueryRow(ctx, query, user.Email, user.Name, user.PasswordHash)

	err = row.Scan(&createdUser.ID, &createdUser.Email, &createdUser.Name, &createdUser.PasswordHash, &createdUser.CreatedAt, &createdUser.UpdatedAt, &createdUser.EmailVerifiedAt)

	if err != nil {
		var pgError *pgconn.PgError

		if errors.As(err, &pgError) && pgError.Code == "23505" {
			return nil, ErrEmailAlreadyExists
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	if err := insertVerification(ctx, tx, createdUser.ID, token); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit registration: %w", err)
	}
	return createdUser, nil

}

func (repo *UserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	const query = `
		SELECT id, email, name, COALESCE(password_hash, ''),
		       created_at, updated_at, email_verified_at
		FROM users
		WHERE email = $1;
	`

	var found User
	err := repo.db.QueryRow(ctx, query, email).Scan(
		&found.ID, &found.Email, &found.Name, &found.PasswordHash,
		&found.CreatedAt, &found.UpdatedAt, &found.EmailVerifiedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	return &found, nil
}

func (repo *UserRepository) FindByID(ctx context.Context, id string) (*User, error) {
	const query = `
		SELECT id, email, name, COALESCE(password_hash, ''),
		       created_at, updated_at, email_verified_at
		FROM users
		WHERE id = $1;
	`

	var found User
	err := repo.db.QueryRow(ctx, query, id).Scan(
		&found.ID, &found.Email, &found.Name, &found.PasswordHash,
		&found.CreatedAt, &found.UpdatedAt, &found.EmailVerifiedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by ID: %w", err)
	}

	return &found, nil
}

func (repo *UserRepository) FindByGoogleSub(ctx context.Context, sub string) (*User, error) {
	const query = `
		SELECT id, email, name, created_at, updated_at, email_verified_at
		FROM users
		WHERE google_sub = $1;
	`

	var found User
	err := repo.db.QueryRow(ctx, query, sub).Scan(
		&found.ID, &found.Email, &found.Name,
		&found.CreatedAt, &found.UpdatedAt, &found.EmailVerifiedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by Google subject: %w", err)
	}

	return &found, nil
}

func (repo *UserRepository) CreateGoogleUser(ctx context.Context, sub, email, name string) (*User, error) {
	const query = `
		INSERT INTO users (google_sub, email, name, email_verified_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT DO NOTHING
		RETURNING id, email, name, created_at, updated_at, email_verified_at;
	`

	var created User
	err := repo.db.QueryRow(ctx, query, sub, email, name).Scan(
		&created.ID, &created.Email, &created.Name,
		&created.CreatedAt, &created.UpdatedAt, &created.EmailVerifiedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		// A concurrent callback may have inserted this subject already.
		found, findErr := repo.FindByGoogleSub(ctx, sub)
		if errors.Is(findErr, ErrUserNotFound) {
			return nil, ErrEmailAlreadyExists
		}
		return found, findErr
	}
	if err != nil {
		return nil, fmt.Errorf("create Google user: %w", err)
	}

	return &created, nil
}

func insertVerification(ctx context.Context, tx pgx.Tx, userID string, token *EmailVerificationToken) error {
	const query = `
		INSERT INTO email_verification_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, clock_timestamp() + ($3 * INTERVAL '1 second'))
		RETURNING id, user_id, expires_at, created_at;
	`
	if err := tx.QueryRow(ctx, query, userID, token.TokenHash, int64(VerificationLifetime/time.Second)).Scan(
		&token.ID, &token.UserID, &token.ExpiresAt, &token.CreatedAt,
	); err != nil {
		return fmt.Errorf("insert verification token: %w", err)
	}
	return nil
}

func (repo *UserRepository) IssueVerification(ctx context.Context, email string, token *EmailVerificationToken) (*User, error) {
	tx, err := repo.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin verification resend: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Both resend and confirmation lock the user first, then its tokens.
	u := &User{}
	err = tx.QueryRow(ctx, `
		SELECT id, email, email_verified_at FROM users WHERE email = $1 FOR UPDATE;
	`, email).Scan(&u.ID, &u.Email, &u.EmailVerifiedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrVerificationNotPending
	}
	if err != nil {
		return nil, fmt.Errorf("find user for verification: %w", err)
	}
	if u.EmailVerifiedAt != nil {
		return nil, ErrVerificationNotPending
	}

	// clock_timestamp() is evaluated after the lock, not at transaction start.
	var blocked bool
	err = tx.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE created_at > clock_timestamp() - INTERVAL '1 hour') >= 5
			OR COUNT(*) FILTER (WHERE created_at > clock_timestamp() - INTERVAL '60 seconds') > 0
		FROM email_verification_tokens WHERE user_id = $1;
	`, u.ID).Scan(&blocked)
	if err != nil {
		return nil, fmt.Errorf("check verification resend limit: %w", err)
	}
	if blocked {
		return nil, ErrVerificationThrottled
	}

	if _, err := tx.Exec(ctx, `
		UPDATE email_verification_tokens SET used_at = clock_timestamp()
		WHERE user_id = $1 AND used_at IS NULL;
	`, u.ID); err != nil {
		return nil, fmt.Errorf("invalidate old verification token: %w", err)
	}
	if err := insertVerification(ctx, tx, u.ID, token); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit verification resend: %w", err)
	}
	return u, nil
}

func (repo *UserRepository) ConfirmEmail(ctx context.Context, hash []byte) error {
	tx, err := repo.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin email confirmation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userID string
	err = tx.QueryRow(ctx, `
		SELECT user_id FROM email_verification_tokens WHERE token_hash = $1;
	`, hash).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidVerificationToken
	}
	if err != nil {
		return fmt.Errorf("find verification token: %w", err)
	}

	var verifiedAt *time.Time
	if err := tx.QueryRow(ctx, `
		SELECT email_verified_at FROM users WHERE id = $1 FOR UPDATE;
	`, userID).Scan(&verifiedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInvalidVerificationToken
		}
		return fmt.Errorf("lock user for verification: %w", err)
	}
	if verifiedAt != nil {
		return ErrInvalidVerificationToken
	}
	var tokenID string
	var expiresAt time.Time
	var usedAt *time.Time
	err = tx.QueryRow(ctx, `
		SELECT id, expires_at, used_at
		FROM email_verification_tokens WHERE token_hash = $1 FOR UPDATE;
	`, hash).Scan(&tokenID, &expiresAt, &usedAt)
	if err != nil {
		return fmt.Errorf("lock verification token: %w", err)
	}
	var now time.Time
	if err := tx.QueryRow(ctx, "SELECT clock_timestamp();").Scan(&now); err != nil {
		return fmt.Errorf("check verification time: %w", err)
	}
	if usedAt != nil || !now.Before(expiresAt) {
		return ErrInvalidVerificationToken
	}
	if _, err := tx.Exec(ctx, `
		UPDATE users SET email_verified_at = $2, updated_at = $2 WHERE id = $1;
	`, userID, now); err != nil {
		return fmt.Errorf("confirm email: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE email_verification_tokens SET used_at = $2 WHERE user_id = $1 AND used_at IS NULL;
	`, userID, now); err != nil {
		return fmt.Errorf("consume verification token: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit email confirmation: %w", err)
	}
	return nil
}
