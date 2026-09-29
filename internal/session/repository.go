package session

import (
	"context"
	"fmt"

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
	const query = "INSERT INTO sessions (user_id, token_hash, expires_at) VALUES($1, $2, $3) RETURNING id, user_id, token_hash, expires_at, created_at"
	row := repo.db.QueryRow(ctx, query, session.UserID, session.TokenHash, session.ExpiresAt)

	err := row.Scan(&createdSession.ID, &createdSession.UserID, &createdSession.TokenHash, &createdSession.ExpiresAt, &createdSession.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return createdSession, nil

}
