package user

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"qtp/internal/httpx"
)

const (
	googleIssuer     = "https://accounts.google.com"
	googleKeysURL    = "https://www.googleapis.com/oauth2/v3/certs"
	googleStateTTL   = 5 * time.Minute
	googleCookieName = "qtp_google_oauth"
)

type googleLogin struct {
	oauth    oauth2.Config
	verifier *oidc.IDTokenVerifier
	client   *http.Client
}

type googleClaims struct {
	Email           string `json:"email"`
	EmailVerified   bool   `json:"email_verified"`
	Name            string `json:"name"`
	Nonce           string `json:"nonce"`
	AuthorizedParty string `json:"azp"`
}

func (handler *UserHandler) ConfigureGoogleLogin(clientID, clientSecret, redirectURL string) {
	client := &http.Client{
		Timeout: 8 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	keys := oidc.NewRemoteKeySet(oidc.ClientContext(context.Background(), client), googleKeysURL)
	handler.google = &googleLogin{
		oauth: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     google.Endpoint,
			Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
		},
		verifier: oidc.NewVerifier(googleIssuer, keys, &oidc.Config{ClientID: clientID}),
		client:   client,
	}
}

func (handler *UserHandler) GoogleStart(w http.ResponseWriter, r *http.Request) {
	if !handler.allowGoogleLogin(w, r) {
		return
	}
	state := oauth2.GenerateVerifier()
	verifier := oauth2.GenerateVerifier()
	nonce := oauth2.GenerateVerifier()

	cookie := handler.googleStateCookie()
	cookie.Value = handler.google.encodeState(state, verifier, nonce, time.Now().Add(googleStateTTL))
	cookie.MaxAge = int(googleStateTTL / time.Second)
	http.SetCookie(w, cookie)

	location := handler.google.oauth.AuthCodeURL(state,
		oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("nonce", nonce),
	)
	http.Redirect(w, r, location, http.StatusFound)
}

func (handler *UserHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	if !handler.allowGoogleLogin(w, r) {
		return
	}
	query := r.URL.Query()
	cookie := handler.googleStateCookie()
	savedCookie, err := r.Cookie(cookie.Name)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Google sign-in expired; start again")
		return
	}
	verifier, nonce, ok := handler.google.decodeState(savedCookie.Value, query.Get("state"), time.Now())
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "Google sign-in expired or invalid; start again")
		return
	}
	cookie.MaxAge = -1
	cookie.Expires = time.Unix(1, 0).UTC()
	http.SetCookie(w, cookie)

	code := query.Get("code")
	if query.Get("error") != "" || code == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Google sign-in was canceled or incomplete")
		return
	}
	ctx, cancel := context.WithTimeout(oidc.ClientContext(r.Context(), handler.google.client), 12*time.Second)
	defer cancel()
	token, err := handler.google.oauth.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		var providerErr *oauth2.RetrieveError
		if errors.As(err, &providerErr) && providerErr.ErrorCode == "invalid_grant" {
			httpx.WriteError(w, http.StatusBadRequest, "Google sign-in code expired or invalid; start again")
			return
		}
		slog.Warn("Google sign-in code exchange failed")
		httpx.WriteError(w, http.StatusBadGateway, "Google sign-in failed; try again")
		return
	}

	rawIDToken, _ := token.Extra("id_token").(string)
	idToken, err := handler.google.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		slog.Warn("Google ID token verification failed")
		httpx.WriteError(w, http.StatusBadGateway, "could not verify Google sign-in; start again")
		return
	}
	var claims googleClaims
	if err := idToken.Claims(&claims); err != nil || idToken.Subject == "" || len(idToken.Subject) > 255 {
		httpx.WriteError(w, http.StatusBadGateway, "invalid Google account profile")
		return
	}
	if subtle.ConstantTimeCompare([]byte(claims.Nonce), []byte(nonce)) != 1 {
		httpx.WriteError(w, http.StatusBadRequest, "Google sign-in does not match this browser; start again")
		return
	}
	if claims.AuthorizedParty != handler.google.oauth.ClientID &&
		(claims.AuthorizedParty != "" || len(idToken.Audience) > 1) {
		httpx.WriteError(w, http.StatusBadGateway, "invalid Google account profile")
		return
	}

	result, err := handler.service.loginWithGoogle(r.Context(), idToken.Subject, claims)
	if err != nil {
		switch {
		case errors.Is(err, ErrGoogleEmailRequired), errors.Is(err, ErrEmailNotVerified):
			httpx.WriteError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, ErrEmailAlreadyExists):
			httpx.WriteError(w, http.StatusConflict, "this email is already registered; use your existing sign-in method")
		default:
			slog.Error("create application session for Google sign-in failed")
			httpx.WriteError(w, http.StatusInternalServerError, "server error")
		}
		return
	}
	handler.setSessionCookie(w, result)
	http.Redirect(w, r, "/v1/auth/me", http.StatusSeeOther)
}

func (handler *UserHandler) allowGoogleLogin(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return false
	}
	if handler.google == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "Google sign-in is not configured")
		return false
	}
	return true
}

func (handler *UserHandler) googleStateCookie() *http.Cookie {
	name := googleCookieName
	if handler.cookieSecure {
		name = "__Host-" + name
	}
	return &http.Cookie{
		Name:     name,
		Path:     "/",
		HttpOnly: true,
		Secure:   handler.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	}
}

func (g *googleLogin) encodeState(state, verifier, nonce string, expiresAt time.Time) string {
	payload := strings.Join([]string{state, verifier, nonce, strconv.FormatInt(expiresAt.Unix(), 10)}, ".")
	return payload + "." + g.signState(payload)
}

func (g *googleLogin) decodeState(value, expectedState string, now time.Time) (verifier, nonce string, ok bool) {
	separator := strings.LastIndexByte(value, '.')
	if separator < 0 || expectedState == "" {
		return "", "", false
	}
	payload, signature := value[:separator], value[separator+1:]
	if !hmac.Equal([]byte(signature), []byte(g.signState(payload))) {
		return "", "", false
	}
	parts := strings.Split(payload, ".")
	if len(parts) != 4 || subtle.ConstantTimeCompare([]byte(parts[0]), []byte(expectedState)) != 1 {
		return "", "", false
	}
	expiresAt, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil || now.Unix() >= expiresAt {
		return "", "", false
	}
	return parts[1], parts[2], true
}

func (g *googleLogin) signState(payload string) string {
	mac := hmac.New(sha256.New, []byte(g.oauth.ClientSecret))
	mac.Write([]byte("qtp-google-oauth-state\n"))
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
