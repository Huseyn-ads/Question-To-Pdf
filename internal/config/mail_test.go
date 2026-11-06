package config

import "testing"

func TestMailConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name       string
		production bool
		values     map[string]string
		valid      bool
	}{
		{"local defaults", false, nil, true},
		{"production missing host and sender", true, nil, false},
		{"gmail STARTTLS", true, map[string]string{
			"SMTP_HOST": "smtp.gmail.com", "SMTP_PORT": "587", "SMTP_USER": "sender@example.com",
			"SMTP_PASSWORD": "test-only-not-a-real-password", "SMTP_FROM": "sender@example.com", "SMTP_TLS": "starttls",
		}, true},
		{"implicit TLS", true, map[string]string{"SMTP_HOST": "smtp.example.test", "SMTP_PORT": "465", "SMTP_FROM": "sender@example.test", "SMTP_TLS": "tls"}, true},
		{"production plaintext", true, map[string]string{"SMTP_HOST": "mailpit", "SMTP_FROM": "sender@example.test", "SMTP_TLS": "none"}, false},
		{"plaintext auth", false, map[string]string{"SMTP_USER": "sender@example.test", "SMTP_PASSWORD": "test"}, false},
		{"missing auth password", false, map[string]string{"SMTP_USER": "sender@example.test", "SMTP_TLS": "starttls"}, false},
		{"header injection", false, map[string]string{"SMTP_FROM": "sender@example.test\r\nBcc: victim@example.test"}, false},
		{"unsupported TLS mode", false, map[string]string{"SMTP_TLS": "auto"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, key := range []string{"SMTP_HOST", "SMTP_PORT", "SMTP_USER", "SMTP_PASSWORD", "SMTP_FROM", "SMTP_TLS"} {
				t.Setenv(key, "")
			}
			for key, value := range tc.values {
				t.Setenv(key, value)
			}
			_, err := loadMail(tc.production)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v, err=%v", tc.valid, err)
			}
		})
	}
}
