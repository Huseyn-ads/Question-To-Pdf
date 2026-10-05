package question

import "errors"

var (
	ErrTextRequired            = errors.New("question text is required")
	ErrNotEnoughOptions        = errors.New("question must have at least 2 options")
	ErrOptionTextRequired      = errors.New("option text is required")
	ErrExactlyOneCorrectOption = errors.New("question must have exactly one correct option")
	ErrDuplicateOptions        = errors.New("question options must be unique")
	ErrQuestionNotFound        = errors.New("question not found")
	ErrInvalidQuestionID       = errors.New("invalid question id")
)
