package user

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"qtp/internal/httpx"
)

const sessionCookieName = "session"

type contextKey string

const authenticatedUserKey contextKey = "authenticated_user"

func (handler *UserHandler) RequireAuth(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(sessionCookieName)
			if err != nil {
				if !errors.Is(err, http.ErrNoCookie) {
					slog.Error(
						"read session cookie",
						"error",
						err,
					)
				}

				if writeErr := httpx.WriteError(
					w,
					http.StatusUnauthorized,
					ErrUnauthorized.Error(),
				); writeErr != nil {
					slog.Error(
						"write unauthorized response",
						"error",
						writeErr,
					)
				}

				return
			}

			currentUser, err := handler.service.CurrentUser(
				r.Context(),
				cookie.Value,
			)
			if err != nil {
				if errors.Is(err, ErrUnauthorized) {
					if writeErr := httpx.WriteError(
						w,
						http.StatusUnauthorized,
						ErrUnauthorized.Error(),
					); writeErr != nil {
						slog.Error(
							"write unauthorized response",
							"error",
							writeErr,
						)
					}

					return
				}

				slog.Error(
					"authenticate request",
					"error",
					err,
				)

				if writeErr := httpx.WriteError(
					w,
					http.StatusInternalServerError,
					"internal server error",
				); writeErr != nil {
					slog.Error(
						"write internal error response",
						"error",
						writeErr,
					)
				}

				return
			}

			ctx := context.WithValue(
				r.Context(),
				authenticatedUserKey,
				currentUser,
			)

			next.ServeHTTP(
				w,
				r.WithContext(ctx),
			)
		},
	)
}

func UserFromContext(
	ctx context.Context,
) (*User, bool) {
	currentUser, ok := ctx.Value(
		authenticatedUserKey,
	).(*User)

	return currentUser, ok
}
