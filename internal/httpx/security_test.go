package httpx

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestReadJSONBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, body, contentType string
		limit                   int64
		status                  int
	}{
		{"valid", `{"name":"test"} `, "application/json; charset=utf-8", 100, 200},
		{"second object", `{"name":"test"}{}`, "application/json", 100, 400},
		{"trailing value", `{}null`, "application/json", 100, 400},
		{"trailing garbage", `{}!`, "application/json", 100, 400},
		{"null", `null`, "application/json", 100, 400},
		{"array", `[]`, "application/json", 100, 400},
		{"unknown field", `{"password":"secret"}`, "application/json", 100, 400},
		{"empty", ``, "application/json", 100, 400},
		{"oversize", `{"name":"long value"}`, "application/json", 10, 413},
		{"oversize whitespace", `{}` + strings.Repeat(" ", 100), "application/json", 10, 413},
		{"form", `name=test`, "application/x-www-form-urlencoded", 100, 415},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/", strings.NewReader(tc.body))
			r.ContentLength = -1 // Also enforce limits on unknown/chunked lengths.
			r.Header.Set("Content-Type", tc.contentType)
			w := httptest.NewRecorder()
			var dst struct {
				Name string `json:"name"`
			}
			ok := ReadJSON(w, r, &dst, tc.limit)
			if ok != (tc.status == 200) || w.Code != tc.status {
				t.Fatalf("ok=%v status=%d body=%s", ok, w.Code, w.Body.String())
			}
			if strings.Contains(w.Body.String(), "secret") {
				t.Fatal("request content leaked")
			}
		})
	}
}

func TestWriteJSONNoContentAndEncodingFailure(t *testing.T) {
	w := httptest.NewRecorder()
	if err := WriteJSON(w, 204, map[string]string{"ignored": "body"}); err != nil {
		t.Fatal(err)
	}
	if w.Code != 204 || w.Body.Len() != 0 {
		t.Fatal("204 must have no body")
	}
	w = httptest.NewRecorder()
	if err := WriteJSON(w, 200, make(chan int)); err == nil {
		t.Fatal("expected encoding error")
	}
	if w.Code != 500 || w.Body.String() != "{\"error\":\"server error\"}\n" {
		t.Fatal(w.Body.String())
	}
}

func TestTrustedOrigin(t *testing.T) {
	for _, tc := range []struct {
		method, origin, referer, fetchSite string
		allowed                            bool
	}{
		{"POST", "https://quiz.test", "", "same-origin", true},
		{"POST", "", "https://quiz.test/page", "", true},
		{"POST", "", "", "", false},
		{"POST", "null", "https://quiz.test/page", "", false},
		{"POST", "https://quiz.test.evil.test", "", "", false},
		{"POST", "https://evil.test", "https://quiz.test/page", "", false},
		{"POST", "http://quiz.test", "", "", false},
		{"POST", "https://quiz.test", "", "cross-site", false},
		{"OPTIONS", "https://evil.test", "", "", false},
		{"GET", "", "", "", true},
	} {
		r := httptest.NewRequest(tc.method, "https://quiz.test/v1/auth/login", nil)
		r.Header.Set("Origin", tc.origin)
		r.Header.Set("Referer", tc.referer)
		r.Header.Set("Sec-Fetch-Site", tc.fetchSite)
		called := false
		h := TrustedOrigin("https://quiz.test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(204) }))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if called != tc.allowed {
			t.Fatalf("%+v: called=%v status=%d", tc, called, w.Code)
		}
		if !tc.allowed && w.Code != 403 {
			t.Fatal(w.Code)
		}
		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatal("unexpected CORS permission")
		}
	}
}

func TestRecoveryAndMetadataDoNotLeakRequest(t *testing.T) {
	var logs bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(old) })
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/items/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") == "panic" {
			panic("secret-document-content")
		}
		_ = WriteJSON(w, 200, map[string]string{"ok": "yes"})
	})
	h := Observe(mux)
	for _, tc := range []struct {
		path   string
		status int
	}{{"/v1/items/panic", 500}, {"/v1/items/secret-id?token=secret-token", 200}} {
		r := httptest.NewRequest("GET", tc.path, nil)
		r.Header.Set("Cookie", "session=secret-cookie")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.status || !validRequestID(w.Header().Get("X-Request-ID")) {
			t.Fatal(w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), "secret") {
			t.Fatal("panic data leaked")
		}
	}
	for _, secret := range []string{"secret-document-content", "secret-id", "secret-token", "secret-cookie"} {
		if strings.Contains(logs.String(), secret) {
			t.Fatalf("logged %s", secret)
		}
	}
	if !strings.Contains(logs.String(), "GET /v1/items/{id}") {
		t.Fatal("missing route template")
	}
}

func TestRequestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
	Observe(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Err() != context.Canceled {
			t.Error("request cancellation lost")
		}
		if _, ok := r.Context().Deadline(); !ok {
			t.Error("missing deadline")
		}
	})).ServeHTTP(httptest.NewRecorder(), r)
}

func TestRouterErrorsRemainJSON(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /items/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") != "one" {
			t.Fatal("path values lost")
		}
		_ = WriteJSON(w, 200, map[string]string{"id": "one"})
	})
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/items/one", 200},
		{"DELETE", "/items/one", 405},
		{"GET", "/missing", 404},
	} {
		w := httptest.NewRecorder()
		JSONRouter(mux).ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		if w.Code != tc.status || !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
			t.Fatal(w.Code, w.Body.String())
		}
		if tc.status == 405 && !strings.Contains(w.Header().Get("Allow"), "GET") {
			t.Fatal("missing Allow header")
		}
	}
}

func TestRateLimiterConcurrentBoundedAndExpiry(t *testing.T) {
	l := NewRateLimiter(10, time.Minute, 2)
	now := time.Now()
	l.now = func() time.Time { return now }
	var allowed atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if l.allow("one") {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()
	if allowed.Load() != 10 {
		t.Fatal(allowed.Load())
	}
	if !l.allow("two") || l.allow("three") {
		t.Fatal("table not bounded")
	}
	now = now.Add(time.Minute)
	if !l.allow("three") || len(l.entries) > 2 {
		t.Fatal("expiry did not free entries")
	}
	if !l.allow("four") {
		t.Fatal("cannot fill table")
	}
	w := httptest.NewRecorder()
	l.Middleware(func(*http.Request) string { return "full" }, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("limited request executed") })).ServeHTTP(w, httptest.NewRequest("POST", "/", nil))
	if w.Code != 429 || w.Header().Get("Retry-After") == "" {
		t.Fatal(w.Code)
	}
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "192.0.2.10:1234"
	r.Header.Set("X-Forwarded-For", "attacker")
	if RemoteIP(r) != "192.0.2.10" {
		t.Fatal("trusted unverified proxy header")
	}
}
