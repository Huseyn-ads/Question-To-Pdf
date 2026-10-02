package questionbank

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

type QuestionBankService struct {
	repository *QuestionBankRepository
}

func NewQuestionBankService(repository *QuestionBankRepository) *QuestionBankService {
	return &QuestionBankService{
		repository: repository,
	}
}

func (s *QuestionBankService) ListByUserID(ctx context.Context, userID string) ([]QuestionBank, error) {
	questionBanks, err := s.repository.ListByUserID(ctx, userID)

	if err != nil {
		return nil, fmt.Errorf("list question banks: %w", err)
	}

	return questionBanks, nil

}

func (s *QuestionBankService) FindByID(
	ctx context.Context,
	bankID string,
	userID string,
) (*QuestionBank, error) {
	questionBank, err := s.repository.FindByID(ctx, bankID, userID)
	if err != nil {
		if errors.Is(err, ErrQuestionBankNotFound) {
			return nil, ErrQuestionBankNotFound
		}

		return nil, fmt.Errorf("find question bank: %w", err)
	}

	return questionBank, nil
}

func (s *QuestionBankService) Create(ctx context.Context, userID string, request CreateRequest) (*QuestionBank, error) {
	request.Title = strings.TrimSpace(request.Title)
	request.Description = strings.TrimSpace(request.Description)

	if request.Title == "" {
		return nil, ErrTitleRequired
	}

	titleLength := utf8.RuneCountInString(request.Title)
	if titleLength > 150 {
		return nil, ErrTitleTooLong
	}

	newQuestionBank := &QuestionBank{
		UserID:      userID,
		Title:       request.Title,
		Description: request.Description,
	}

	createdQuestionBank, err := s.repository.Create(ctx, newQuestionBank)

	if err != nil {
		return nil, fmt.Errorf("create question bank: %w", err)
	}

	return createdQuestionBank, nil
}

func (s *QuestionBankService) Update(ctx context.Context, userID string, bankID string, request UpdateRequest) (*QuestionBank, error) {
	if request.Title == nil && request.Description == nil {
		return nil, ErrNoFieldsToUpdate
	}

	*request.Title = strings.TrimSpace(*request.Title)
	*request.Description = strings.TrimSpace(*request.Description)

	if request.Title != nil {
		title := strings.TrimSpace(*request.Title)

		if title == "" {
			return nil, ErrTitleRequired
		}

		if utf8.RuneCountInString(title) > 150 {
			return nil, ErrTitleTooLong
		}

		request.Title = &title
	}

	if request.Description != nil {
		description := strings.TrimSpace(*request.Description)
		request.Description = &description
	}

	updatedQuestionBank, err := s.repository.Update(ctx, request.Title, request.Description, bankID, userID)

	if err != nil {
		if errors.Is(err, ErrQuestionBankNotFound) {
			return nil, ErrQuestionBankNotFound
		}

		return nil, fmt.Errorf("update question bank: %w", err)
	}

	return updatedQuestionBank, nil

}

func (s *QuestionBankService) Delete(ctx context.Context, bankID, userID string) error {
	err := s.repository.Delete(ctx, bankID, userID)
	if err != nil {
		if errors.Is(err, ErrQuestionBankNotFound) {
			return ErrQuestionBankNotFound
		}
		return fmt.Errorf("delete question bank: %w", err)
	}

	return nil

}
