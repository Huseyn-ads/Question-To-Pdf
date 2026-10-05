package question

import "time"

type CreateRequest struct {
	Text    string                `json:"text"`
	Options []CreateOptionRequest `json:"options"`
}

type CreateOptionRequest struct {
	Text      string `json:"text"`
	IsCorrect bool   `json:"is_correct"`
}

type Response struct {
	ID             string           `json:"id"`
	QuestionBankID string           `json:"question_bank_id"`
	Text           string           `json:"text"`
	Options        []OptionResponse `json:"options"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

type OptionResponse struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Position  int    `json:"position"`
	IsCorrect bool   `json:"is_correct"`
}
