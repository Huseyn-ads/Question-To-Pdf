package httpx

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type rateEntry struct {
	count int
	reset time.Time
}

// RateLimiter uses bounded memory. Full tables reject new keys until expiry.
// It is intended for a single backend instance.
type RateLimiter struct {
	mu         sync.Mutex
	entries    map[string]rateEntry
	maxEntries int
	limit      int
	window     time.Duration
	now        func() time.Time
}

func NewRateLimiter(limit int, window time.Duration, maxEntries int) *RateLimiter {
	return &RateLimiter{entries: make(map[string]rateEntry), limit: limit,
		window: window, maxEntries: maxEntries, now: time.Now}
}

func (l *RateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	entry, exists := l.entries[key]
	if !exists || !now.Before(entry.reset) {
		if !exists && len(l.entries) >= l.maxEntries {
			for k, old := range l.entries {
				if !now.Before(old.reset) {
					delete(l.entries, k)
				}
			}
			if len(l.entries) >= l.maxEntries {
				return false
			}
		}
		entry = rateEntry{reset: now.Add(l.window)}
	}
	if entry.count >= l.limit {
		return false
	}
	entry.count++
	l.entries[key] = entry
	return true
}

func (l *RateLimiter) Middleware(key func(*http.Request) string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(key(r)) {
			w.Header().Set("Retry-After", strconv.Itoa(int(l.window.Seconds())+1))
			_ = WriteError(w, http.StatusTooManyRequests, "too many requests; try again later")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RemoteIP deliberately ignores X-Forwarded-For supplied by an arbitrary client.
func RemoteIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
