package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"digidocs-mgt/backend-go/internal/transport/http/response"
)

// RateLimiter is a simple in-memory fixed-window limiter keyed by client IP.
// It is intended to protect sensitive endpoints (e.g. login) from brute force,
// not as a general-purpose quota system.
type RateLimiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	counters map[string]*rlCounter
	now      func() time.Time
}

type rlCounter struct {
	windowStart time.Time
	count       int
}

// NewRateLimiter builds a limiter allowing limit requests per window per key.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:    limit,
		window:   window,
		counters: make(map[string]*rlCounter),
		now:      time.Now,
	}
}

func (r *RateLimiter) allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.now()
	c, ok := r.counters[key]
	if !ok || now.Sub(c.windowStart) >= r.window {
		r.sweep(now)
		r.counters[key] = &rlCounter{windowStart: now, count: 1}
		return true
	}
	if c.count >= r.limit {
		return false
	}
	c.count++
	return true
}

// sweep drops expired entries; called opportunistically when the map is large
// so distinct-IP traffic cannot grow memory without bound.
func (r *RateLimiter) sweep(now time.Time) {
	if len(r.counters) < 1024 {
		return
	}
	for k, c := range r.counters {
		if now.Sub(c.windowStart) >= r.window {
			delete(r.counters, k)
		}
	}
}

// Limit wraps a handler, rejecting requests over the per-client limit with 429.
// A non-positive limit disables limiting (passes through).
func (r *RateLimiter) Limit(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if r.limit > 0 && !r.allow(clientIP(req)) {
			response.WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many requests, please retry later")
			return
		}
		next(w, req)
	})
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
