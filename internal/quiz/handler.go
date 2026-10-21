package quiz

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"qtp/internal/httpx"
	"qtp/internal/user"
)

type QuizHandler struct {
	service *QuizService
}

func NewQuizHandler(service *QuizService) *QuizHandler {
	return &QuizHandler{
		service: service,
	}
}

func (handler *QuizHandler) CreateQuiz(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	currentUser, ok := user.UserFromContext(r.Context())

	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	bankID := r.PathValue("bankID")

	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid JSON json")
		return
	}

	createdQuiz, err := handler.service.CreateQuiz(r.Context(), currentUser.ID, bankID, req)

	if err != nil {

		switch {
		case errors.Is(err, ErrInvalidQuizID),
			errors.Is(err, ErrInvalidQuestionCount),
			errors.Is(err, ErrNotEnoughtQuestions),
			errors.Is(err, ErrInvalidTimeLimiter),
			errors.Is(err, ErrWrongQuizQuestionID),
			errors.Is(err, ErrEmptyQuestions):
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrQuizNotFound):
			httpx.WriteError(w, http.StatusNotFound, err.Error())
		default:
			slog.Error("create question", "error", err)
			httpx.WriteError(
				w,
				http.StatusInternalServerError,
				"server error",
			)
		}
		return
	}

	quizResponse := QuizSummary{
		ID:             createdQuiz.ID,
		QuestionBankID: createdQuiz.QuestionBankID,
		QuestionCount:  createdQuiz.QuestionCount,
		CreatedAt:      createdQuiz.CreatedAt,
	}

	httpx.WriteJSON(w, http.StatusOK, quizResponse)

}
