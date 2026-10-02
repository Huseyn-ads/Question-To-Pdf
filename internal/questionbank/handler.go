package questionbank

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"qtp/internal/httpx"
	"qtp/internal/user"
)

type QuestionBankHandler struct {
	service *QuestionBankService
}

func NewQuestionBankHandler(service *QuestionBankService) *QuestionBankHandler {
	return &QuestionBankHandler{
		service: service,
	}
}

func (handler *QuestionBankHandler) List(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := user.UserFromContext(r.Context())

	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	banks, err := handler.service.ListByUserID(r.Context(), currentUser.ID)

	if err != nil {
		slog.Error("list question banks", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "server error")
		return
	}

	responses := make([]Response, 0, len(banks))

	for _, bank := range banks {
		responses = append(responses, Response{
			ID:          bank.ID,
			Title:       bank.Title,
			Description: bank.Description,
			CreatedAt:   bank.CreatedAt,
			UpdatedAt:   bank.UpdatedAt,
		})
	}

	httpx.WriteJSON(w, http.StatusOK, responses)

}

func (handler *QuestionBankHandler) Find(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := user.UserFromContext(r.Context())

	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	bankID := r.PathValue("bankID")
	questionBank, err := handler.service.FindByID(r.Context(), bankID, currentUser.ID)

	if err != nil {
		if errors.Is(err, ErrQuestionBankNotFound) {
			httpx.WriteError(
				w,
				http.StatusNotFound,
				ErrQuestionBankNotFound.Error(),
			)
			return
		}

		slog.Error("find question bank", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "server error")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, Response{
		ID:          questionBank.ID,
		Title:       questionBank.Title,
		Description: questionBank.Description,
		CreatedAt:   questionBank.CreatedAt,
		UpdatedAt:   questionBank.UpdatedAt,
	})
}

func (handler *QuestionBankHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	currentUser, ok := user.UserFromContext(r.Context())

	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	createdQuestionBank, err := handler.service.Create(r.Context(), currentUser.ID, req)

	if err != nil {
		switch {
		case errors.Is(err, ErrTitleRequired),
			errors.Is(err, ErrTitleTooLong):
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
		default:
			slog.Error("create question bank", "error", err)
			httpx.WriteError(w, http.StatusInternalServerError, "server error")
		}

		return
	}

	response := Response{
		ID:          createdQuestionBank.ID,
		Title:       createdQuestionBank.Title,
		Description: createdQuestionBank.Description,
		CreatedAt:   createdQuestionBank.CreatedAt,
		UpdatedAt:   createdQuestionBank.UpdatedAt,
	}

	httpx.WriteJSON(w, http.StatusCreated, response)

}

func (handler *QuestionBankHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var req UpdateRequest

	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	currentUser, ok := user.UserFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	bankID := r.PathValue("bankID")

	updatedQuestionBank, err := handler.service.Update(r.Context(), currentUser.ID, bankID, req)

	if err != nil {
		switch {
		case errors.Is(err, ErrNoFieldsToUpdate),
			errors.Is(err, ErrTitleRequired),
			errors.Is(err, ErrTitleTooLong):
			httpx.WriteError(w, http.StatusBadRequest, err.Error())

		case errors.Is(err, ErrQuestionBankNotFound):
			httpx.WriteError(
				w,
				http.StatusNotFound,
				ErrQuestionBankNotFound.Error(),
			)

		default:
			slog.Error("update question bank", "error", err)
			httpx.WriteError(
				w,
				http.StatusInternalServerError,
				"server error",
			)
		}

		return
	}

	httpx.WriteJSON(w, http.StatusOK, Response{
		ID:          updatedQuestionBank.ID,
		Title:       updatedQuestionBank.Title,
		Description: updatedQuestionBank.Description,
		CreatedAt:   updatedQuestionBank.CreatedAt,
		UpdatedAt:   updatedQuestionBank.UpdatedAt,
	})
}

func (handler *QuestionBankHandler) Delete(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := user.UserFromContext(r.Context())

	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	bankID := r.PathValue("bankID")

	err := handler.service.Delete(r.Context(), bankID, currentUser.ID)

	if err != nil {
		if errors.Is(err, ErrQuestionBankNotFound) {
			httpx.WriteError(
				w,
				http.StatusNotFound,
				ErrQuestionBankNotFound.Error(),
			)
			return
		}
		slog.Error("delete question bank", "error", err)
		httpx.WriteError(
			w,
			http.StatusInternalServerError,
			"server error",
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)

}
