package attempt

import "errors"

var (
	ErrInvalidAttemptID = errors.New("invalid attempt id")
	ErrAttemptNotFound  = errors.New("attempt not found")
)
