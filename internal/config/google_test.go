package config

import "testing"

func TestGoogleOAuthEnvironment(t *testing.T) {
	local := SecurityConfig{PublicOrigin: "http://localhost:8080"}
	t.Setenv("GOOGLE_CLIENT_ID", "")
	t.Setenv("GOOGLE_CLIENT_SECRET", "")
	t.Setenv("GOOGLE_REDIRECT_URL", "")
	if cfg, err := loadGoogleOAuth(local); err != nil || cfg.ClientID != "" {
		t.Fatal("empty credentials should disable Google sign-in", err)
	}
	t.Setenv("GOOGLE_CLIENT_ID", "test.apps.googleusercontent.com")
	if _, err := loadGoogleOAuth(local); err == nil {
		t.Fatal("partial credentials were accepted")
	}
	t.Setenv("GOOGLE_CLIENT_SECRET", "test-secret")
	if cfg, err := loadGoogleOAuth(local); err != nil || cfg.RedirectURL != local.PublicOrigin+GoogleCallbackPath {
		t.Fatal("callback should default to the app origin", err)
	}
	t.Setenv("GOOGLE_REDIRECT_URL", "http://127.0.0.1:8080"+GoogleCallbackPath)
	if _, err := loadGoogleOAuth(local); err == nil {
		t.Fatal("callback with a different origin was accepted")
	}
	t.Setenv("GOOGLE_REDIRECT_URL", "")
	if _, err := loadGoogleOAuth(SecurityConfig{PublicOrigin: "http://quiz.test"}); err == nil {
		t.Fatal("public HTTP origin was accepted")
	}
}
