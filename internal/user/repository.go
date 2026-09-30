package user

import (
	"context"
	"errors"
	"fmt"

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

func (repo *UserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	foundUser := &User{}
	const query = "SELECT id, email, name, password_hash, created_at, updated_at FROM users WHERE email = $1;"
	row := repo.db.QueryRow(ctx, query, email)

	err := row.Scan(&foundUser.ID, &foundUser.Email, &foundUser.Name, &foundUser.PasswordHash, &foundUser.CreatedAt, &foundUser.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	return foundUser, nil
}

func (repo *UserRepository) FindByID(ctx context.Context, id string) (*User, error) {
	foundUser := &User{}
	const query = "SELECT id, email, name, password_hash, created_at, updated_at FROM users WHERE id = $1;"
	row := repo.db.QueryRow(ctx, query, id)

	err := row.Scan(&foundUser.ID, &foundUser.Email, &foundUser.Name, &foundUser.PasswordHash, &foundUser.CreatedAt, &foundUser.UpdatedAt)

	if err != nil {

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}

		return nil, fmt.Errorf("find user by ID: %w", err)
	}

	return foundUser, nil

}
