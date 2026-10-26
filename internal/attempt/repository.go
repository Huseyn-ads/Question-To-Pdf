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

func (repo *AttemptRepository) Start(ctx context.Context, userID string, quizID string, attempt Attempt) (*Attempt, error) {

	createdAttempt := Attempt{
		QuizID:        quizID,
		UserID:        userID,
		Status:        attempt.Status,
		StartedAt:     attempt.StartedAt,
		DeadlineAt:    attempt.DeadlineAt,
		FinishedAt:    attempt.FinishedAt,
		CorrectCount:  attempt.CorrectCount,
		GradableCount: attempt.GradableCount,
	}

	tx, err := repo.db.Begin(ctx)

	if err != nil {
		return nil, fmt.Errorf("begin to start attempt transaction: %w", err)
	}

	defer tx.Rollback(ctx)
	const checkQuery = `
	SELECT id FROM quizzes WHERE id = $1 AND user_id = $2 FOR UPDATE;
	`

	var foundQuizID string

	err = tx.QueryRow(ctx, checkQuery, quizID, userID).Scan(&foundQuizID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, quiz.ErrQuizNotFound
		}
		return nil, fmt.Errorf("check quiz ownership: %w", err)
	}

	var foundActiveAttempt Attempt

	const findActiveAttemptQuery = `
    SELECT id, quiz_id, user_id, status,
           started_at, deadline_at, finished_at,
           correct_count, gradable_count
    FROM quiz_attempts
    WHERE quiz_id = $1
      AND user_id = $2
      AND status = 'in_progress'
	`

	err = tx.QueryRow(ctx, findActiveAttemptQuery, quizID, userID).
		Scan(
			&foundActiveAttempt.ID,
			&foundActiveAttempt.QuizID,
			&foundActiveAttempt.UserID,
			&foundActiveAttempt.Status,
			&foundActiveAttempt.StartedAt,
			&foundActiveAttempt.DeadlineAt,
			&foundActiveAttempt.FinishedAt,
			&foundActiveAttempt.CorrectCount,
			&foundActiveAttempt.GradableCount,
		)

	var foundExpiredAttempt Attempt
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			const expiredAttemptQuery = `
			SELECT id, quiz_id, user_id, status,
           started_at, deadline_at, finished_at,
           correct_count, gradable_count
    FROM quiz_attempts
    WHERE quiz_id = $1
      AND user_id = $2
      AND status = 'expired'
			`

			err = tx.QueryRow(ctx, expiredAttemptQuery, quizID, userID).Scan(
				&foundExpiredAttempt.ID,
				&foundExpiredAttempt.QuizID,
				&foundExpiredAttempt.UserID,
				&foundExpiredAttempt.Status,
				&foundExpiredAttempt.StartedAt,
				&foundExpiredAttempt.DeadlineAt,
				&foundExpiredAttempt.FinishedAt,
				&foundExpiredAttempt.CorrectCount,
				&foundExpiredAttempt.GradableCount,
			)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					const createAttemptQuery = `
					INSERT INTO quiz_attempts (
					quiz_id,
					user_id,
					status,
					started_at,
					deadline_at,
					finished_at,
					correct_count,
					gradable_count)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
					RETURNING id;
					`

					err := tx.QueryRow(ctx, createAttemptQuery,
						quizID, userID, attempt.Status,
						attempt.StartedAt, attempt.DeadlineAt, attempt.FinishedAt,
						attempt.CorrectCount, attempt.GradableCount).Scan(&createdAttempt.ID)

					if err != nil {
						return nil, fmt.Errorf("create attempt: %w", err)
					}
				}
				return nil, fmt.Errorf("create attempt: %w", err)
			}
			return nil, fmt.Errorf("active attempt: %w", err)
		}
	} else {
		return &foundActiveAttempt, nil
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit start attempt transaction: %w", err)
	}
	return &createdAttempt, nil
}
