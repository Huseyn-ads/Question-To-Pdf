package config

import (
	"errors"
	"net"
	"net/url"
	"os"
	"strings"
)

const GoogleCallbackPath = "/v1/auth/google/callback"

type GoogleOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

func loadGoogleOAuth(security SecurityConfig) (GoogleOAuthConfig, error) {
	clientID := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET"))
	if clientID == "" && clientSecret == "" {
		return GoogleOAuthConfig{}, nil
	}
	if clientID == "" || clientSecret == "" {
		return GoogleOAuthConfig{}, errors.New("set both GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET to enable Google login")
	}
	if !security.CookieSecure {
		origin, err := url.Parse(security.PublicOrigin)
		if err != nil {
			return GoogleOAuthConfig{}, errors.New("invalid PUBLIC_ORIGIN for Google login")
		}
		host := origin.Hostname()
		if !strings.EqualFold(host, "localhost") && !net.ParseIP(host).IsLoopback() {
			return GoogleOAuthConfig{}, errors.New("Google login requires HTTPS or a loopback PUBLIC_ORIGIN")
		}
	}
	redirectURL := security.PublicOrigin + GoogleCallbackPath
	if value := strings.TrimSpace(os.Getenv("GOOGLE_REDIRECT_URL")); value != "" && value != redirectURL {
		return GoogleOAuthConfig{}, errors.New("GOOGLE_REDIRECT_URL must be PUBLIC_ORIGIN followed by " + GoogleCallbackPath)
	}
	return GoogleOAuthConfig{ClientID: clientID, ClientSecret: clientSecret, RedirectURL: redirectURL}, nil
}
