package question

import (
	"context"
	"errors"
	"fmt"
	"qtp/internal/questionbank"
	"strings"

	"github.com/google/uuid"
)

type QuestionService struct {
	repository *QuestionRepository
}

func NewQuestionService(repository *QuestionRepository) *QuestionService {
	return &QuestionService{
		repository: repository,
	}
}

func (s *QuestionService) FindByID(ctx context.Context, userID string, bankID string, questionID string) (*Question, error) {
	if _, err := uuid.Parse(bankID); err != nil {
		return nil, questionbank.ErrInvalidQuestionBankID
	}

	if _, err := uuid.Parse(questionID); err != nil {
		return nil, ErrInvalidQuestionID
	}

	foundQuestion, err := s.repository.FindByID(ctx, userID, bankID, questionID)

	if err != nil {
		if errors.Is(err, ErrQuestionNotFound) {
			return nil, ErrQuestionNotFound
		}
		return nil, fmt.Errorf("find question: %w", err)
	}

	return foundQuestion, nil
}

func (s *QuestionService) Create(ctx context.Context, bankID string, userID string, request CreateRequest) (*Question, error) {
	text := strings.TrimSpace(request.Text)
	options := request.Options

	if _, err := uuid.Parse(bankID); err != nil {
		return nil, questionbank.ErrInvalidQuestionBankID
	}

	if text == "" {
		return nil, ErrTextRequired
	}

	if len(options) < 2 {
		return nil, ErrNotEnoughOptions
	}

	correctCount := 0
	seenOptions := make(map[string]struct{}, len(options))
	answerOptions := make([]AnswerOption, 0, len(options))

	for index, option := range options {
		optionText := strings.TrimSpace(option.Text)

		if optionText == "" {
			return nil, ErrOptionTextRequired
		}

		normalizedOption := strings.ToLower(optionText)

		if _, exists := seenOptions[normalizedOption]; exists {
			return nil, ErrDuplicateOptions
		}

		seenOptions[normalizedOption] = struct{}{}

		if option.IsCorrect {
			correctCount++
		}

		answerOptions = append(answerOptions, AnswerOption{
			Text:      optionText,
			Position:  index + 1,
			IsCorrect: option.IsCorrect,
		})
	}

	if correctCount != 1 {
		return nil, ErrExactlyOneCorrectOption
	}
	question := Question{
		QuestionBankID: bankID,
		Text:           text,
		Options:        answerOptions,
	}

	createdQuestion, err := s.repository.Create(ctx, userID, &question)
	if err != nil {
		return nil, fmt.Errorf("create question: %w", err)
	}

	return createdQuestion, nil

}
