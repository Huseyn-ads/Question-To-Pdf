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

type AttemptDetails struct {
	Attempt
	ServerTime time.Time
	Questions  []AttemptQuestion
	Answers    []AttemptAnswer
}

type AttemptQuestion struct {
	ID       string
	QuizID   string
	Text     string
	Position int
	Options  []AttemptOption
}

type AttemptOption struct {
	ID             string
	QuizQuestionID string
	Text           string
	Position       int
}

type AttemptAnswer struct {
	ID                 string
	QuizAttemptID      string
	QuizQuestionID     string
	QuizAnswerOptionID string
	UpdatedAt          time.Time
}

type AttemptResult struct {
	AttemptID           string
	QuizID              string
	Status              QuizStatus
	TotalQuestions      int
	AnsweredQuestions   int
	CorrectAnswers      int
	WrongAnswers        int
	UnansweredQuestions int
	Percentage          float64
	StartedAt           time.Time
	FinishedAt          time.Time

	CorrectAnswersList      []ResultQuestion
	WrongAnswersList        []ResultQuestion
	UnansweredQuestionsList []ResultQuestion
}

type ResultQuestion struct {
	ID               string
	QuizID           string
	Text             string
	Position         int
	SelectedOptionID *string
	IsCorrect        *bool
	Options          []ResultOption
}

type ResultOption struct {
	ID             string
	QuizQuestionID string
	Text           string
	Position       int
	IsCorrect      bool
	IsSelected     bool
}
