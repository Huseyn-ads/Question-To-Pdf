package questionbank

import "time"

type QuestionBank struct {
	ID          string
	UserID      string
	Title       string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
