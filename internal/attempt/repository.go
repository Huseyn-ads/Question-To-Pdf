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
