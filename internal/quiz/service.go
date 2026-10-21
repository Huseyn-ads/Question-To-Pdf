package quiz

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"qtp/internal/question"
	"qtp/internal/questionbank"

	"github.com/google/uuid"
)

type QuizService struct {
	repository      *QuizRepository
	questionService *question.QuestionService
}

func NewQuizService(repository *QuizRepository, questionService *question.QuestionService) *QuizService {
	return &QuizService{
		repository:      repository,
		questionService: questionService,
	}
}

func (s *QuizService) ListByUserID(ctx context.Context, userID string, limit int, offset int) ([]QuizListItems, error) {
	if limit <= 0 || limit > 100 {
		return nil, ErrInvalidLimit
	}

	if offset < 0 {
		return nil, ErrInvalidOffset
	}

	quizzes, err := s.repository.ListByUserID(ctx, userID, limit, offset)

	if err != nil {
		return nil, fmt.Errorf("list quizzes by user id: %w", err)
	}

	return quizzes, nil

}

func (s *QuizService) FindByOwnedID(ctx context.Context, userID string, quizID string) (*Quiz, error) {
	if _, err := uuid.Parse(quizID); err != nil {
		return nil, questionbank.ErrInvalidQuestionBankID
	}

	foundedQuiz, err := s.repository.FindByOwnedID(ctx, userID, quizID)

	if err != nil {
		if errors.Is(err, ErrInvalidQuizID) {
			return nil, ErrInvalidQuizID
		}
		if errors.Is(err, ErrQuizNotFound) {
			return nil, ErrQuizNotFound
		}
		return nil, fmt.Errorf("found quiz: %w", err)
	}

	return foundedQuiz, nil
}

func (s *QuizService) CreateQuiz(ctx context.Context, userID string, bankID string, request CreateRequest) (*Quiz, error) {
	quiz := Quiz{
		QuestionBankID:   bankID,
		QuestionCount:    request.QuestionCount,
		TimeLimitMinutes: request.TimeLimitMinutes,
		ShuffleOptions:   request.ShuffleOptions,
		Questions:        make([]QuizQuestion, 0),
	}
	if _, err := uuid.Parse(bankID); err != nil {
		return nil, questionbank.ErrInvalidQuestionBankID
	}

	if request.QuestionCount <= 0 {
		return nil, ErrInvalidQuestionCount
	}

	if request.TimeLimitMinutes < 0 || request.TimeLimitMinutes > 180 {
		return nil, ErrInvalidTimeLimiter
	}

	questions, err := s.questionService.ListByBankID(ctx, userID, bankID)

	if err != nil {
		return nil, fmt.Errorf("getting questions from ListByBankID: %w", err)
	}

	if len(questions) == 0 {
		return nil, ErrEmptyQuestions
	}

	if request.QuestionCount > len(questions) {
		return nil, ErrNotEnoughtQuestions
	}

	shuffled := append([]question.Question(nil), questions...)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	for index, question := range shuffled {
		if index >= request.QuestionCount {
			break
		}
		qQuestion := QuizQuestion{
			QuizID:   quiz.ID,
			Text:     question.Text,
			Position: index + 1,
			Options:  make([]QuizAnswerOptions, 0),
		}

		for index, option := range question.Options {
			qAnswerOption := QuizAnswerOptions{
				QuizQuestionID: qQuestion.ID,
				Text:           option.Text,
				Position:       index + 1,
				IsCorrect:      option.IsCorrect,
			}
			qQuestion.Options = append(qQuestion.Options, qAnswerOption)
		}
		if request.ShuffleOptions {
			rand.Shuffle(len(qQuestion.Options), func(i, j int) {
				qQuestion.Options[i], qQuestion.Options[j] = qQuestion.Options[j], qQuestion.Options[i]
			})
		}

		for index := range qQuestion.Options {
			qQuestion.Options[index].Position = index + 1
		}

		quiz.Questions = append(quiz.Questions, qQuestion)
	}

	createdQuiz, err := s.repository.Create(ctx, userID, bankID, quiz)

	if err != nil {
		return nil, fmt.Errorf("create quiz: %w", err)
	}

	return createdQuiz, nil
}

func (s *QuizService) Delete(ctx context.Context, userID string, quizID string) error {
	if _, err := uuid.Parse(quizID); err != nil {
		return ErrInvalidQuizID
	}

	err := s.repository.Delete(ctx, userID, quizID)

	if err != nil {
		if errors.Is(err, ErrQuizNotFound) {
			return ErrQuizNotFound
		}
		return fmt.Errorf("delete quiz: %w", err)
	}

	return nil

}
