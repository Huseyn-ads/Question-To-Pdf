package quiz

import "errors"

var (
	ErrInvalidQuizID        = errors.New("invalid quiz id")
	ErrQuizNotFound         = errors.New("quiz not found")
	ErrInvalidQuestionCount = errors.New("incorrect question count")
	ErrNotEnoughtQuestions  = errors.New("not enough questions in bank")
	ErrInvalidTimeLimiter   = errors.New("incorrect time limiter")
	ErrWrongQuizQuestionID  = errors.New("invalid quiz question id")
	ErrEmptyQuestions       = errors.New("there is no questions")
)
