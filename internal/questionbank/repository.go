package questionbank

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type QuestionBankRepository struct {
	db *pgxpool.Pool
}

func NewQuestionBankRepository(db *pgxpool.Pool) *QuestionBankRepository {
	return &QuestionBankRepository{
		db: db,
	}
}

func (repo *QuestionBankRepository) FindByID(ctx context.Context, bankID string, userID string) (*QuestionBank, error) {
	foundQuestionBank := &QuestionBank{}

	const query = "SELECT id, user_id, title, description, created_at, updated_at FROM question_banks WHERE id = $1 AND user_id = $2;"
	row := repo.db.QueryRow(ctx, query, bankID, userID)

	err := row.Scan(&foundQuestionBank.ID, &foundQuestionBank.UserID, &foundQuestionBank.Title, &foundQuestionBank.Description, &foundQuestionBank.CreatedAt, &foundQuestionBank.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrQuestionBankNotFound
		}
		return nil, fmt.Errorf("scan question bank: %w", err)
	}

	return foundQuestionBank, nil

}

func (repo *QuestionBankRepository) ListByUserID(ctx context.Context, userID string) ([]QuestionBank, error) {
	banks := make([]QuestionBank, 0)
	const query = "SELECT id, user_id, title, description, created_at, updated_at FROM question_banks WHERE user_id = $1 ORDER BY created_at DESC;"
	row, err := repo.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list by user id: %w", err)
	}
	defer row.Close()
	for row.Next() {
		var bank QuestionBank

		if err := row.Scan(
			&bank.ID,
			&bank.UserID,
			&bank.Title,
			&bank.Description,
			&bank.CreatedAt,
			&bank.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan question bank: %w", err)
		}

		banks = append(banks, bank)
	}

	if err := row.Err(); err != nil {
		return nil, fmt.Errorf("list by user id: %w", err)
	}

	return banks, nil

}

func (repo *QuestionBankRepository) Create(ctx context.Context, bank *QuestionBank) (*QuestionBank, error) {
	createdBank := &QuestionBank{}

	const query = "INSERT INTO question_banks (user_id, title, description) VALUES($1, $2, $3) RETURNING id, user_id, title, description, created_at, updated_at"

	row := repo.db.QueryRow(ctx, query, bank.UserID, bank.Title, bank.Description)

	err := row.Scan(&createdBank.ID, &createdBank.UserID, &createdBank.Title, &createdBank.Description, &createdBank.CreatedAt, &createdBank.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("create question bank: %w", err)
	}

	return createdBank, nil
}

func (repo *QuestionBankRepository) Update(ctx context.Context, title, description *string, bankID string, userID string) (*QuestionBank, error) {
	updatedBank := &QuestionBank{}
	const query = "UPDATE question_banks SET title = COALESCE($1, title), description = COALESCE($2, description), updated_at = NOW() WHERE id = $3 AND user_id = $4 RETURNING id, user_id, title, description, created_at, updated_at"
	row := repo.db.QueryRow(ctx, query, title, description, bankID, userID)

	err := row.Scan(&updatedBank.ID, &updatedBank.UserID, &updatedBank.Title, &updatedBank.Description, &updatedBank.CreatedAt, &updatedBank.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrQuestionBankNotFound
		}
		return nil, fmt.Errorf("update question bank: %w", err)
	}

	return updatedBank, nil
}

func (repo *QuestionBankRepository) Delete(ctx context.Context, bankID string, userID string) error {
	const query = "DELETE FROM question_banks WHERE id = $1 AND user_id = $2"
	result, err := repo.db.Exec(ctx, query, bankID, userID)

	if err != nil {
		return fmt.Errorf("delete question bank: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrQuestionBankNotFound
	}

	return nil

}
