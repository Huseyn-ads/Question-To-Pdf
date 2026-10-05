package question

import "time"

type Question struct {
	ID             string
	QuestionBankID string
	Text           string
	Options        []AnswerOption
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type AnswerOption struct {
	ID         string
	QuestionID string
	Text       string
	Position   int
	IsCorrect  bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
