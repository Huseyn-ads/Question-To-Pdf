package attempt

import "errors"

var (
	ErrInvalidAttemptID     = errors.New("invalid attempt id")
	ErrAttemptNotFound      = errors.New("attempt not found")
	ErrAttemptClosed        = errors.New("your attempt expired")
	ErrAnswerTargetNotFound = errors.New("answer target not found")
	ErrInvalidOptionID      = errors.New("invalid option id")
)
