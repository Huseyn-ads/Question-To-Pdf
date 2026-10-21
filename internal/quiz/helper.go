package quiz

func toResponseQuiz(quizQuestionResponse []QuizQuestionResponse, quiz *Quiz) *QuizResponse {
	for _, question := range quiz.Questions {

		quizAnswerOptionsResponse := make([]QuizAnswerOptionsResponse, 0, len(question.Options))

		for _, option := range question.Options {
			quizAnswerOptionsResponse = append(quizAnswerOptionsResponse, QuizAnswerOptionsResponse{
				ID:             option.ID,
				QuizQuestionID: option.QuizQuestionID,
				Text:           option.Text,
				Position:       option.Position,
			})
		}

		quizQuestionResponse = append(quizQuestionResponse, QuizQuestionResponse{
			ID:       question.ID,
			QuizID:   question.QuizID,
			Text:     question.Text,
			Position: question.Position,
			Options:  quizAnswerOptionsResponse,
		})
	}

	response := QuizResponse{
		ID:               quiz.ID,
		QuestionBankID:   quiz.QuestionBankID,
		QuestionCount:    quiz.QuestionCount,
		TimeLimitMinutes: quiz.TimeLimitMinutes,
		ShuffleOptions:   quiz.ShuffleOptions,
		CreatedAt:        quiz.CreatedAt,
		Questions:        quizQuestionResponse,
	}

	return &response
}
