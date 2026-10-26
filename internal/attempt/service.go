package attempt

import (
	"context"
	"errors"
	"fmt"
	"qtp/internal/quiz"

	"github.com/google/uuid"
)

type AttemptService struct {
	repository *AttemptRepository
}

func NewAttemptService(repository *AttemptRepository) *AttemptService {
	return &AttemptService{
		repository: repository,
	}
}

func (service *AttemptService) GetAttempt(ctx context.Context, userID string, attemptID string) (*AttemptDetails, error) {
	if _, err := uuid.Parse(attemptID); err != nil {
		return nil, ErrInvalidAttemptID
	}

	details, err := service.repository.GetAttempt(ctx, userID, attemptID)
	if err != nil {
		if errors.Is(err, ErrAttemptNotFound) {
			return nil, ErrAttemptNotFound
		}

		return nil, fmt.Errorf("get attempt: %w", err)
	}

	return details, nil
}

func (s *AttemptService) Start(ctx context.Context, userID string, quizID string) (*Attempt, error) {
	if _, err := uuid.Parse(quizID); err != nil {
		return nil, quiz.ErrInvalidQuizID
	}

	attempt, _, err := s.repository.Start(ctx, userID, quizID)

	if err != nil {
		if errors.Is(err, ErrAttemptNotFound) {
			return nil, ErrAttemptNotFound
		}

		if errors.Is(err, ErrInvalidAttemptID) {
			return nil, ErrInvalidAttemptID
		}
		return nil, fmt.Errorf("start attempt: %w", err)
	}

	return attempt, nil
}
