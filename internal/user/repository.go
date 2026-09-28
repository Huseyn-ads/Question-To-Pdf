package user

import (
	"context"
	"errors"
	"fmt"

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

func (repo *UserRepository) Create(ctx context.Context, user *User) (*User, error) {
	createdUser := &User{}
	const query = "INSERT INTO users (email, name, password_hash) VALUES ($1, $2, $3) RETURNING id, email, name, password_hash, created_at, updated_at;"
	row := repo.db.QueryRow(ctx, query, user.Email, user.Name, user.PasswordHash)

	err := row.Scan(&createdUser.ID, &createdUser.Email, &createdUser.Name, &createdUser.PasswordHash, &createdUser.CreatedAt, &createdUser.UpdatedAt)

	if err != nil {
		var pgError *pgconn.PgError

		if errors.As(err, &pgError) && pgError.Code == "23505" {
			return nil, ErrEmailAlreadyExists
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	return createdUser, nil

}
