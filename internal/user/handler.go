package user

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"qtp/internal/httpx"
)

type UserHandler struct {
	service *UserService
}

func NewUserHandler(us *UserService) *UserHandler {
	return &UserHandler{
		service: us,
	}
}

func (handler *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	result, err := handler.service.Login(r.Context(), req)

	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.WriteError(w, http.StatusUnauthorized, ErrInvalidCredentials.Error())
			return
		}
		slog.Error("login user", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "server error")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    result.Token,
		Path:     "/",
		Expires:  result.ExpiresAt,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	httpx.WriteJSON(w, http.StatusOK, UserResponse{
		ID:        result.User.ID,
		Email:     result.User.Email,
		Name:      result.User.Name,
		CreatedAt: result.User.CreatedAt,
	})

}

func (handler *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid JSON body")
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
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrEmailAlreadyExists):
			httpx.WriteError(w, http.StatusConflict, ErrEmailAlreadyExists.Error())
		default:
			slog.Error("register user", "error", err)
			httpx.WriteError(w, http.StatusInternalServerError, "Internal server errror")
		}
		return
	}

	response := UserResponse{
		ID:        createdUser.ID,
		Email:     createdUser.Email,
		Name:      createdUser.Name,
		CreatedAt: createdUser.CreatedAt,
	}

	if err := httpx.WriteJSON(
		w,
		http.StatusCreated,
		response,
	); err != nil {
		slog.Error("write register response", "error", err)
	}

	httpx.WriteJSON(w, http.StatusCreated, response)
}
