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

func (s *QuestionService) ListByBankID(ctx context.Context, userID string, bankID string) ([]Question, error) {
	if _, err := uuid.Parse(bankID); err != nil {
		return nil, questionbank.ErrInvalidQuestionBankID
	}

	questions, err := s.repository.ListByBankID(ctx, userID, bankID)

	if err != nil {
		if errors.Is(err, questionbank.ErrQuestionBankNotFound) {
			return nil, questionbank.ErrQuestionBankNotFound
		}
		return nil, fmt.Errorf("list by bank id: %w", err)
	}

	return questions, nil

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
	text, options, err := prepareQuestionData(
		request.Text,
		request.Options,
	)
	if err != nil {
		return nil, err
	}

	newQuestion := &Question{
		QuestionBankID: bankID,
		Text:           text,
		Options:        options,
	}

	createdQuestion, err := s.repository.Create(
		ctx,
		userID,
		newQuestion,
	)
	if err != nil {
		return nil, fmt.Errorf("create question: %w", err)
	}

	return createdQuestion, nil
}

func (s *QuestionService) Update(ctx context.Context, userID string, bankID string, questionID string, request UpdateRequest) (*Question, error) {
	if _, err := uuid.Parse(bankID); err != nil {
		return nil, questionbank.ErrInvalidQuestionBankID
	}

	if _, err := uuid.Parse(questionID); err != nil {
		return nil, ErrInvalidQuestionID
	}

	text, options, err := prepareQuestionData(
		request.Text,
		request.Options,
	)
	if err != nil {
		return nil, err
	}

	newQuestion := &Question{
		QuestionBankID: bankID,
		Text:           text,
		Options:        options,
	}

	updatedQuestion, err := s.repository.Update(ctx, userID, bankID, questionID, newQuestion)
	if err != nil {
		return nil, fmt.Errorf(
			"update question: %w",
			err,
		)
	}

	return updatedQuestion, nil
}

func (s *QuestionService) Delete(ctx context.Context, userID string, bankID string, questionID string) error {
	if _, err := uuid.Parse(bankID); err != nil {
		return questionbank.ErrInvalidQuestionBankID
	}

	if _, err := uuid.Parse(questionID); err != nil {
		return ErrInvalidQuestionID
	}

	err := s.repository.Delete(ctx, userID, bankID, questionID)

	if err != nil {
		if errors.Is(err, ErrQuestionNotFound) {
			return ErrQuestionNotFound
		}
		return fmt.Errorf("delete question: %w", err)
	}

	return nil

}

func prepareQuestionData(text string, options []CreateOptionRequest) (string, []AnswerOption, error) {
	text = strings.TrimSpace(text)

	if text == "" {
		return "", nil, ErrTextRequired
	}

	if len(options) < 2 {
		return "", nil, ErrNotEnoughOptions
	}

	preparedOptions := make(
		[]AnswerOption,
		0,
		len(options),
	)

	seenOptions := make(
		map[string]struct{},
		len(options),
	)

	correctCount := 0

	for index, option := range options {
		optionText := strings.TrimSpace(option.Text)

		if optionText == "" {
			return "", nil, ErrOptionTextRequired
		}

		normalizedOption := strings.ToLower(optionText)

		if _, exists := seenOptions[normalizedOption]; exists {
			return "", nil, ErrDuplicateOptions
		}

		seenOptions[normalizedOption] = struct{}{}

		if option.IsCorrect {
			correctCount++
		}

		preparedOptions = append(
			preparedOptions,
			AnswerOption{
				Text:      optionText,
				Position:  index + 1,
				IsCorrect: option.IsCorrect,
			},
		)
	}

	if correctCount != 1 {
		return "", nil, ErrExactlyOneCorrectOption
	}

	return text, preparedOptions, nil
}
