package question

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"qtp/internal/questionbank"
)

type QuestionRepository struct {
	db *pgxpool.Pool
}

func NewQuestionRepository(db *pgxpool.Pool) *QuestionRepository {
	return &QuestionRepository{
		db: db,
	}
}

func (repo *QuestionRepository) ListByBankID(ctx context.Context, userID string, bankID string) ([]Question, error) {
	var bankExists bool

	const checkBankQuery = `
		SELECT EXISTS (
			SELECT 1
			FROM question_banks
			WHERE id = $1 AND user_id = $2
		);
	`

	err := repo.db.QueryRow(
		ctx,
		checkBankQuery,
		bankID,
		userID,
	).Scan(&bankExists)

	if err != nil {
		return nil, fmt.Errorf(
			"check question bank existence: %w",
			err,
		)
	}

	if !bankExists {
		return nil, questionbank.ErrQuestionBankNotFound
	}

	const query = `
		SELECT
			q.id,
			q.question_bank_id,
			q.text,
			q.created_at,
			q.updated_at,
			ao.id,
			ao.question_id,
			ao.text,
			ao.position,
			ao.is_correct,
			ao.created_at,
			ao.updated_at
		FROM questions q
		JOIN question_banks qb
			ON qb.id = q.question_bank_id
		JOIN answer_options ao
			ON ao.question_id = q.id
		WHERE q.question_bank_id = $1
			AND qb.user_id = $2
		ORDER BY
			q.created_at,
			q.id,
			ao.position;
	`

	rows, err := repo.db.Query(
		ctx,
		query,
		bankID,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"query questions by bank id: %w",
			err,
		)
	}
	defer rows.Close()

	questions := make([]Question, 0)

	for rows.Next() {
		currentQuestion := Question{}
		currentOption := AnswerOption{}

		err := rows.Scan(
			&currentQuestion.ID,
			&currentQuestion.QuestionBankID,
			&currentQuestion.Text,
			&currentQuestion.CreatedAt,
			&currentQuestion.UpdatedAt,
			&currentOption.ID,
			&currentOption.QuestionID,
			&currentOption.Text,
			&currentOption.Position,
			&currentOption.IsCorrect,
			&currentOption.CreatedAt,
			&currentOption.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"scan question with option: %w",
				err,
			)
		}

		isFirstQuestion := len(questions) == 0

		if isFirstQuestion ||
			questions[len(questions)-1].ID != currentQuestion.ID {

			currentQuestion.Options = make(
				[]AnswerOption,
				0,
			)

			questions = append(
				questions,
				currentQuestion,
			)
		}

		lastQuestionIndex := len(questions) - 1

		questions[lastQuestionIndex].Options = append(
			questions[lastQuestionIndex].Options,
			currentOption,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate questions by bank id: %w",
			err,
		)
	}

	return questions, nil
}

func (repo *QuestionRepository) FindByID(ctx context.Context, userID, bankID, questionID string) (*Question, error) {
	foundQuestion := Question{
		Options: make([]AnswerOption, 0),
	}

	const query = `SELECT
  q.id,
  q.question_bank_id,
  q.text,
  q.created_at,
  q.updated_at,
  ao.id,
  ao.question_id,
  ao.text,
  ao.position,
  ao.is_correct,
  ao.created_at,
  ao.updated_at
	FROM questions q
	JOIN question_banks qb
  ON qb.id = q.question_bank_id
	JOIN answer_options ao
  ON ao.question_id = q.id
	WHERE q.id = $1
  AND q.question_bank_id = $2
  AND qb.user_id = $3
	ORDER BY ao.position;`

	row, err := repo.db.Query(ctx, query, questionID, bankID, userID)
	if err != nil {
		return nil, fmt.Errorf("find question: %w", err)
	}

	defer row.Close()
	found := false

	for row.Next() {
		answerOption := AnswerOption{}
		err := row.Scan(
			&foundQuestion.ID,
			&foundQuestion.QuestionBankID,
			&foundQuestion.Text,
			&foundQuestion.CreatedAt,
			&foundQuestion.UpdatedAt,

			&answerOption.ID,
			&answerOption.QuestionID,
			&answerOption.Text,
			&answerOption.Position,
			&answerOption.IsCorrect,
			&answerOption.CreatedAt,
			&answerOption.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("scan question with option: %w", err)
		}

		foundQuestion.Options = append(
			foundQuestion.Options,
			answerOption,
		)
		found = true
	}

	if !found {
		return nil, ErrQuestionNotFound
	}

	if err := row.Err(); err != nil {
		return nil, fmt.Errorf("iterate question rows: %w", err)
	}
	return &foundQuestion, nil
}

func (repo *QuestionRepository) Create(ctx context.Context, userID string, question *Question) (*Question, error) {
	tx, err := repo.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"begin create question transaction: %w",
			err,
		)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	createdQuestion := &Question{
		Options: make(
			[]AnswerOption,
			0,
			len(question.Options),
		),
	}

	const createQuestionQuery = `
		INSERT INTO questions (question_bank_id, text)
		SELECT id, $2
		FROM question_banks
		WHERE id = $1 AND user_id = $3
		RETURNING
			id,
			question_bank_id,
			text,
			created_at,
			updated_at;
	`

	err = tx.QueryRow(
		ctx,
		createQuestionQuery,
		question.QuestionBankID,
		question.Text,
		userID,
	).Scan(
		&createdQuestion.ID,
		&createdQuestion.QuestionBankID,
		&createdQuestion.Text,
		&createdQuestion.CreatedAt,
		&createdQuestion.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, questionbank.ErrQuestionBankNotFound
		}

		return nil, fmt.Errorf("create question: %w", err)
	}

	const createOptionQuery = `
		INSERT INTO answer_options (
			question_id,
			text,
			position,
			is_correct
		)
		VALUES ($1, $2, $3, $4)
		RETURNING
			id,
			question_id,
			text,
			position,
			is_correct,
			created_at,
			updated_at;
	`

	for _, option := range question.Options {
		createdOption := AnswerOption{}

		err := tx.QueryRow(
			ctx,
			createOptionQuery,
			createdQuestion.ID,
			option.Text,
			option.Position,
			option.IsCorrect,
		).Scan(
			&createdOption.ID,
			&createdOption.QuestionID,
			&createdOption.Text,
			&createdOption.Position,
			&createdOption.IsCorrect,
			&createdOption.CreatedAt,
			&createdOption.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"create answer option at position %d: %w",
				option.Position,
				err,
			)
		}

		createdQuestion.Options = append(
			createdQuestion.Options,
			createdOption,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf(
			"commit create question transaction: %w",
			err,
		)
	}

	return createdQuestion, nil
}
