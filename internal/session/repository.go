package session

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepository struct {
	db *pgxpool.Pool
}

func NewSessionRepository(db *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{
		db: db,
	}
}

func (repo *SessionRepository) Create(ctx context.Context, session *Session) (*Session, error) {
	createdSession := &Session{}
	const query = "INSERT INTO sessions (user_id, token_hash, expires_at, csrf_token) VALUES($1, $2, $3, $4) RETURNING id, user_id, token_hash, expires_at, created_at, csrf_token"
	row := repo.db.QueryRow(ctx, query, session.UserID, session.TokenHash, session.ExpiresAt, session.CSRFToken)

	err := row.Scan(&createdSession.ID, &createdSession.UserID, &createdSession.TokenHash, &createdSession.ExpiresAt, &createdSession.CreatedAt, &createdSession.CSRFToken)

	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return createdSession, nil

}

func (repo *SessionRepository) FindActiveByTokenHash(ctx context.Context, tokenHash []byte) (*Session, error) {
	foundSession := &Session{}
	const query = "SELECT id, user_id, token_hash, expires_at, created_at, csrf_token FROM sessions WHERE token_hash = $1 AND expires_at > NOW();"
	row := repo.db.QueryRow(ctx, query, tokenHash)

	err := row.Scan(&foundSession.ID, &foundSession.UserID, &foundSession.TokenHash, &foundSession.ExpiresAt, &foundSession.CreatedAt, &foundSession.CSRFToken)
	if err != nil {

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}

		return nil, fmt.Errorf("find session: %w", err)
	}

	return foundSession, nil
}

func (repo *SessionRepository) DeleteByTokenHash(ctx context.Context, tokenHash []byte) error {
	const query = "DELETE FROM sessions WHERE token_hash = $1"
	_, err := repo.db.Exec(ctx, query, tokenHash)

	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil

}
