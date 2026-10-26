package attempt

import "time"

type QuizStatus string

const (
	StatusInProgress QuizStatus = "in_progress"
	StatusCompleted  QuizStatus = "completed"
	StatusExpired    QuizStatus = "expired"
)

type Attempt struct {
	ID            string
	QuizID        string
	UserID        string
	Status        QuizStatus
	StartedAt     time.Time
	DeadlineAt    *time.Time
	FinishedAt    *time.Time
	CorrectCount  int
	GradableCount int
}

type AttemptAnswers struct {
	ID                 string
	QuizAttemptID      string
	QuizQuestionID     string
	QuizAnswerOptionID string
	UpdatedAt          time.Time
}
