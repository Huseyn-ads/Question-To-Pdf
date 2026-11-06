package user

import (
	"errors"
	"log/slog"
	"net/http"
	"qtp/internal/httpx"
	"qtp/internal/session"
	"time"
)

type UserHandler struct {
	service      *UserService
	cookieSecure bool
	google       *googleLogin
}

func NewUserHandler(us *UserService, cookieSecure bool) *UserHandler {
	return &UserHandler{
		service:      us,
		cookieSecure: cookieSecure,
	}
}

func (handler *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	if !httpx.ReadJSON(w, r, &req, httpx.MaxAuthBody) {
		return
	}

	result, err := handler.service.Login(r.Context(), req)

	if err != nil {
		if errors.Is(err, ErrEmailNotVerified) {
			httpx.WriteError(w, http.StatusForbidden, ErrEmailNotVerified.Error())
			return
		}
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.WriteError(w, http.StatusUnauthorized, ErrInvalidCredentials.Error())
			return
		}
		slog.Error("login user", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "server error")
		return
	}

	handler.setSessionCookie(w, result)

	httpx.WriteJSON(w, http.StatusOK, UserResponse{
		ID:              result.User.ID,
		Email:           result.User.Email,
		Name:            result.User.Name,
		CreatedAt:       result.User.CreatedAt,
		EmailVerifiedAt: result.User.EmailVerifiedAt,
	})

}

func (handler *UserHandler) setSessionCookie(w http.ResponseWriter, result *LoginResult) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    result.Token,
		Path:     "/",
		Expires:  result.ExpiresAt,
		HttpOnly: true,
		Secure:   handler.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (handler *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	if !httpx.ReadJSON(w, r, &req, httpx.MaxAuthBody) {
		return
	}

	result, err := handler.service.Register(r.Context(), req)

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
			httpx.WriteError(w, http.StatusInternalServerError, "server error")
		}
		return
	}

	response := RegisterResponse{
		UserResponse: UserResponse{
			ID:              result.User.ID,
			Email:           result.User.Email,
			Name:            result.User.Name,
			CreatedAt:       result.User.CreatedAt,
			EmailVerifiedAt: result.User.EmailVerifiedAt,
		},
		VerificationRequired:  true,
		VerificationEmailSent: result.VerificationEmailSent,
	}

	if err := httpx.WriteJSON(
		w,
		http.StatusCreated,
		response,
	); err != nil {
		slog.Error("write register response", "error", err)
	}

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
		ID:              currentUser.ID,
		Email:           currentUser.Email,
		Name:            currentUser.Name,
		CreatedAt:       currentUser.CreatedAt,
		EmailVerifiedAt: currentUser.EmailVerifiedAt,
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
		Secure:   handler.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	w.WriteHeader(http.StatusNoContent)
}

func (handler *UserHandler) CSRF(w http.ResponseWriter, r *http.Request) {
	currentSession, ok := r.Context().Value(authenticatedSessionKey).(*session.Session)
	if !ok {
		_ = httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	_ = httpx.WriteJSON(w, http.StatusOK, struct {
		Token string `json:"csrf_token"`
	}{Token: currentSession.CSRFToken})
}

func (handler *UserHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req VerifyEmailRequest
	if !httpx.ReadJSON(w, r, &req, httpx.MaxAuthBody) {
		return
	}
	if err := handler.service.VerifyEmail(r.Context(), req.Token); err != nil {
		if errors.Is(err, ErrInvalidVerificationToken) {
			httpx.WriteError(w, http.StatusBadRequest, ErrInvalidVerificationToken.Error())
			return
		}
		slog.Error("verify email failed")
		httpx.WriteError(w, http.StatusInternalServerError, "server error")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, MessageResponse{Message: "email verified; you can now log in"})
}

func (handler *UserHandler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	var req ResendVerificationRequest
	if !httpx.ReadJSON(w, r, &req, httpx.MaxAuthBody) {
		return
	}
	if err := handler.service.ResendVerification(r.Context(), req.Email); err != nil {
		if errors.Is(err, ErrInvalidEmail) {
			httpx.WriteError(w, http.StatusBadRequest, ErrInvalidEmail.Error())
			return
		}
		slog.Error("resend verification failed")
		httpx.WriteError(w, http.StatusInternalServerError, "server error")
		return
	}
	// Same response for absent, verified, throttled and pending accounts.
	httpx.WriteJSON(w, http.StatusAccepted, MessageResponse{
		Message: "if verification is needed, an email will be sent; please wait before trying again",
	})
}
