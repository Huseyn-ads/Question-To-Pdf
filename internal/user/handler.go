package user

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"qtp/internal/httpx"
	"time"
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
		Name:     sessionCookieName,
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

func (handler *UserHandler) Me(
	w http.ResponseWriter,
	r *http.Request,
) {
	currentUser, ok := UserFromContext(r.Context())
	if !ok {
		if err := httpx.WriteError(
			w,
			http.StatusUnauthorized,
			ErrUnauthorized.Error(),
		); err != nil {
			slog.Error(
				"write unauthorized response",
				"error",
				err,
			)
		}

		return
	}

	response := UserResponse{
		ID:        currentUser.ID,
		Email:     currentUser.Email,
		Name:      currentUser.Name,
		CreatedAt: currentUser.CreatedAt,
	}

	if err := httpx.WriteJSON(
		w,
		http.StatusOK,
		response,
	); err != nil {
		slog.Error(
			"write current user response",
			"error",
			err,
		)
	}
}

func (handler *UserHandler) Logout(
	w http.ResponseWriter,
	r *http.Request,
) {
	cookie, err := r.Cookie(sessionCookieName)

	if err == nil {
		err = handler.service.Logout(
			r.Context(),
			cookie.Value,
		)
		if err != nil {
			slog.Error("logout user", "error", err)

			if writeErr := httpx.WriteError(
				w,
				http.StatusInternalServerError,
				"internal server error",
			); writeErr != nil {
				slog.Error(
					"write logout error response",
					"error",
					writeErr,
				)
			}

			return
		}
	} else if !errors.Is(err, http.ErrNoCookie) {
		slog.Error("read session cookie", "error", err)

		if writeErr := httpx.WriteError(
			w,
			http.StatusInternalServerError,
			"internal server error",
		); writeErr != nil {
			slog.Error(
				"write cookie error response",
				"error",
				writeErr,
			)
		}

		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(1, 0).UTC(),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	w.WriteHeader(http.StatusNoContent)
}
