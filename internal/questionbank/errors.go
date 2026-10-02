package questionbank

import "errors"

var (
	ErrTitleRequired        = errors.New("title is required")
	ErrTitleTooLong         = errors.New("title is too long")
	ErrQuestionBankNotFound = errors.New("question bank not found")
	ErrNoFieldsToUpdate     = errors.New("no fields to update")
)
