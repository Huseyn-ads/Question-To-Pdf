package config

import "testing"

func TestSecurityEnvironment(t *testing.T) {
	for _, tc := range []struct {
		env, origin   string
		valid, secure bool
	}{
		{"development", "", true, false},
		{"production", "", false, false},
		{"production", "http://quiz.test", false, false},
		{"production", "https://quiz.test", true, true},
		{"production", "https://quiz.test/", true, true},
		{"production", "https://quiz.test/path", false, false},
		{"production", "https://quiz.test?x=1", false, false},
		{"production", "https://*.quiz.test", false, false},
		{"production", "https://user:password@quiz.test", false, false},
	} {
		t.Setenv("APP_ENV", tc.env)
		t.Setenv("PUBLIC_ORIGIN", tc.origin)
		cfg, err := loadSecurity(8080)
		if (err == nil) != tc.valid {
			t.Fatalf("%+v: %v", tc, err)
		}
		if tc.valid && cfg.CookieSecure != tc.secure {
			t.Fatal("wrong cookie security")
		}
	}
}
