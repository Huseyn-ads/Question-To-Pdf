package quiz

import "time"

type Quiz struct {
	ID               string
	UserID           string
	QuestionBankID   string
	QuestionCount    int
	TimeLimitMinutes int
	ShuffleOptions   bool
	CreatedAt        time.Time
	Questions        []QuizQuestion
}

type QuizQuestion struct {
	ID       string
	QuizID   string
	Text     string
	Position int
	Options  []QuizAnswerOptions
}

type QuizAnswerOptions struct {
	ID             string
	QuizQuestionID string
	Text           string
	Position       int
	IsCorrect      bool
}
