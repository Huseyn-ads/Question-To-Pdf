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

func (s *AttemptService) Start(ctx context.Context, userID string, quizID string) (*AttemptDetails, bool, error) {
	if _, err := uuid.Parse(quizID); err != nil {
		return nil, false, quiz.ErrInvalidQuizID
	}

	startedAttempt, created, err := s.repository.Start(ctx, userID, quizID)
	if err != nil {
		return nil, false, fmt.Errorf("start attempt: %w", err)
	}

	details, err := s.repository.GetAttempt(
		ctx,
		userID,
		startedAttempt.ID,
	)
	if err != nil {
		return nil, false, fmt.Errorf("get started attempt: %w", err)
	}

	return details, created, nil
}
