package attempt

import (
	"context"
	"errors"
	"fmt"
	"qtp/internal/quiz"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AttemptRepository struct {
	db *pgxpool.Pool
}

func NewAttemptRepository(db *pgxpool.Pool) *AttemptRepository {
	return &AttemptRepository{
		db: db,
	}
}

func (repo *AttemptRepository) Start(ctx context.Context, userID string, quizID string) (*Attempt, bool, error) {
	tx, err := repo.db.Begin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("begin start attempt transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Locking the quiz serializes concurrent starts for the same quiz. The
	// partial unique index on active attempts remains the final database-level
	// protection against duplicate in-progress attempts.
	const findQuizQuery = `
		SELECT time_limit_minutes
		FROM quizzes
		WHERE id = $1 AND user_id = $2
		FOR UPDATE;
	`

	var timeLimitMinutes int
	if err := tx.QueryRow(ctx, findQuizQuery, quizID, userID).Scan(&timeLimitMinutes); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, quiz.ErrQuizNotFound
		}

		return nil, false, fmt.Errorf("find owned quiz for attempt: %w", err)
	}

	const findActiveAttemptQuery = `
		SELECT
			id,
			quiz_id,
			user_id,
			status,
			started_at,
			deadline_at,
			finished_at,
			correct_count,
			gradable_count,
			deadline_at IS NOT NULL AND deadline_at <= NOW()
		FROM quiz_attempts
		WHERE quiz_id = $1
			AND user_id = $2
			AND status = 'in_progress'
		FOR UPDATE;
	`

	activeAttempt := &Attempt{}
	var deadlineExpired bool

	err = tx.QueryRow(ctx, findActiveAttemptQuery, quizID, userID).Scan(
		&activeAttempt.ID,
		&activeAttempt.QuizID,
		&activeAttempt.UserID,
		&activeAttempt.Status,
		&activeAttempt.StartedAt,
		&activeAttempt.DeadlineAt,
		&activeAttempt.FinishedAt,
		&activeAttempt.CorrectCount,
		&activeAttempt.GradableCount,
		&deadlineExpired,
	)

	switch {
	case err == nil && !deadlineExpired:
		if err := tx.Commit(ctx); err != nil {
			return nil, false, fmt.Errorf("commit existing attempt transaction: %w", err)
		}

		return activeAttempt, false, nil

	case err == nil && deadlineExpired:
		const expireAttemptQuery = `
			UPDATE quiz_attempts
			SET
				status = 'expired',
				finished_at = NOW()
			WHERE id = $1 AND status = 'in_progress';
		`

		result, err := tx.Exec(ctx, expireAttemptQuery, activeAttempt.ID)
		if err != nil {
			return nil, false, fmt.Errorf("expire active attempt: %w", err)
		}

		if result.RowsAffected() != 1 {
			return nil, false, fmt.Errorf("expire active attempt: expected one updated row, got %d", result.RowsAffected())
		}

	case errors.Is(err, pgx.ErrNoRows):
		// There is no active attempt, so a new one can be created.

	default:
		return nil, false, fmt.Errorf("find active attempt: %w", err)
	}

	const createAttemptQuery = `
		INSERT INTO quiz_attempts (
			quiz_id,
			user_id,
			status,
			started_at,
			deadline_at
		)
		VALUES (
			$1,
			$2,
			'in_progress',
			NOW(),
			CASE
				WHEN $3 > 0 THEN NOW() + ($3 * INTERVAL '1 minute')
				ELSE NULL
			END
		)
		RETURNING
			id,
			quiz_id,
			user_id,
			status,
			started_at,
			deadline_at,
			finished_at,
			correct_count,
			gradable_count;
	`

	createdAttempt := &Attempt{}
	err = tx.QueryRow(
		ctx,
		createAttemptQuery,
		quizID,
		userID,
		timeLimitMinutes,
	).Scan(
		&createdAttempt.ID,
		&createdAttempt.QuizID,
		&createdAttempt.UserID,
		&createdAttempt.Status,
		&createdAttempt.StartedAt,
		&createdAttempt.DeadlineAt,
		&createdAttempt.FinishedAt,
		&createdAttempt.CorrectCount,
		&createdAttempt.GradableCount,
	)
	if err != nil {
		return nil, false, fmt.Errorf("create attempt: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf("commit start attempt transaction: %w", err)
	}

	return createdAttempt, true, nil
}

func (repo *AttemptRepository) GetAttempt(ctx context.Context, userID string, attemptID string) (*AttemptDetails, error) {
	tx, err := repo.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin get attempt transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	details := &AttemptDetails{
		Questions: make([]AttemptQuestion, 0),
		Answers:   make([]AttemptAnswer, 0),
	}

	const findAttemptQuery = `
		SELECT
			id,
			quiz_id,
			user_id,
			status,
			started_at,
			deadline_at,
			finished_at,
			correct_count,
			gradable_count,
			NOW(),
			status = 'in_progress'
				AND deadline_at IS NOT NULL
				AND deadline_at <= NOW()
		FROM quiz_attempts
		WHERE id = $1 AND user_id = $2
		FOR UPDATE;
	`

	var deadlineExpired bool
	err = tx.QueryRow(ctx, findAttemptQuery, attemptID, userID).Scan(
		&details.ID,
		&details.QuizID,
		&details.UserID,
		&details.Status,
		&details.StartedAt,
		&details.DeadlineAt,
		&details.FinishedAt,
		&details.CorrectCount,
		&details.GradableCount,
		&details.ServerTime,
		&deadlineExpired,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAttemptNotFound
		}

		return nil, fmt.Errorf("find owned attempt: %w", err)
	}

	if deadlineExpired {
		const expireAttemptQuery = `
			UPDATE quiz_attempts
			SET
				status = 'expired',
				finished_at = NOW()
			WHERE id = $1 AND status = 'in_progress'
			RETURNING status, finished_at;
		`

		if err := tx.QueryRow(ctx, expireAttemptQuery, details.ID).Scan(
			&details.Status,
			&details.FinishedAt,
		); err != nil {
			return nil, fmt.Errorf("expire attempt while getting it: %w", err)
		}
	}

	const findQuestionsQuery = `
		SELECT
			qq.id,
			qq.quiz_id,
			qq.text,
			qq.position,
			qao.id,
			qao.quiz_question_id,
			qao.text,
			qao.position
		FROM quiz_questions qq
		JOIN quiz_answer_options qao
			ON qao.quiz_question_id = qq.id
		WHERE qq.quiz_id = $1
		ORDER BY qq.position, qao.position;
	`

	rows, err := tx.Query(ctx, findQuestionsQuery, details.QuizID)
	if err != nil {
		return nil, fmt.Errorf("query attempt questions: %w", err)
	}

	for rows.Next() {
		question := AttemptQuestion{}
		option := AttemptOption{}

		if err := rows.Scan(
			&question.ID,
			&question.QuizID,
			&question.Text,
			&question.Position,
			&option.ID,
			&option.QuizQuestionID,
			&option.Text,
			&option.Position,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan attempt question: %w", err)
		}

		isFirstQuestion := len(details.Questions) == 0
		if isFirstQuestion || details.Questions[len(details.Questions)-1].ID != question.ID {
			question.Options = make([]AttemptOption, 0)
			details.Questions = append(details.Questions, question)
		}

		lastQuestionIndex := len(details.Questions) - 1
		details.Questions[lastQuestionIndex].Options = append(
			details.Questions[lastQuestionIndex].Options,
			option,
		)
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate attempt questions: %w", err)
	}
	rows.Close()

	const findAnswersQuery = `
		SELECT
			id,
			quiz_attempt_id,
			quiz_question_id,
			quiz_answer_option_id,
			updated_at
		FROM attempt_answers
		WHERE quiz_attempt_id = $1
		ORDER BY updated_at, id;
	`

	answerRows, err := tx.Query(ctx, findAnswersQuery, details.ID)
	if err != nil {
		return nil, fmt.Errorf("query attempt answers: %w", err)
	}

	for answerRows.Next() {
		answer := AttemptAnswer{}
		if err := answerRows.Scan(
			&answer.ID,
			&answer.QuizAttemptID,
			&answer.QuizQuestionID,
			&answer.QuizAnswerOptionID,
			&answer.UpdatedAt,
		); err != nil {
			answerRows.Close()
			return nil, fmt.Errorf("scan attempt answer: %w", err)
		}

		details.Answers = append(details.Answers, answer)
	}

	if err := answerRows.Err(); err != nil {
		answerRows.Close()
		return nil, fmt.Errorf("iterate attempt answers: %w", err)
	}
	answerRows.Close()

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit get attempt transaction: %w", err)
	}

	return details, nil
}
