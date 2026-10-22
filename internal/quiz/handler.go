package quiz

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"qtp/internal/httpx"
	"qtp/internal/user"
	"strconv"
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

func (handler *QuizHandler) FindQuizByOwnID(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := user.UserFromContext(r.Context())

	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	quizID := r.PathValue("quizID")

	foundedQuiz, err := handler.service.FindByOwnedID(r.Context(), currentUser.ID, quizID)

	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidQuizID):
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrQuizNotFound):
			httpx.WriteError(w, http.StatusNotFound, err.Error())
		default:
			slog.Error("find quiz", "error", err)
			httpx.WriteError(w, http.StatusInternalServerError, "server error")
		}
		return
	}

	quizQuestionResponse := make([]QuizQuestionResponse, 0, len(foundedQuiz.Questions))
	response := toResponseQuiz(quizQuestionResponse, foundedQuiz)

	httpx.WriteJSON(w, http.StatusOK, response)
}

func (handler *QuizHandler) List(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := user.UserFromContext(r.Context())

	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
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

	quizzes, err := handler.service.ListByUserID(r.Context(), currentUser.ID, limit, offset)

	if err != nil {
		if errors.Is(err, ErrInvalidLimit) {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		if errors.Is(err, ErrInvalidOffset) {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		slog.Error("list quizzes by user id", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "server error")
		return
	}

	response := QuizListResponse{
		Items:  quizzes,
		Limit:  limit,
		Offset: offset,
	}

	httpx.WriteJSON(w, http.StatusOK, response)
}

func (handler *QuizHandler) Delete(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := user.UserFromContext(r.Context())

	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	quizID := r.PathValue("quizID")

	err := handler.service.Delete(r.Context(), currentUser.ID, quizID)

	if err != nil {
		switch {
		case errors.Is(err, ErrQuizNotFound):
			httpx.WriteError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrInvalidQuizID):
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
		default:
			slog.Error("delete quiz", "error", err)
			httpx.WriteError(w, http.StatusInternalServerError, "server error")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
