package attempt

import (
	"errors"
	"log/slog"
	"net/http"
	"qtp/internal/httpx"
	"qtp/internal/user"
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
	}

	quizID := r.PathValue("quizID")

	attempt, err := handler.service.Start(r.Context(), currentUser.ID, quizID)

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

	response := ResultResponse{
		ID:            attempt.ID,
		QuizID:        attempt.QuizID,
		Status:        attempt.Status,
		CorrectCount:  attempt.CorrectCount,
		GradableCount: attempt.GradableCount,
		StartedAt:     attempt.StartedAt,
		FinishedAt:    attempt.FinishedAt,
	}

	httpx.WriteJSON(w, http.StatusCreated, response)
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
