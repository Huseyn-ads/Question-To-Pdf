package questionimport

import (
	"context"
	"fmt"
	"qtp/internal/importer"
	"qtp/internal/questionbank"
)

type Service struct {
	questionBankService *questionbank.QuestionBankService
}

func NewService(
	questionBankService *questionbank.QuestionBankService,
) *Service {
	return &Service{
		questionBankService: questionBankService,
	}
}

func (service *Service) Preview(
	ctx context.Context,
	userID string,
	bankID string,
	filePath string,
) (*importer.PreviewResult, error) {
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
		importer.ExtractParagraphs(filePath)
	if err != nil {
		return nil, fmt.Errorf(
			"extract DOCX: %w",
			err,
		)
	}

	questions := importer.ParseQuestions(paragraphs)

	return &importer.PreviewResult{
		QuestionsCount: len(questions),
		Warnings:       make([]string, 0),
		Questions:      questions,
	}, nil
}
