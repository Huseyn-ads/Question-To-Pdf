package attempt

import "time"

type SavedAnswer struct {
	ID                 string `json:"id"`
	QuizQuestionID     string `json:"quiz_question_id"`
	QuizAnswerOptionID string `json:"quiz_answer_option_id"`
}

type SaveAnswerRequest struct {
	OptionID string `json:"option_id"`
}

type SaveAnswerResponse struct {
	QuizQuestionID string    `json:"quiz_question_id"`
	OptionID       string    `json:"option_id"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type AttemptResponse struct {
	ID         string                    `json:"id"`
	QuizID     string                    `json:"quiz_id"`
	Status     QuizStatus                `json:"status"`
	StartedAt  time.Time                 `json:"started_at"`
	DeadlineAt *time.Time                `json:"deadline_at"`
	FinishedAt *time.Time                `json:"finished_at"`
	ServerTime time.Time                 `json:"server_time"`
	Questions  []AttemptQuestionResponse `json:"questions"`
	Answers    []SavedAnswer             `json:"answers"`
}

type AttemptQuestionResponse struct {
	ID       string                  `json:"id"`
	Text     string                  `json:"text"`
	Position int                     `json:"position"`
	Options  []AttemptOptionResponse `json:"options"`
}

type AttemptOptionResponse struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Position int    `json:"position"`
}

type ResultResponse struct {
	ID            string     `json:"id"`
	QuizID        string     `json:"quiz_id"`
	Status        QuizStatus `json:"status"`
	CorrectCount  int        `json:"correct_count"`
	GradableCount int        `json:"gradable_count"`
	StartedAt     time.Time  `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at"`
}

type FinishSummary struct {
	TotalQuestions    int     `json:"total_questions"`
	AnsweredQuestions int     `json:"answered_questions"`
	CorrectAnswers    int     `json:"correct_answers"`
	WrongAnswers      int     `json:"wrong_answers"`
	UnansweredAnswers int     `json:"unanswered_questions"`
	Percentage        float64 `json:"percentage"`
}
type FinishResponse struct {
	AttemptID           string     `json:"attempt_id"`
	QuizID              string     `json:"quiz_id"`
	Status              QuizStatus `json:"status"`
	TotalQuestions      int        `json:"total_questions"`
	AnsweredQuestions   int        `json:"answered_questions"`
	CorrectAnswers      int        `json:"correct_answers"`
	WrongAnswers        int        `json:"wrong_answers"`
	UnansweredQuestions int        `json:"unanswered_questions"`
	Percentage          float64    `json:"percentage"`
	StartedAt           time.Time  `json:"started_at"`
	FinishedAt          time.Time  `json:"finished_at"`

	CorrectAnswersList []ResultQuestionResponse `json:"correct_answers_list"`

	WrongAnswersList []ResultQuestionResponse `json:"wrong_answers_list"`

	UnansweredQuestionsList []ResultQuestionResponse `json:"unanswered_questions_list"`
}

type ResultQuestionResponse struct {
	ID               string                 `json:"id"`
	Text             string                 `json:"text"`
	Position         int                    `json:"position"`
	SelectedOptionID *string                `json:"selected_option_id"`
	IsCorrect        *bool                  `json:"is_correct"`
	Options          []ResultOptionResponse `json:"options"`
}

type ResultOptionResponse struct {
	ID             string `json:"id"`
	QuizQuestionID string `json:"quiz_question_id"`
	Text           string `json:"text"`
	Position       int    `json:"position"`
	IsCorrect      bool   `json:"is_correct"`
	IsSelected     bool   `json:"is_selected"`
}
