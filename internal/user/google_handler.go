package user

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"qtp/internal/httpx"
)

const googleOAuthCookieName = "google_oauth"

type googleProfile struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
}

func (handler *UserHandler) SetGoogleOAuth(config *oauth2.Config) {
	handler.googleOAuth = config
}

func (handler *UserHandler) googleCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     googleOAuthCookieName,
		Value:    value,
		Path:     "/v1/auth/google/callback",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   handler.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	}
}

func (handler *UserHandler) GoogleStart(w http.ResponseWriter, r *http.Request) {
	if handler.googleOAuth == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "google oauth is not configured")
		return
	}

	state := oauth2.GenerateVerifier()
	verifier := oauth2.GenerateVerifier()

	http.SetCookie(w, handler.googleCookie(state+"."+verifier, 300))

	loginURL := handler.googleOAuth.AuthCodeURL(
		state,
		oauth2.S256ChallengeOption(verifier),
	)
	http.Redirect(w, r, loginURL, http.StatusFound)
}

func (handler *UserHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	if handler.googleOAuth == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "google oauth is not configured")
		return
	}

	cookie, err := r.Cookie(googleOAuthCookieName)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "start google sign-in again")
		return
	}

	query := r.URL.Query()
	state, verifier, ok := strings.Cut(cookie.Value, ".")
	if !ok || state == "" || verifier == "" || query.Get("state") != state {
		httpx.WriteError(w, http.StatusBadRequest, "invalid google oauth state")
		return
	}

	http.SetCookie(w, handler.googleCookie("", -1))

	code := query.Get("code")
	if query.Get("error") != "" || code == "" {
		httpx.WriteError(w, http.StatusBadRequest, "google sign-in was not completed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	token, err := handler.googleOAuth.Exchange(
		ctx,
		code,
		oauth2.VerifierOption(verifier),
	)
	if err != nil {
		httpx.WriteError(w, http.StatusBadGateway, "google token exchange failed")
		return
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"https://openidconnect.googleapis.com/v1/userinfo",
		nil,
	)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "could not create profile request")
		return
	}

	response, err := handler.googleOAuth.Client(ctx, token).Do(req)
	if err != nil {
		httpx.WriteError(w, http.StatusBadGateway, "google profile request failed")
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		httpx.WriteError(w, http.StatusBadGateway, "google rejected the profile request")
		return
	}

	var profile googleProfile
	if err := json.NewDecoder(io.LimitReader(response.Body, 64*1024)).Decode(&profile); err != nil {
		httpx.WriteError(w, http.StatusBadGateway, "invalid google profile")
		return
	}
	if profile.Sub == "" || profile.Email == "" {
		httpx.WriteError(w, http.StatusBadGateway, "incomplete google profile")
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	_ = httpx.WriteJSON(w, http.StatusOK, profile)
}
