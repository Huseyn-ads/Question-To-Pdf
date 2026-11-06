package questionimport

import (
	"context"
	"fmt"
	"qtp/internal/importer"
	"qtp/internal/question"
	"qtp/internal/questionbank"
)

type Service struct {
	questionBankService *questionbank.QuestionBankService
	questionRepository  *question.QuestionRepository
}

func NewService(
	questionBankService *questionbank.QuestionBankService,
	questionRepository *question.QuestionRepository,
) *Service {
	return &Service{
		questionBankService: questionBankService,
		questionRepository:  questionRepository,
	}
}

func (service *Service) Preview(ctx context.Context, userID string, bankID string, filePath string, answerMode string) (*importer.PreviewResult, error) {
	_, err := service.questionBankService.FindByID(
		ctx,
		bankID,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"find question bank: %w",
			err,
		)
	}

	paragraphs, err :=
		importer.ExtractParagraphsContext(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: %w",
			ErrInvalidDocument,
			err,
		)
	}

	result := importer.BuildPreview(paragraphs, importer.ParseOptions{
		AnswerMode: answerMode,
	})

	return &result, nil
}

func (service *Service) CheckBank(ctx context.Context, userID, bankID string) error {
	_, err := service.questionBankService.FindByID(ctx, bankID, userID)
	return err
}

func (s *Service) Import(ctx context.Context, userID string, bankID string, request ImportRequest) (*ImportResponse, error) {
	models := make([]question.Question, 0, len(request.Questions))
	optionsCount := 0

	for _, q := range request.Questions {
		var options []question.AnswerOption

		for index, option := range q.Options {
			options = append(options, question.AnswerOption{
				Text:      option.Text,
				Position:  index + 1,
				IsCorrect: option.IsCorrect,
			})
		}
		models = append(models, question.Question{
			QuestionBankID: bankID,
			Text:           q.Text,
			Options:        options,
		})

		optionsCount += len(options)
	}

	err := s.questionRepository.CreateMany(
		ctx,
		userID,
		bankID,
		models,
	)

	if err != nil {
		return nil, fmt.Errorf("import question: %w", err)
	}

	return &ImportResponse{
		ImportedQuestions: len(models),
		ImportedOptions:   optionsCount,
	}, nil
}
