package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"digidocs-mgt/backend-go/internal/transport/http/response"
)

// TrustedProxyDepth is how many trailing X-Forwarded-For entries were appended
// by reverse proxies under our control. Set from TRUSTED_PROXY_DEPTH at startup;
// 1 (just the frontend nginx) is the safe default. See clientIP.
var TrustedProxyDepth = 1

// maxLoginBodyPeek bounds how much of a login body the limiter reads to find the
// identifier. Login payloads are a few hundred bytes.
const maxLoginBodyPeek = 8 << 10

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
	return r.LimitKeyed(next, nil)
}

// LimitKeyed is Limit with a caller-supplied bucket key.
//
// Why this exists: keying purely on client IP silently degrades to a single
// global bucket whenever the real client address does not survive the network
// path. That is exactly the case for this deployment — public traffic arrives
// through an frp raw-TCP tunnel, so nginx observes the docker gateway for every
// visitor and every internet user lands in the same bucket. Measured behaviour
// before this change: twelve requests carrying twelve different
// X-Forwarded-For values were counted together and the eleventh got a 429.
//
// The consequence is backwards. It does not slow an attacker down relative to
// anyone else; it lets any single user (or attacker) exhaust the shared budget
// and lock every other user out for the rest of the window.
//
// keyFn lets a caller bucket by something that is meaningful even when the IP
// is not — for login, the account being targeted. Returning "" falls back to
// the IP-derived key.
func (r *RateLimiter) LimitKeyed(next http.HandlerFunc, keyFn func(*http.Request) string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if r.limit <= 0 {
			next(w, req)
			return
		}
		key := ""
		if keyFn != nil {
			key = keyFn(req)
		}
		if key == "" {
			key = clientIP(req)
		}
		if !r.allow(key) {
			response.WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many requests, please retry later")
			return
		}
		next(w, req)
	})
}

// clientIP resolves the caller's address for rate-limit keying.
//
// backend-go is only ever reachable through the frontend nginx reverse proxy,
// and that proxy sets X-Forwarded-For via nginx's $proxy_add_x_forwarded_for,
// which APPENDS $remote_addr to whatever the client already sent rather than
// replacing it. So entries closer to the end were observed by infrastructure we
// control, and entries closer to the front are attacker-controlled request data.
//
// TrustedProxyDepth says how many trailing entries were added by proxies we
// trust; the entry just before them is the real client. The default of 1 means
// "only our own nginx", which yields nginx's observed address — unspoofable, but
// only as useful as what nginx actually sees.
//
// IMPORTANT for this deployment: public traffic reaches nginx through an frp
// raw-TCP tunnel, so nginx observes the docker gateway (172.29.0.1) for every
// visitor and the real client address is already gone before any header is set.
// No value of TrustedProxyDepth can recover it. Fixing that requires the tunnel
// to carry the client address — PROXY protocol on the frp proxy plus
// `listen 80 proxy_protocol` / `set_real_ip_from` / `real_ip_header
// proxy_protocol` on nginx — and only then does raising the depth make sense.
// See docs/部署准备与运行说明.md.
//
// Do not raise TrustedProxyDepth beyond the number of proxies that actually
// rewrite the header: every extra step of trust hands one more entry of
// attacker-controlled data to the limiter.
// ClientIP is the caller address used for audit records; it follows the same
// TRUSTED_PROXY_DEPTH rule as rate limiting.
func ClientIP(r *http.Request) string { return clientIP(r) }

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		idx := len(parts) - TrustedProxyDepth
		if idx < 0 {
			idx = 0
		}
		if candidate := strings.TrimSpace(parts[idx]); candidate != "" {
			return candidate
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// LoginIdentifierKey buckets a login attempt by the account it targets.
//
// The body is read and restored so the handler still sees it. Reads are capped:
// the login body is tiny, and an unbounded read here would hand an attacker a
// memory amplifier on an unauthenticated endpoint.
func LoginIdentifierKey(r *http.Request) string {
	if r.Body == nil {
		return ""
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxLoginBodyPeek))
	_ = r.Body.Close()
	// Always restore the body, including on the error path: swallowing it here
	// would turn a peek failure into an unexplained 400 from the handler.
	r.Body = io.NopCloser(bytes.NewReader(raw))
	if err != nil {
		return ""
	}

	var payload struct {
		Identifier string `json:"identifier"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	identifier := strings.ToLower(strings.TrimSpace(payload.Identifier))
	if identifier == "" {
		return ""
	}
	return "login:" + identifier
}
