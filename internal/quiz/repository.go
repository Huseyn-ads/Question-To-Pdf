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

func (repo *QuizRepository) FindByOwnedID(ctx context.Context, userID string, quizID string) (*Quiz, error) {
	quiz := Quiz{
		Questions: make([]QuizQuestion, 0),
	}

	const findQuery = `
		SELECT
			q.id,
			q.question_bank_id,
			q.question_count,
			q.time_limit_minutes,
			q.shuffle_options,
			q.created_at,
			qq.id,
			qq.quiz_id,
			qq.text,
			qq.position,
			qao.id,
			qao.quiz_question_id,
			qao.text,
			qao.position,
			qao.is_correct
		FROM quizzes q
		LEFT JOIN quiz_questions qq
			ON qq.quiz_id = q.id
		LEFT JOIN quiz_answer_options qao
			ON qao.quiz_question_id = qq.id
		WHERE q.id = $1 AND q.user_id = $2
		ORDER BY qq.position, qao.position
	`

	rows, err := repo.db.Query(ctx, findQuery, quizID, userID)
	if err != nil {
		return nil, fmt.Errorf("find owned quiz: %w", err)
	}
	defer rows.Close()

	questionsMap := make(map[string]*QuizQuestion)
	questionOrder := make([]string, 0)
	found := false

	for rows.Next() {
		var (
			qqID       *string
			qqQuizID   *string
			qqText     *string
			qqPosition *int

			qaoID         *string
			qaoQuestionID *string
			qaoText       *string
			qaoPosition   *int
			qaoIsCorrect  *bool
		)

		if err := rows.Scan(
			&quiz.ID,
			&quiz.QuestionBankID,
			&quiz.QuestionCount,
			&quiz.TimeLimitMinutes,
			&quiz.ShuffleOptions,
			&quiz.CreatedAt,
			&qqID,
			&qqQuizID,
			&qqText,
			&qqPosition,
			&qaoID,
			&qaoQuestionID,
			&qaoText,
			&qaoPosition,
			&qaoIsCorrect,
		); err != nil {
			return nil, fmt.Errorf("scan quiz row: %w", err)
		}

		found = true

		if qqID == nil {
			continue
		}

		qID := *qqID

		question, exists := questionsMap[qID]
		if !exists {
			if qqQuizID == nil || qqText == nil || qqPosition == nil {
				return nil, fmt.Errorf("incomplete quiz question data: question_id=%s", qID)
			}

			question = &QuizQuestion{
				ID:       *qqID,
				QuizID:   *qqQuizID,
				Text:     *qqText,
				Position: *qqPosition,
				Options:  make([]QuizAnswerOptions, 0),
			}

			questionsMap[qID] = question
			questionOrder = append(questionOrder, qID)
		}

		if qaoID == nil {
			continue
		}

		if qaoQuestionID == nil || qaoText == nil || qaoPosition == nil || qaoIsCorrect == nil {
			return nil, fmt.Errorf("incomplete quiz answer option data: option_id=%s", *qaoID)
		}

		question.Options = append(question.Options, QuizAnswerOptions{
			ID:             *qaoID,
			QuizQuestionID: *qaoQuestionID,
			Text:           *qaoText,
			Position:       *qaoPosition,
			IsCorrect:      *qaoIsCorrect,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate quiz rows: %w", err)
	}

	if !found {
		return nil, ErrQuizNotFound
	}

	for _, questionID := range questionOrder {
		quiz.Questions = append(quiz.Questions, *questionsMap[questionID])
	}

	return &quiz, nil
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
			Options: make([]QuizAnswerOptions, 0, len(question.Options)),
		}
		err = tx.QueryRow(ctx, questionCreateQuery, createdQuiz.ID, question.Text, question.Position).Scan(&createdQuestion.ID, &createdQuestion.QuizID, &createdQuestion.Text, &createdQuestion.Position)
		if err != nil {
			return nil, fmt.Errorf("create quiz question: %w", err)
		}
		const optionCreateQuery = `INSERT INTO quiz_answer_options (quiz_question_id, text, position, is_correct) VALUES ($1, $2, $3, $4) RETURNING id, quiz_question_id, text, position, is_correct`
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
