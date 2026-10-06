package question

func toResponse(optionResponses []OptionResponse, question *Question) *Response {
	for _, option := range question.Options {
		optionResponses = append(optionResponses, OptionResponse{
			ID:        option.ID,
			Text:      option.Text,
			Position:  option.Position,
			IsCorrect: option.IsCorrect,
		})
	}

	response := Response{
		ID:             question.ID,
		QuestionBankID: question.QuestionBankID,
		Text:           question.Text,
		Options:        optionResponses,
		CreatedAt:      question.CreatedAt,
		UpdatedAt:      question.UpdatedAt,
	}

	return &response
}
