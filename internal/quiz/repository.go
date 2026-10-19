package quiz

import (
	"context"
	"errors"
	"fmt"
	"qtp/internal/questionbank"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type QuizRepository struct {
	db *pgxpool.Pool
}

func NewQuizRepository(db *pgxpool.Pool) *QuizRepository {
	return &QuizRepository{
		db: db,
	}
}

func (repo *QuizRepository) Create(ctx context.Context, userID string, bankID string, quiz Quiz) (*Quiz, error) {
	createdQuiz := Quiz{
		UserID:           userID,
		QuestionBankID:   bankID,
		QuestionCount:    quiz.QuestionCount,
		TimeLimitMinutes: quiz.TimeLimitMinutes,
		ShuffleOptions:   quiz.ShuffleOptions,
		Questions:        make([]QuizQuestion, 0, len(quiz.Questions)),
	}
	tx, err := repo.db.Begin(ctx)

	if err != nil {
		return nil, fmt.Errorf("begin to create quiz transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const checkQuery = `SELECT id FROM question_banks WHERE id = $1 AND user_id = $2 `

	var foundBankID string

	err = tx.QueryRow(
		ctx,
		checkQuery,
		bankID,
		userID,
	).Scan(&foundBankID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, questionbank.ErrQuestionBankNotFound
		}
		return nil, fmt.Errorf("check question bank ownership: %w", err)
	}

	const createQuery = `INSERT INTO quizzes (user_id, question_bank_id, question_count, time_limit_minutes, shuffle_options) VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at;`
	err = tx.QueryRow(ctx, createQuery, userID, bankID, quiz.QuestionCount, quiz.TimeLimitMinutes, quiz.ShuffleOptions).Scan(&createdQuiz.ID, &createdQuiz.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("create quiz: %w", err)
	}

	const questionCreateQuery = `INSERT INTO quiz_questions (quiz_id, text, position) VALUES ($1, $2, $3) RETURNING id, quiz_id, text, position;`

	for _, question := range quiz.Questions {
		createdQuestion := QuizQuestion{
			Options: make([]QuizAnswerOptions, len(question.Options)),
		}
		err = tx.QueryRow(ctx, questionCreateQuery, createdQuiz.ID, question.Text, question.Position).Scan(&createdQuestion.ID, &createdQuestion.QuizID, &createdQuestion.Text, &createdQuestion.Position)
		if err != nil {
			return nil, fmt.Errorf("create quiz question: %w", err)
		}
		const optionCreateQuery = `INSERT INTO quiz_answer_options (quiz_questions_id, text, position, is_correct) VALUES ($1, $2, $3, $4) RETURNING id, quiz_questions_id, text, position, is_correct`
		for _, option := range question.Options {
			createdOption := QuizAnswerOptions{}
			err = tx.QueryRow(ctx, optionCreateQuery, createdQuestion.ID, option.Text, option.Position, option.IsCorrect).Scan(&createdOption.ID, &createdOption.QuizQuestionID, &createdOption.Text, &createdOption.Position, &createdOption.IsCorrect)
			if err != nil {
				return nil, fmt.Errorf("create quiz option: %w", err)
			}
			createdQuestion.Options = append(createdQuestion.Options, createdOption)
		}
		createdQuiz.Questions = append(createdQuiz.Questions, createdQuestion)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("transaction commit error: %w", err)
	}

	return &createdQuiz, nil
}
