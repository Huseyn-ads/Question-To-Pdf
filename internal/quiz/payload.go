package quiz

import "time"

type CreateRequest struct {
	QuestionCount    int  `json:"question_count"`
	TimeLimitMinutes int  `json:"time_limit_minutes"`
	ShuffleOptions   bool `json:"shuffle_options"`
}

type QuizSummary struct {
	ID             string    `json:"id"`
	QuestionBankID string    `json:"question_bank_id"`
	QuestionCount  int       `json:"question_count"`
	CreatedAt      time.Time `json:"created_at"`
}

type QuizResponse struct {
	ID               string    `json:"id"`
	QuestionBankID   string    `json:"question_bank_id"`
	QuestionCount    int       `json:"question_count"`
	TimeLimitMinutes int       `json:"time_limit_minutes"`
	ShuffleOptions   bool      `json:"shuffle_options"`
	CreatedAt        time.Time `json:"created_at"`
	Questions        []QuizQuestionResponse
}

type QuizQuestionResponse struct {
	ID       string                      `json:"id"`
	QuizID   string                      `json:"question_id"`
	Text     string                      `json:"text"`
	Position int                         `json:"position"`
	Options  []QuizAnswerOptionsResponse `json:"quiz_answer_options"`
}

type QuizAnswerOptionsResponse struct {
	ID             string `json:"id"`
	QuizQuestionID string `json:"quiz_question_id"`
	Text           string `json:"text"`
	Position       int    `json:"position"`
}

type QuizListResponse struct {
	Items  []QuizListItems `json:"items"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

type QuizListItems struct {
	ID               string    `json:"id"`
	QuestionBankID   string    `json:"question_bank_id"`
	QuestionCount    int       `json:"question_count"`
	TimeLimitMinutes int       `json:"time_limit_minutes"`
	ShuffleOptions   bool      `json:"shuffle_options"`
	CreatedAt        time.Time `json:"created_at"`
}

type QuizPage struct {
	Number     int `json:"number"`
	Size       int `json:"size"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

type PaginatedQuizListResponse struct {
	Items []QuizResponse `json:"items"`
	Page  QuizPage       `json:"page"`
}
