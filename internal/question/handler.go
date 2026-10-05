package question

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"qtp/internal/httpx"
	"qtp/internal/questionbank"
	"qtp/internal/user"
)

type QuestionHandler struct {
	service *QuestionService
}

func NewQuestionHandler(service *QuestionService) *QuestionHandler {
	return &QuestionHandler{
		service: service,
	}
}

func (handler *QuestionHandler) FindByID(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := user.UserFromContext(r.Context())

	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	bankID := r.PathValue("bankID")
	questionID := r.PathValue("questionID")

	foundQuestion, err := handler.service.FindByID(r.Context(), currentUser.ID, bankID, questionID)

	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidQuestionID),
			errors.Is(err, questionbank.ErrInvalidQuestionBankID):
			httpx.WriteError(
				w,
				http.StatusBadRequest,
				err.Error(),
			)

		case errors.Is(err, ErrQuestionNotFound):
			httpx.WriteError(
				w,
				http.StatusNotFound,
				ErrQuestionNotFound.Error(),
			)

		default:
			slog.Error("find question", "error", err)
			httpx.WriteError(
				w,
				http.StatusInternalServerError,
				"server error",
			)
		}

		return
	}

	optionResponses := make([]OptionResponse, 0, len(foundQuestion.Options))
	for _, option := range foundQuestion.Options {
		optionResponses = append(optionResponses, OptionResponse{
			ID:        option.ID,
			Text:      option.Text,
			Position:  option.Position,
			IsCorrect: option.IsCorrect,
		})
	}

	response := Response{
		ID:             foundQuestion.ID,
		QuestionBankID: foundQuestion.QuestionBankID,
		Text:           foundQuestion.Text,
		Options:        optionResponses,
		CreatedAt:      foundQuestion.CreatedAt,
		UpdatedAt:      foundQuestion.UpdatedAt,
	}

	httpx.WriteJSON(w, http.StatusOK, response)

}

func (handler *QuestionHandler) Create(w http.ResponseWriter, r *http.Request) {

	var req CreateRequest

	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid JSON json")
		return
	}

	currentUser, ok := user.UserFromContext(r.Context())

	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	bankID := r.PathValue("bankID")

	createdQuestion, err := handler.service.Create(r.Context(), bankID, currentUser.ID, req)

	if err != nil {
		switch {
		case errors.Is(err, ErrTextRequired),
			errors.Is(err, ErrOptionTextRequired),
			errors.Is(err, ErrDuplicateOptions),
			errors.Is(err, ErrNotEnoughOptions),
			errors.Is(err, ErrExactlyOneCorrectOption):
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, questionbank.ErrQuestionBankNotFound):
			httpx.WriteError(
				w,
				http.StatusNotFound,
				questionbank.ErrQuestionBankNotFound.Error(),
			)
		case errors.Is(err, questionbank.ErrInvalidQuestionBankID):
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
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

	optionResponses := make(
		[]OptionResponse,
		0,
		len(createdQuestion.Options),
	)

	for _, option := range createdQuestion.Options {
		optionResponses = append(optionResponses, OptionResponse{
			ID:        option.ID,
			Text:      option.Text,
			Position:  option.Position,
			IsCorrect: option.IsCorrect,
		})
	}

	httpx.WriteJSON(w, http.StatusCreated, Response{
		ID:             createdQuestion.ID,
		QuestionBankID: createdQuestion.QuestionBankID,
		Text:           createdQuestion.Text,
		Options:        optionResponses,
		CreatedAt:      createdQuestion.CreatedAt,
		UpdatedAt:      createdQuestion.UpdatedAt,
	})

}
