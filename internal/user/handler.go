package user

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

type UserHandler struct {
	service *UserService
}

func NewUserHandler(us *UserService) *UserHandler {
	return &UserHandler{
		service: us,
	}
}

func (handler *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		http.Error(w, "Invalid JSON  body", http.StatusBadRequest)
		return
	}

	createdUser, err := handler.service.Register(r.Context(), req)

	if err != nil {
		switch {
		case errors.Is(err, ErrNameRequired),
			errors.Is(err, ErrEmailRequired),
			errors.Is(err, ErrInvalidEmail),
			errors.Is(err, ErrPasswordRequired),
			errors.Is(err, ErrPasswordTooLong),
			errors.Is(err, ErrPasswordTooShort):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrEmailAlreadyExists):
			http.Error(w, ErrEmailAlreadyExists.Error(), http.StatusConflict)
		default:
			slog.Error("register user", "error", err)
			http.Error(w, "Internal server errror", http.StatusInternalServerError)
		}
		return
	}

	response := UserResponse{
		ID:        createdUser.ID,
		Email:     createdUser.Email,
		Name:      createdUser.Name,
		CreatedAt: createdUser.CreatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)

}
