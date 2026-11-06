package user

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

type googleTestTransport func(*http.Request) (*http.Response, error)

func (f googleTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGoogleLoginCallback(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	const clientID = "test.apps.googleusercontent.com"
	const origin = "https://quiz.test"
	const callback = origin + "/v1/auth/google/callback"
	users := &memoryUsers{}
	handler := NewUserHandler(NewUserService(users, &memorySessions{}, nil, origin), true)
	handler.ConfigureGoogleLogin(clientID, "test-secret", callback)

	publicKey := map[string]string{
		"kty": "RSA", "kid": "test-key", "alg": "RS256", "use": "sig",
		"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}
	var rawToken, challenge string
	var exchanges int
	handler.google.client.Transport = googleTestTransport(func(r *http.Request) (*http.Response, error) {
		var body any
		switch r.URL.String() {
		case handler.google.oauth.Endpoint.TokenURL:
			exchanges++
			if err := r.ParseForm(); err != nil {
				return nil, err
			}
			if r.Form.Get("code") != "test-code" || r.Form.Get("redirect_uri") != callback ||
				r.Form.Get("code_verifier") == "" || oauth2.S256ChallengeFromVerifier(r.Form.Get("code_verifier")) != challenge {
				return nil, fmt.Errorf("incorrect OAuth code exchange")
			}
			body = map[string]any{"access_token": "test-access-token", "token_type": "Bearer", "id_token": rawToken}
		case googleKeysURL:
			body = map[string]any{"keys": []any{publicKey}}
		default:
			return nil, fmt.Errorf("unexpected Google endpoint %s", r.URL)
		}
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": {"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(data)),
		}, nil
	})

	var localID, lastSession string
	for _, tc := range []struct {
		name       string
		change     map[string]any
		wrongState bool
		want       int
	}{
		{name: "first login", want: http.StatusSeeOther},
		{name: "same subject with new email", change: map[string]any{"email": "renamed@gmail.com"}, want: http.StatusSeeOther},
		{name: "same email with different subject", change: map[string]any{"sub": "other-google-id"}, want: http.StatusConflict},
		{name: "unverified Gmail", change: map[string]any{"sub": "other-google-id", "email_verified": false}, want: http.StatusForbidden},
		{name: "external email", change: map[string]any{"sub": "other-google-id", "email": "student@example.test"}, want: http.StatusForbidden},
		{name: "wrong nonce", change: map[string]any{"nonce": "wrong"}, want: http.StatusBadRequest},
		{name: "wrong audience", change: map[string]any{"aud": "other-client"}, want: http.StatusBadGateway},
		{name: "wrong state", wrongState: true, want: http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start := httptest.NewRecorder()
			handler.GoogleStart(start, httptest.NewRequest(http.MethodGet, origin+"/v1/auth/google/start", nil))
			location, err := url.Parse(start.Header().Get("Location"))
			if start.Code != http.StatusFound || err != nil || location.Host != "accounts.google.com" {
				t.Fatalf("start redirect = %d %v", start.Code, location)
			}
			query := location.Query()
			challenge = query.Get("code_challenge")
			state, nonce := query.Get("state"), query.Get("nonce")
			if query.Get("code_challenge_method") != "S256" || challenge == "" || state == "" || nonce == "" || query.Get("redirect_uri") != callback {
				t.Fatal("Google redirect missing OAuth parameters")
			}
			cookies := start.Result().Cookies()
			if len(cookies) != 1 || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].SameSite != http.SameSiteLaxMode ||
				cookies[0].Path != "/" || !strings.HasPrefix(cookies[0].Name, "__Host-") {
				t.Fatal("invalid state cookie")
			}
			claims := map[string]any{
				"iss": googleIssuer, "aud": clientID, "sub": "google-id", "nonce": nonce,
				"email": "Student@gmail.com", "email_verified": true, "name": "Test Student",
				"exp": time.Now().Add(time.Minute).Unix(), "iat": time.Now().Unix(),
			}
			for k, v := range tc.change {
				claims[k] = v
			}
			rawToken = signGoogleTestToken(t, key, claims)
			if tc.wrongState {
				state = "wrong"
			}
			before := exchanges
			request := httptest.NewRequest(http.MethodGet, callback+"?code=test-code&state="+url.QueryEscape(state), nil)
			request.AddCookie(cookies[0])
			response := httptest.NewRecorder()
			handler.GoogleCallback(response, request)
			if response.Code != tc.want {
				t.Fatalf("callback status = %d, want %d, body = %s", response.Code, tc.want, response.Body)
			}
			if tc.wrongState && exchanges != before {
				t.Fatal("exchanged a code with the wrong state")
			}
			if tc.want != http.StatusSeeOther {
				for _, cookie := range response.Result().Cookies() {
					if cookie.Name == sessionCookieName {
						t.Fatal("failed login set a session cookie")
					}
				}
				return
			}
			if response.Header().Get("Location") != "/v1/auth/me" {
				t.Fatal("wrong callback destination")
			}
			cookies = response.Result().Cookies()
			if len(cookies) != 2 || cookies[0].MaxAge != -1 || cookies[1].Name != sessionCookieName || cookies[1].Value == lastSession {
				t.Fatal("callback should clear its state and set a new session")
			}
			lastSession = cookies[1].Value
			meRequest := httptest.NewRequest(http.MethodGet, origin+"/v1/auth/me", nil)
			meRequest.AddCookie(cookies[1])
			me := httptest.NewRecorder()
			handler.RequireAuth(http.HandlerFunc(handler.Me)).ServeHTTP(me, meRequest)
			var profile UserResponse
			if err := json.Unmarshal(me.Body.Bytes(), &profile); err != nil || me.Code != http.StatusOK || profile.Email != "student@gmail.com" || profile.EmailVerifiedAt == nil || profile.ID == "" {
				t.Fatalf("Google session could not read current user: %d %s", me.Code, me.Body)
			}
			if localID != "" && profile.ID != localID {
				t.Fatal("repeat sign-in changed the local user ID")
			}
			localID = profile.ID
		})
	}
	if users.creates != 1 {
		t.Fatalf("created %d users, want 1", users.creates)
	}
}

func signGoogleTestToken(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	data, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	encode := base64.RawURLEncoding.EncodeToString
	message := encode([]byte(`{"alg":"RS256","kid":"test-key"}`)) + "." + encode(data)
	hash := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hash[:])
	if err != nil {
		t.Fatal(err)
	}
	return message + "." + encode(signature)
}

func TestGoogleStateRejectsTamperingAndExpiry(t *testing.T) {
	login := &googleLogin{oauth: oauth2.Config{ClientSecret: "test-secret"}}
	now := time.Now().Truncate(time.Second)
	value := login.encodeState("state", "verifier", "nonce", now.Add(googleStateTTL))
	if v, n, ok := login.decodeState(value, "state", now); !ok || v != "verifier" || n != "nonce" {
		t.Fatal("valid state cookie rejected")
	}
	if _, _, ok := login.decodeState("tampered"+value, "state", now); ok {
		t.Fatal("tampered state cookie accepted")
	}
	if _, _, ok := login.decodeState(value, "state", now.Add(googleStateTTL)); ok {
		t.Fatal("expired state cookie accepted")
	}
}
