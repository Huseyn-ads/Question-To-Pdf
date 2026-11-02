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

func (s *AttemptService) SaveAnswer(ctx context.Context, userID string, attemptID string, quizQuestionID string, request SaveAnswerRequest) (*AttemptAnswer, error) {
	if _, err := uuid.Parse(attemptID); err != nil {
		return nil, ErrInvalidAttemptID
	}

	if _, err := uuid.Parse(quizQuestionID); err != nil {
		return nil, quiz.ErrWrongQuizQuestionID
	}

	if _, err := uuid.Parse(request.OptionID); err != nil {
		return nil, ErrInvalidOptionID
	}

	attemptAnswer, err := s.repository.SaveAnswer(ctx, userID, attemptID, quizQuestionID, request.OptionID)

	if err != nil {
		switch {
		case errors.Is(err, ErrAttemptNotFound):
			return nil, ErrAttemptNotFound

		case errors.Is(err, ErrAnswerTargetNotFound):
			return nil, ErrAnswerTargetNotFound

		case errors.Is(err, ErrAttemptClosed):
			return nil, ErrAttemptClosed

		default:
			return nil, fmt.Errorf("save answer: %w", err)
		}
	}

	return attemptAnswer, nil
}

func (s *AttemptService) Finish(ctx context.Context, userID string, attemptID string) (*AttemptResult, error) {

	if _, err := uuid.Parse(attemptID); err != nil {
		return nil, ErrInvalidAttemptID
	}

	attemptResult, err := s.repository.Finish(ctx, userID, attemptID)

	if err != nil {

		if errors.Is(err, ErrAttemptNotFound) {
			return nil, ErrAttemptNotFound
		}
		return nil, fmt.Errorf("finish attempt result: %w", err)
	}

	return attemptResult, nil

}

func (s *AttemptService) GetResult(ctx context.Context, userID string, attemptID string) (*AttemptResult, error) {
	if _, err := uuid.Parse(attemptID); err != nil {
		return nil, ErrInvalidAttemptID
	}

	attemptResult, err := s.repository.GetResult(
		ctx,
		userID,
		attemptID,
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrAttemptNotFound):
			return nil, ErrAttemptNotFound

		case errors.Is(err, ErrAttemptInProgress):
			return nil, ErrAttemptInProgress

		default:
			return nil, fmt.Errorf(
				"get attempt result: %w",
				err,
			)
		}
	}

	return attemptResult, nil
}

func (s *AttemptService) GetHistory(ctx context.Context, userID string, quizID string, limit int, offset int) ([]AttemptSummary, error) {
	if _, err := uuid.Parse(quizID); err != nil {
		return nil, quiz.ErrInvalidQuizID
	}

	if limit <= 0 || limit > 100 {
		return nil, quiz.ErrInvalidLimit
	}

	if offset < 0 {
		return nil, quiz.ErrInvalidOffset
	}

	attemptSummaries, err := s.repository.GetHistory(ctx, userID, quizID, limit, offset)

	if err != nil {
		return nil, fmt.Errorf("get history: %w", err)
	}

	return attemptSummaries, nil
}
