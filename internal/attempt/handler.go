package attempt

import (
	"errors"
	"log/slog"
	"net/http"
	"qtp/internal/httpx"
	"qtp/internal/quiz"
	"qtp/internal/user"
	"strconv"
)

type AttemptHandler struct {
	service *AttemptService
}

func NewAttemptHandler(service *AttemptService) *AttemptHandler {
	return &AttemptHandler{
		service: service,
	}
}

func (handler *AttemptHandler) Get(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := user.UserFromContext(r.Context())

	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	attemptID := r.PathValue("attemptID")

	attempt, err := handler.service.GetAttempt(r.Context(), currentUser.ID, attemptID)

	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidAttemptID):
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrAttemptNotFound):
			httpx.WriteError(w, http.StatusNotFound, err.Error())
		default:
			slog.Error("get attempt", "error", err)
			httpx.WriteError(w, http.StatusInternalServerError, "server error")
		}

		return
	}

	response := toAttemptResponse(attempt)

	httpx.WriteJSON(w, http.StatusOK, response)
}

func (handler *AttemptHandler) Start(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := user.UserFromContext(r.Context())

	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	quizID := r.PathValue("quizID")

	details, created, err := handler.service.Start(r.Context(), currentUser.ID, quizID)

	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidAttemptID), errors.Is(err, quiz.ErrInvalidQuizID):
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrAttemptNotFound), errors.Is(err, quiz.ErrQuizNotFound):
			httpx.WriteError(w, http.StatusNotFound, err.Error())
		default:
			slog.Error("get attempt", "error", err)
			httpx.WriteError(w, http.StatusInternalServerError, "server error")
		}

		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}

	httpx.WriteJSON(w, status, toAttemptResponse(details))
}

func (handler *AttemptHandler) SaveAnswer(w http.ResponseWriter, r *http.Request) {

	var req SaveAnswerRequest

	currentUser, ok := user.UserFromContext(r.Context())

	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !httpx.ReadJSON(w, r, &req, httpx.MaxJSONBody) {
		return
	}

	attemptID := r.PathValue("attemptID")
	quizQuestionID := r.PathValue("quizQuestionID")
	attemptAnswer, err := handler.service.SaveAnswer(r.Context(), currentUser.ID, attemptID, quizQuestionID, req)

	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidAttemptID),
			errors.Is(err, quiz.ErrWrongQuizQuestionID),
			errors.Is(err, ErrInvalidOptionID):
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrAttemptNotFound), errors.Is(err, ErrAnswerTargetNotFound):
			httpx.WriteError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrAttemptClosed):
			httpx.WriteError(w, http.StatusConflict, err.Error())
		default:
			slog.Error("save answer", "error", err)
			httpx.WriteError(w, http.StatusInternalServerError, "server error")
		}
		return
	}

	response := SaveAnswerResponse{
		QuizQuestionID: attemptAnswer.QuizQuestionID,
		OptionID:       attemptAnswer.QuizAnswerOptionID,
		UpdatedAt:      attemptAnswer.UpdatedAt,
	}

	httpx.WriteJSON(w, http.StatusOK, response)
}

func (handler *AttemptHandler) Finish(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := user.UserFromContext(r.Context())

	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	attemptID := r.PathValue("attemptID")

	attemptResult, err := handler.service.Finish(r.Context(), currentUser.ID, attemptID)

	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidAttemptID):
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrAttemptNotFound):
			httpx.WriteError(w, http.StatusNotFound, err.Error())
		default:
			slog.Error("finish attempt", "error", err)
			httpx.WriteError(w, http.StatusInternalServerError, "server error")
		}
		return
	}

	response := toFinishResponse(attemptResult)

	httpx.WriteJSON(w, http.StatusOK, response)

}

func (handler *AttemptHandler) GetResult(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := user.UserFromContext(r.Context())

	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	attemptID := r.PathValue("attemptID")

	attemptResult, err := handler.service.GetResult(r.Context(), currentUser.ID, attemptID)

	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidAttemptID):
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrAttemptNotFound):
			httpx.WriteError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrAttemptInProgress):
			httpx.WriteError(w, http.StatusConflict, err.Error())
		default:
			slog.Error("get result", "error", err)
			httpx.WriteError(w, http.StatusInternalServerError, "server error")
		}
		return
	}

	response := toFinishResponse(attemptResult)

	httpx.WriteJSON(w, http.StatusOK, response)
}

func (handler *AttemptHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := user.UserFromContext(r.Context())

	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	quizID := r.PathValue("quizID")

	limit := 20
	offset := 0

	query := r.URL.Query()

	if value := query.Get("limit"); value != "" {
		parsedLimit, err := strconv.Atoi(value)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "limit must be an integer")
			return
		}

		limit = parsedLimit
	}

	if value := query.Get("offset"); value != "" {
		parsedOffset, err := strconv.Atoi(value)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "offset must be an integer")
			return
		}

		offset = parsedOffset
	}

	attemptSummaries, err := handler.service.GetHistory(r.Context(), currentUser.ID, quizID, limit, offset)

	if err != nil {
		switch {
		case errors.Is(err, quiz.ErrInvalidQuizID),
			errors.Is(err, quiz.ErrInvalidLimit),
			errors.Is(err, quiz.ErrInvalidOffset):
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, quiz.ErrQuizNotFound):
			httpx.WriteError(w, http.StatusNotFound, err.Error())
		default:
			slog.Error("get attempt history", "error", err)
			httpx.WriteError(w, http.StatusInternalServerError, "server error")
		}
		return
	}

	response := make([]AttemptSummaryResponse, 0, len(attemptSummaries))
	for _, attemptSummary := range attemptSummaries {
		newAttemptSummaryResponse := AttemptSummaryResponse{
			AttemptID:      attemptSummary.AttemptID,
			QuizID:         attemptSummary.QuizID,
			Status:         attemptSummary.Status,
			StartedAt:      attemptSummary.StartedAt,
			DeadlineAt:     attemptSummary.DeadlineAt,
			FinishedAt:     attemptSummary.FinishedAt,
			CorrectAnswers: attemptSummary.CorrectAnswers,
			GradableCount:  attemptSummary.GradableCount,
			Percentage:     attemptSummary.Percentage,
		}

		response = append(response, newAttemptSummaryResponse)
	}

	httpx.WriteJSON(w, http.StatusOK, response)
}

func toFinishResponse(result *AttemptResult) FinishResponse {
	convertQuestions := func(
		questions []ResultQuestion,
	) []ResultQuestionResponse {
		response := make(
			[]ResultQuestionResponse,
			0,
			len(questions),
		)

		for _, question := range questions {
			options := make(
				[]ResultOptionResponse,
				0,
				len(question.Options),
			)

			for _, option := range question.Options {
				options = append(
					options,
					ResultOptionResponse{
						ID:         option.ID,
						Text:       option.Text,
						Position:   option.Position,
						IsCorrect:  option.IsCorrect,
						IsSelected: option.IsSelected,
					},
				)
			}

			response = append(
				response,
				ResultQuestionResponse{
					ID:               question.ID,
					Text:             question.Text,
					Position:         question.Position,
					SelectedOptionID: question.SelectedOptionID,
					IsCorrect:        question.IsCorrect,
					Options:          options,
				},
			)
		}

		return response
	}

	return FinishResponse{
		AttemptID:           result.AttemptID,
		QuizID:              result.QuizID,
		Status:              result.Status,
		TotalQuestions:      result.TotalQuestions,
		AnsweredQuestions:   result.AnsweredQuestions,
		CorrectAnswers:      result.CorrectAnswers,
		WrongAnswers:        result.WrongAnswers,
		UnansweredQuestions: result.UnansweredQuestions,
		Percentage:          result.Percentage,
		StartedAt:           result.StartedAt,
		FinishedAt:          result.FinishedAt,

		CorrectAnswersList: convertQuestions(
			result.CorrectAnswersList,
		),

		WrongAnswersList: convertQuestions(
			result.WrongAnswersList,
		),

		UnansweredQuestionsList: convertQuestions(
			result.UnansweredQuestionsList,
		),
	}
}

func toAttemptResponse(details *AttemptDetails) AttemptResponse {
	questions := make([]AttemptQuestionResponse, 0, len(details.Questions))
	for _, question := range details.Questions {
		options := make([]AttemptOptionResponse, 0, len(question.Options))
		for _, option := range question.Options {
			options = append(options, AttemptOptionResponse{
				ID:       option.ID,
				Text:     option.Text,
				Position: option.Position,
			})
		}

		questions = append(questions, AttemptQuestionResponse{
			ID:       question.ID,
			Text:     question.Text,
			Position: question.Position,
			Options:  options,
		})
	}

	answers := make([]SavedAnswer, 0, len(details.Answers))
	for _, answer := range details.Answers {
		answers = append(answers, SavedAnswer{
			ID:                 answer.ID,
			QuizQuestionID:     answer.QuizQuestionID,
			QuizAnswerOptionID: answer.QuizAnswerOptionID,
		})
	}

	return AttemptResponse{
		ID:         details.ID,
		QuizID:     details.QuizID,
		Status:     details.Status,
		StartedAt:  details.StartedAt,
		DeadlineAt: details.DeadlineAt,
		FinishedAt: details.FinishedAt,
		ServerTime: details.ServerTime,
		Questions:  questions,
		Answers:    answers,
	}
}
