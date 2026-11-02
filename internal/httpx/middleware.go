package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"time"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *statusWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

// Observe recovers request panics and logs metadata, never URLs, bodies or tokens.
func Observe(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		requestID := r.Header.Get("X-Request-ID")
		if !validRequestID(requestID) {
			var id [16]byte
			if _, err := rand.Read(id[:]); err != nil {
				_ = WriteError(w, http.StatusInternalServerError, "server error")
				return
			}
			requestID = hex.EncodeToString(id[:])
		}
		w.Header().Set("X-Request-ID", requestID)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "same-origin")
		writer := &statusWriter{ResponseWriter: w}
		defer func() {
			if recovered := recover(); recovered != nil {
				// Panic values can contain user data; log the stack only.
				slog.Error("request panic", "request_id", requestID, "stack", string(debug.Stack()))
				if writer.status == 0 {
					_ = WriteError(writer, http.StatusInternalServerError, "server error")
				} else {
					panic(http.ErrAbortHandler)
				}
			}
			route := r.Pattern
			if route == "" {
				route = "unmatched"
			}
			status := writer.status
			if status == 0 {
				status = http.StatusOK
			}
			slog.Info("HTTP request", "request_id", requestID, "method", r.Method,
				"route", route, "status", status, "duration", time.Since(started))
		}()
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		// Keep the same request pointer so ServeMux's matched pattern is logged.
		*r = *r.WithContext(ctx)
		next.ServeHTTP(writer, r)
	})
}

func validRequestID(id string) bool {
	if len(id) == 0 || len(id) > 64 {
		return false
	}
	for _, c := range id {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

func IsSafeMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

// TrustedOrigin protects browser writes, including pre-session login/register.
// An absent Origin requires a trusted Referer. Proxy headers are not trusted.
func TrustedOrigin(publicOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsSafeMethod(r.Method) || r.Method == http.MethodOptions {
			origin := r.Header.Get("Origin")
			if origin == "" && r.Method != http.MethodOptions {
				if referer, err := url.Parse(r.Header.Get("Referer")); err == nil && referer.User == nil {
					origin = referer.Scheme + "://" + referer.Host
				}
			}
			if origin != publicOrigin {
				_ = WriteError(w, http.StatusForbidden, "untrusted request origin")
				return
			}
			if strings.EqualFold(r.Header.Get("Sec-Fetch-Site"), "cross-site") {
				_ = WriteError(w, http.StatusForbidden, "cross-site request rejected")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
