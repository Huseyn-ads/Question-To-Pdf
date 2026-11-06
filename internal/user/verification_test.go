package user

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"qtp/internal/httpx"
)

type captureMailer struct {
	mu      sync.Mutex
	to      string
	link    string
	expires time.Time
	calls   int
	err     error
}

func (m *captureMailer) SendVerification(_ context.Context, to, link string, expires time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.to, m.link, m.expires = to, link, expires
	m.calls++
	return m.err
}

func emailToken(t *testing.T, link string) string {
	t.Helper()
	u, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}
	if u.Host != "quiz.test" || u.Path != "/v1/auth/verify-email" || u.RawQuery != "" {
		t.Fatalf("untrusted or leaky verification URL: %s", link)
	}
	values, err := url.ParseQuery(u.Fragment)
	if err != nil {
		t.Fatal(err)
	}
	return values.Get("token")
}

func verificationApp(t *testing.T) (*memoryUsers, *memorySessions, *captureMailer, http.Handler) {
	t.Helper()
	users, sessions, mailer := &memoryUsers{}, &memorySessions{}, &captureMailer{}
	h := NewUserHandler(NewUserService(users, sessions, mailer, "https://quiz.test"), true)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/auth/register", h.Register)
	mux.HandleFunc("POST /v1/auth/login", h.Login)
	mux.HandleFunc("POST /v1/auth/verify-email", h.VerifyEmail)
	mux.HandleFunc("GET /v1/auth/verify-email", h.VerificationPage)
	mux.HandleFunc("POST /v1/auth/resend-verification", h.ResendVerification)
	mux.Handle("GET /v1/auth/me", h.RequireAuth(http.HandlerFunc(h.Me)))
	return users, sessions, mailer, httpx.TrustedOrigin("https://quiz.test", mux)
}

const registrationJSON = `{"email":"student@example.com","name":"Student","password":"correct-password"}`
const loginJSON = `{"email":"student@example.com","password":"correct-password"}`

func TestEmailVerificationHTTPLifecycle(t *testing.T) {
	users, sessions, mailer, app := verificationApp(t)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, jsonRequest("POST", "/v1/auth/register", registrationJSON))
	if w.Code != 201 || len(w.Result().Cookies()) != 0 || mailer.calls != 1 {
		t.Fatal(w.Code, w.Body.String(), mailer.calls)
	}
	var registered RegisterResponse
	if err := json.Unmarshal(w.Body.Bytes(), &registered); err != nil {
		t.Fatal(err)
	}
	if !registered.VerificationRequired || !registered.VerificationEmailSent || registered.EmailVerifiedAt != nil {
		t.Fatal("registration incorrectly claims verified email")
	}
	token := emailToken(t, mailer.link)
	hash, err := verificationTokenHash(token)
	if err != nil || !bytes.Equal(hash, users.token.TokenHash) {
		t.Fatal("verification token not hashed correctly")
	}
	if strings.Contains(w.Body.String(), token) || bytes.Equal([]byte(token), users.token.TokenHash) {
		t.Fatal("raw verification token disclosed or persisted")
	}
	w = httptest.NewRecorder()
	app.ServeHTTP(w, jsonRequest("POST", "/v1/auth/login", loginJSON))
	if w.Code != 403 || sessions.current != nil || len(w.Result().Cookies()) != 0 {
		t.Fatal("unverified account logged in")
	}
	w = httptest.NewRecorder()
	app.ServeHTTP(w, jsonRequest("POST", "/v1/auth/login", `{"email":"student@example.com","password":"wrong-password"}`))
	if w.Code != 401 {
		t.Fatal("verification disclosed before password check")
	}

	// Merely opening the email page never consumes the token.
	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest("GET", "https://quiz.test/v1/auth/verify-email", nil))
	if w.Code != 200 || users.user.EmailVerifiedAt != nil ||
		!strings.Contains(w.Header().Get("Content-Security-Policy"), "script-src 'nonce-") ||
		w.Header().Get("Referrer-Policy") != "no-referrer" ||
		strings.Contains(w.Body.String(), token) {
		t.Fatal("unsafe verification landing page")
	}
	payload, _ := json.Marshal(VerifyEmailRequest{Token: token})
	w = httptest.NewRecorder()
	forged := jsonRequest("POST", "/v1/auth/verify-email", string(payload))
	forged.Header.Set("Origin", "https://evil.test")
	app.ServeHTTP(w, forged)
	if w.Code != 403 || users.user.EmailVerifiedAt != nil {
		t.Fatal("Origin bypass")
	}

	w = httptest.NewRecorder()
	app.ServeHTTP(w, jsonRequest("POST", "/v1/auth/verify-email", string(payload)))
	if w.Code != 200 || users.user.EmailVerifiedAt == nil || users.token.UsedAt == nil ||
		len(w.Result().Cookies()) != 0 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	app.ServeHTTP(w, jsonRequest("POST", "/v1/auth/verify-email", string(payload)))
	if w.Code != 400 {
		t.Fatal("verification token reused")
	}
	w = httptest.NewRecorder()
	app.ServeHTTP(w, jsonRequest("POST", "/v1/auth/login", loginJSON))
	if w.Code != 200 || sessions.current == nil || len(w.Result().Cookies()) != 1 {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestResendInvalidatesOldLinkAndHasGenericResponse(t *testing.T) {
	users, _, mailer, app := verificationApp(t)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, jsonRequest("POST", "/v1/auth/register", registrationJSON))
	old := emailToken(t, mailer.link)
	// An immediate resend is quietly limited; it neither sends nor invalidates.
	w = httptest.NewRecorder()
	app.ServeHTTP(w, jsonRequest("POST", "/v1/auth/resend-verification", `{"email":"student@example.com"}`))
	if w.Code != 202 || mailer.calls != 1 {
		t.Fatal("resend cooldown bypass")
	}
	generic := w.Body.String()
	users.token.CreatedAt = time.Now().Add(-2 * time.Minute)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, jsonRequest("POST", "/v1/auth/resend-verification", `{"email":"STUDENT@example.com"}`))
	if w.Code != 202 || mailer.calls != 2 || w.Body.String() != generic {
		t.Fatal(w.Code, w.Body.String(), mailer.calls)
	}
	replacement := emailToken(t, mailer.link)
	if replacement == old {
		t.Fatal("resend reused token")
	}
	for _, email := range []string{"missing@example.com", "student@example.com"} {
		w = httptest.NewRecorder()
		b, _ := json.Marshal(ResendVerificationRequest{Email: email})
		app.ServeHTTP(w, jsonRequest("POST", "/v1/auth/resend-verification", string(b)))
		if w.Code != 202 || w.Body.String() != generic {
			t.Fatal("resend exposes account status")
		}
	}
	for i, raw := range []string{old, replacement} {
		w = httptest.NewRecorder()
		b, _ := json.Marshal(VerifyEmailRequest{Token: raw})
		app.ServeHTTP(w, jsonRequest("POST", "/v1/auth/verify-email", string(b)))
		expected := 400
		if i == 1 {
			expected = 200
		}
		if w.Code != expected {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}

func TestVerificationExpiryAndConcurrentConsumption(t *testing.T) {
	for _, expired := range []bool{true, false} {
		users := &memoryUsers{user: &User{}}
		raw, token, err := newVerificationToken()
		if err != nil {
			t.Fatal(err)
		}
		token.ExpiresAt = time.Now().Add(time.Hour)
		if expired {
			token.ExpiresAt = time.Now()
		}
		users.token = token
		s := NewUserService(users, &memorySessions{}, &captureMailer{}, "https://quiz.test")
		var wg sync.WaitGroup
		results := make(chan error, 2)
		for range 2 {
			wg.Go(func() { results <- s.VerifyEmail(context.Background(), raw) })
		}
		wg.Wait()
		close(results)
		successes := 0
		for err := range results {
			if err == nil {
				successes++
			} else if !errors.Is(err, ErrInvalidVerificationToken) {
				t.Fatal(err)
			}
		}
		expected := 1
		if expired {
			expected = 0
		}
		if successes != expected {
			t.Fatalf("successful consumptions = %d, want %d", successes, expected)
		}
	}
}

func TestSMTPFailureKeepsPendingAccountAndReturnsNoSecret(t *testing.T) {
	users, sessions, mailer, app := verificationApp(t)
	mailer.err = errors.New("sensitive SMTP reply password and recipient")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, jsonRequest("POST", "/v1/auth/register", registrationJSON))
	if w.Code != 201 || users.user == nil || users.user.EmailVerifiedAt != nil || sessions.current != nil {
		t.Fatal(w.Code, w.Body.String())
	}
	var result RegisterResponse
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.VerificationEmailSent || strings.Contains(w.Body.String(), "sensitive") {
		t.Fatal("SMTP failure misreported or leaked")
	}
}

func TestMalformedVerificationTokens(t *testing.T) {
	s := NewUserService(&memoryUsers{}, &memorySessions{}, &captureMailer{}, "https://quiz.test")
	for _, raw := range []string{"", "bad", strings.Repeat("a", 1000), strings.Repeat("a", 42) + "="} {
		if !errors.Is(s.VerifyEmail(context.Background(), raw), ErrInvalidVerificationToken) {
			t.Fatal("malformed token accepted")
		}
	}
}

func TestMailFailureLoggingIsUsefulWithoutLeakingSecrets(t *testing.T) {
	previous := slog.Default()
	defer slog.SetDefault(previous)
	var logs bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	m := &captureMailer{err: errors.New("SMTP authentication failed")}
	s := NewUserService(&memoryUsers{}, &memorySessions{}, m, "https://quiz.test")
	u := &User{Email: "student@example.com"}
	if s.sendVerification(context.Background(), u, "secret-token", time.Now()) {
		t.Fatal("reported SMTP failure as success")
	}
	if !strings.Contains(logs.String(), "SMTP authentication failed") {
		t.Fatal("SMTP failure stage is missing from logs")
	}
	logs.Reset()
	m.err = errors.New("server response includes secret-password, student@example.com and secret-token")
	_ = s.sendVerification(context.Background(), u, "secret-token", time.Now())
	for _, secret := range []string{"secret-password", "student@example.com", "secret-token"} {
		if strings.Contains(logs.String(), secret) {
			t.Fatalf("mail error log leaked %s", secret)
		}
	}
	if !strings.Contains(logs.String(), "details suppressed") {
		t.Fatal("unknown mail error was not redacted")
	}
}
