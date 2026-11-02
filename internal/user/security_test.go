package user

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"qtp/internal/httpx"
	"qtp/internal/session"
)

type memoryUsers struct {
	user    *User
	err     error
	email   string
	creates int
}

func (s *memoryUsers) Create(_ context.Context, u *User) (*User, error) {
	s.creates++
	if s.err != nil {
		return nil, s.err
	}
	u.ID = "user-id"
	s.user = u
	return u, nil
}
func (s *memoryUsers) FindByEmail(_ context.Context, email string) (*User, error) {
	s.email = email
	if s.err != nil {
		return nil, s.err
	}
	if s.user == nil {
		return nil, ErrUserNotFound
	}
	return s.user, nil
}
func (s *memoryUsers) FindByID(context.Context, string) (*User, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.user == nil {
		return nil, ErrUserNotFound
	}
	return s.user, nil
}

type memorySessions struct {
	current *session.Session
	err     error
	deleted bool
}

func (s *memorySessions) Create(_ context.Context, value *session.Session) (*session.Session, error) {
	if s.err != nil {
		return nil, s.err
	}
	s.current = value
	return value, nil
}
func (s *memorySessions) FindActiveByTokenHash(_ context.Context, hash []byte) (*session.Session, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.current == nil || s.deleted || !bytes.Equal(hash, s.current.TokenHash) {
		return nil, session.ErrSessionNotFound
	}
	return s.current, nil
}
func (s *memorySessions) DeleteByTokenHash(_ context.Context, hash []byte) error {
	if s.err != nil {
		return s.err
	}
	if s.current != nil && bytes.Equal(hash, s.current.TokenHash) {
		s.deleted = true
	}
	return nil
}

func jsonRequest(method, path, body string) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "https://quiz.test")
	return r
}

func TestLoginMissingWrongPasswordAndDatabaseFailure(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	var unauthorizedBody string
	for _, tc := range []struct {
		name   string
		store  *memoryUsers
		status int
	}{
		{"missing account", &memoryUsers{}, 401},
		{"wrong password", &memoryUsers{user: &User{PasswordHash: string(hash)}}, 401},
		{"database failure", &memoryUsers{err: errors.New("postgres internal path /private/password")}, 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := NewUserHandler(NewUserService(tc.store, &memorySessions{}), true)
			w := httptest.NewRecorder()
			h.Login(w, jsonRequest("POST", "/v1/auth/login", `{"email":"student@example.com","password":"wrong-password"}`))
			if w.Code != tc.status {
				t.Fatal(w.Code, w.Body.String())
			}
			if tc.status == 401 {
				if unauthorizedBody != "" && unauthorizedBody != w.Body.String() {
					t.Fatal("login discloses account existence")
				}
				unauthorizedBody = w.Body.String()
			}
			if strings.Contains(w.Body.String(), "postgres") || strings.Contains(w.Body.String(), "/private") {
				t.Fatal("internal error leaked")
			}
		})
	}
}

func TestRegisterValidationAndSingleJSONResponse(t *testing.T) {
	for _, tc := range []struct {
		password string
		expected error
	}{
		{strings.Repeat("я", 7), ErrPasswordTooShort},
		{strings.Repeat("я", 37), ErrPasswordTooLong},
	} {
		s := NewUserService(&memoryUsers{}, &memorySessions{})
		_, err := s.Register(context.Background(), RegisterRequest{Email: "student@example.com", Name: "Student", Password: tc.password})
		if !errors.Is(err, tc.expected) {
			t.Fatal(err)
		}
	}
	for _, email := range []string{"Student <student@example.com>", "a@example.com,b@example.com", "not-an-email", "a\n@example.com"} {
		if validEmail(email) {
			t.Fatalf("accepted invalid email %q", email)
		}
	}
	u := &memoryUsers{}
	h := NewUserHandler(NewUserService(u, &memorySessions{}), false)
	w := httptest.NewRecorder()
	h.Register(w, jsonRequest("POST", "/v1/auth/register", `{"email":"  Student@Example.COM  ","name":" Student ","password":"password123"}`))
	if w.Code != 201 || u.creates != 1 || u.user.Email != "student@example.com" {
		t.Fatal(w.Code, w.Body.String())
	}
	d := json.NewDecoder(w.Body)
	var result map[string]any
	if err := d.Decode(&result); err != nil {
		t.Fatal(err)
	}
	if err := d.Decode(&result); !errors.Is(err, io.EOF) {
		t.Fatal("register wrote multiple responses")
	}
	if strings.Contains(string(mustJSON(t, u.user)), "PasswordHash") || strings.Contains(string(mustJSON(t, u.user)), "$2") {
		t.Fatal("password hash exposed")
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestSessionCSRFAndLogoutLifecycle(t *testing.T) {
	users := &memoryUsers{user: &User{ID: "user-id", Email: "student@example.com", PasswordHash: string(dummyPasswordHash)}}
	sessions := &memorySessions{}
	service := NewUserService(users, sessions)
	h := NewUserHandler(service, true)
	router := http.NewServeMux()
	router.HandleFunc("POST /v1/auth/login", h.Login)
	router.Handle("GET /v1/auth/csrf", h.RequireAuth(http.HandlerFunc(h.CSRF)))
	router.Handle("POST /v1/auth/logout", h.RequireAuth(http.HandlerFunc(h.Logout)))
	called := 0
	router.Handle("PUT /v1/resource", h.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(204)
	})))
	app := httpx.TrustedOrigin("https://quiz.test", router)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, jsonRequest("POST", "/v1/auth/login", `{"email":"STUDENT@example.com","password":"dummy password for missing accounts"}`))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatal("missing session cookie")
	}
	cookie := cookies[0]
	if !cookie.Secure || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" || cookie.Domain != "" {
		t.Fatal("unsafe cookie attributes")
	}
	if cookie.Value == sessions.current.CSRFToken || !bytes.Equal(session.HashToken(cookie.Value), sessions.current.TokenHash) {
		t.Fatal("session/CSRF tokens not independent or session not hashed")
	}
	if delta := cookie.Expires.Sub(sessions.current.ExpiresAt); delta < -time.Second || delta > time.Second {
		t.Fatal("cookie expiry differs from session")
	}
	if strings.Contains(w.Body.String(), cookie.Value) || strings.Contains(w.Body.String(), "PasswordHash") {
		t.Fatal("sensitive login response")
	}
	r := jsonRequest("GET", "/v1/auth/csrf", "")
	r.AddCookie(cookie)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, r)
	var csrf struct {
		Token string `json:"csrf_token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &csrf); err != nil || csrf.Token != sessions.current.CSRFToken {
		t.Fatal("cannot obtain CSRF token")
	}
	for _, token := range []string{"", "another-session-token", csrf.Token} {
		r = jsonRequest("PUT", "/v1/resource", "")
		r.AddCookie(cookie)
		r.Header.Set("X-CSRF-Token", token)
		w = httptest.NewRecorder()
		app.ServeHTTP(w, r)
		expected := 403
		if token == csrf.Token {
			expected = 204
		}
		if w.Code != expected {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	if called != 1 {
		t.Fatal("CSRF bypass")
	}
	// Correct token with a forged Origin must still be rejected.
	r = jsonRequest("PUT", "/v1/resource", "")
	r.AddCookie(cookie)
	r.Header.Set("Origin", "https://evil.test")
	r.Header.Set("X-CSRF-Token", csrf.Token)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, r)
	if w.Code != 403 || called != 1 {
		t.Fatal("Origin bypass")
	}
	// Expired sessions must not expose CSRF tokens.
	expires := sessions.current.ExpiresAt
	sessions.current.ExpiresAt = time.Now().Add(-time.Second)
	r = jsonRequest("GET", "/v1/auth/csrf", "")
	r.AddCookie(cookie)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatal("expired session accepted")
	}
	sessions.current.ExpiresAt = expires
	r = jsonRequest("POST", "/v1/auth/logout", "")
	r.AddCookie(cookie)
	r.Header.Set("X-CSRF-Token", csrf.Token)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, r)
	if w.Code != 204 || w.Body.Len() != 0 || !sessions.deleted {
		t.Fatal("logout did not revoke session")
	}
	cleared := w.Result().Cookies()[0]
	if cleared.Name != cookie.Name || cleared.Path != cookie.Path || cleared.MaxAge != -1 || !cleared.Secure || !cleared.HttpOnly {
		t.Fatal("cookie not cleared consistently")
	}
	r = jsonRequest("GET", "/v1/auth/csrf", "")
	r.AddCookie(cookie)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatal("logged-out session accepted")
	}
}
