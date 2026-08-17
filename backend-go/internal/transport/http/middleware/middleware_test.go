package middleware

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"digidocs-mgt/backend-go/internal/domain/auth"
	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/shared"
)

func TestChainAppliesMiddlewareInOrder(t *testing.T) {
	var order []string
	handler := Chain(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			order = append(order, "handler")
		}),
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, "first-before")
				next.ServeHTTP(w, r)
				order = append(order, "first-after")
			})
		},
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, "second-before")
				next.ServeHTTP(w, r)
				order = append(order, "second-after")
			})
		},
	)

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	want := []string{"first-before", "second-before", "handler", "second-after", "first-after"}
	if len(order) != len(want) {
		t.Fatalf("order length = %d, want %d: %+v", len(order), len(want), order)
	}
	for idx := range want {
		if order[idx] != want[idx] {
			t.Fatalf("order[%d] = %q, want %q; full=%+v", idx, order[idx], want[idx], order)
		}
	}
}

func TestCORS(t *testing.T) {
	handler := CORS("https://a.test, https://b.test")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://b.test")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://b.test" {
		t.Fatalf("allow origin = %q", got)
	}

	disallowed := httptest.NewRequest(http.MethodGet, "/", nil)
	disallowed.Header.Set("Origin", "https://no.test")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, disallowed)
	if rec.Code != http.StatusCreated {
		t.Fatalf("disallowed origin should still reach handler, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected allow origin: %q", got)
	}

	if got := matchOrigin("https://x.test", parseOrigins("*")); got != "*" {
		t.Fatalf("wildcard match = %q", got)
	}
}

func TestJSONContentType(t *testing.T) {
	handler := JSONContentType(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("content type = %q", got)
	}
}

func TestRequestID(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := shared.RequestIDFromContext(r.Context()); got != "req-123" {
			t.Fatalf("context request id = %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", "req-123")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Request-Id"); got != "req-123" {
		t.Fatalf("response request id = %q", got)
	}

	generated := RequestID(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	rec = httptest.NewRecorder()
	generated.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Header().Get("X-Request-Id") == "" {
		t.Fatal("expected generated request id")
	}
	if newRequestID() == "" {
		t.Fatal("expected direct generated request id")
	}
}

func TestAccessLogRecordsStatus(t *testing.T) {
	handler := AccessLog(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/path", nil))

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", rec.Code)
	}
}

func TestAuthAndRequireAdmin(t *testing.T) {
	tokenService := service.NewTokenService("secret")
	adminToken, err := tokenService.Generate(auth.Claims{UserID: "u-admin", Username: "admin", Role: "admin"})
	if err != nil {
		t.Fatalf("generate admin token: %v", err)
	}
	memberToken, err := tokenService.Generate(auth.Claims{UserID: "u-member", Username: "member", Role: "member"})
	if err != nil {
		t.Fatalf("generate member token: %v", err)
	}

	authHandler := Auth(tokenService)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if UserIDFromContext(r.Context()) != "u-member" || UserRoleFromContext(r.Context()) != "member" {
			t.Fatalf("unexpected claims in context")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+memberToken)
	rec := httptest.NewRecorder()
	authHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("auth status = %d", rec.Code)
	}

	missing := httptest.NewRecorder()
	authHandler.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/", nil))
	if missing.Code != http.StatusUnauthorized {
		t.Fatalf("missing auth status = %d", missing.Code)
	}

	adminHandler := RequireAdmin(tokenService)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	adminHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("admin status = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+memberToken)
	rec = httptest.NewRecorder()
	adminHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("member admin status = %d", rec.Code)
	}
}

func TestContextHelpersDefault(t *testing.T) {
	if _, ok := ClaimsFromContext(context.Background()); ok {
		t.Fatal("expected no claims")
	}
	if got := UserIDFromContext(context.Background()); got != "00000000-0000-0000-0000-000000000001" {
		t.Fatalf("default user id = %q", got)
	}
	if got := UserRoleFromContext(context.Background()); got != "" {
		t.Fatalf("default role = %q", got)
	}
}

func TestLimitBodyRejectsOversizedJSON(t *testing.T) {
	handler := LimitBody(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	big := bytes.Repeat([]byte("a"), MaxJSONBodySize+1)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/x", bytes.NewReader(big))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized JSON status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestLimitBodyExemptsMultipartAndGit(t *testing.T) {
	handler := LimitBody(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			http.Error(w, "too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	big := bytes.Repeat([]byte("a"), MaxJSONBodySize+1)

	cases := []struct {
		name        string
		path        string
		contentType string
	}{
		{"multipart", "/api/v1/documents", "multipart/form-data; boundary=x"},
		{"git path", "/api/v1/git/foo.git/git-receive-pack", "application/x-git-receive-pack-request"},
		{"internal callback", "/api/v1/internal/worker-results", "application/json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, bytes.NewReader(big))
			req.Header.Set("Content-Type", tc.contentType)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("%s status = %d, want %d", tc.name, rec.Code, http.StatusOK)
			}
		})
	}
}

func TestRateLimiterAllowsUpToLimitThenBlocks(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)
	calls := 0
	h := rl.Limit(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	})

	codes := make([]int, 0, 5)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		req.RemoteAddr = "203.0.113.7:5555"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		codes = append(codes, rec.Code)
	}
	// first 3 pass, next 2 are 429
	want := []int{200, 200, 200, http.StatusTooManyRequests, http.StatusTooManyRequests}
	for i := range want {
		if codes[i] != want[i] {
			t.Fatalf("req %d code = %d, want %d (all=%v)", i, codes[i], want[i], codes)
		}
	}
	if calls != 3 {
		t.Fatalf("handler calls = %d, want 3", calls)
	}
}

func TestRateLimiterIsolatesByClientIP(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)
	h := rl.Limit(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	do := func(xff string) int {
		req := httptest.NewRequest(http.MethodPost, "/x", nil)
		req.Header.Set("X-Forwarded-For", xff)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	if c := do("198.51.100.1"); c != http.StatusOK {
		t.Fatalf("ip1 first = %d", c)
	}
	if c := do("198.51.100.1"); c != http.StatusTooManyRequests {
		t.Fatalf("ip1 second = %d, want 429", c)
	}
	if c := do("198.51.100.2"); c != http.StatusOK {
		t.Fatalf("ip2 first = %d, want 200 (isolated)", c)
	}
}

func TestRateLimiterTrustsLastXFFHopNotClientSuppliedPrefix(t *testing.T) {
	// nginx's $proxy_add_x_forwarded_for appends the real connecting address
	// as the LAST entry; the client can set any value(s) before it. Keying on
	// the first entry would let a client bypass the limiter by sending a
	// fresh spoofed prefix on every request while its real (last-hop) address
	// stays constant.
	rl := NewRateLimiter(1, time.Minute)
	h := rl.Limit(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	do := func(xff string) int {
		req := httptest.NewRequest(http.MethodPost, "/x", nil)
		req.Header.Set("X-Forwarded-For", xff)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	if c := do("203.0.113.9, 198.51.100.50"); c != http.StatusOK {
		t.Fatalf("first request = %d, want 200", c)
	}
	if c := do("203.0.113.99, 198.51.100.50"); c != http.StatusTooManyRequests {
		t.Fatalf("spoofed-prefix retry = %d, want 429 (real last-hop IP unchanged)", c)
	}
}

func TestRateLimiterZeroDisables(t *testing.T) {
	rl := NewRateLimiter(0, time.Minute)
	h := rl.Limit(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	for i := 0; i < 50; i++ {
		req := httptest.NewRequest(http.MethodPost, "/x", nil)
		req.RemoteAddr = "10.0.0.1:1"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("disabled limiter blocked at %d", i)
		}
	}
}

// The login limiter must bucket by the targeted account, not only by client IP:
// behind an frp raw-TCP tunnel every visitor shares one address, so an IP-only
// limiter degrades to a single global bucket that any one user can exhaust.
func TestLoginIdentifierKeyBucketsByAccount(t *testing.T) {
	newReq := func(body string) *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
		req.RemoteAddr = "10.0.0.1:1234"
		return req
	}

	if got := LoginIdentifierKey(newReq(`{"identifier":"alice","password":"x"}`)); got != "login:alice" {
		t.Fatalf("key = %q, want login:alice", got)
	}
	// Case and surrounding whitespace must not create separate buckets.
	if got := LoginIdentifierKey(newReq(`{"identifier":"  ALICE  ","password":"x"}`)); got != "login:alice" {
		t.Fatalf("key = %q, want the normalized login:alice", got)
	}
	// No identifier: fall back to the IP-derived key rather than lumping every
	// malformed request into one bucket.
	if got := LoginIdentifierKey(newReq(`{"password":"x"}`)); got != "" {
		t.Fatalf("key = %q, want empty so the caller falls back to client IP", got)
	}
	if got := LoginIdentifierKey(newReq("not json")); got != "" {
		t.Fatalf("key = %q, want empty for an unparseable body", got)
	}
}

func TestLoginIdentifierKeyLeavesBodyReadable(t *testing.T) {
	body := `{"identifier":"bob","password":"secret"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))

	if got := LoginIdentifierKey(req); got != "login:bob" {
		t.Fatalf("key = %q, want login:bob", got)
	}
	// The handler still has to be able to read the credentials.
	rest, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read body after peek: %v", err)
	}
	if string(rest) != body {
		t.Fatalf("body after peek = %q, want it restored intact", rest)
	}
}

func TestLimitKeyedSeparatesBuckets(t *testing.T) {
	limiter := NewRateLimiter(1, time.Minute)
	handler := limiter.LimitKeyed(
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) },
		LoginIdentifierKey,
	)

	call := func(identifier string) int {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
			strings.NewReader(`{"identifier":"`+identifier+`","password":"x"}`))
		req.RemoteAddr = "10.0.0.1:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	if code := call("alice"); code != http.StatusOK {
		t.Fatalf("first alice attempt = %d, want 200", code)
	}
	if code := call("alice"); code != http.StatusTooManyRequests {
		t.Fatalf("second alice attempt = %d, want 429", code)
	}
	// Same source address, different account: must not be collateral damage.
	if code := call("bob"); code != http.StatusOK {
		t.Fatalf("bob attempt = %d, want 200 — one account must not lock out another", code)
	}
}

func TestClientIPHonoursTrustedProxyDepth(t *testing.T) {
	original := TrustedProxyDepth
	t.Cleanup(func() { TrustedProxyDepth = original })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.9:1111"
	req.Header.Set("X-Forwarded-For", "203.0.113.7, 198.51.100.1, 172.29.0.1")

	// Depth 1 = only our own nginx appended an entry.
	TrustedProxyDepth = 1
	if got := clientIP(req); got != "172.29.0.1" {
		t.Fatalf("depth 1 = %q, want the address our nginx observed", got)
	}
	// Depth 2 = nginx plus one upstream proxy that rewrites the header.
	TrustedProxyDepth = 2
	if got := clientIP(req); got != "198.51.100.1" {
		t.Fatalf("depth 2 = %q, want the entry before the trusted tail", got)
	}
	// Depth deeper than the chain must clamp instead of panicking.
	TrustedProxyDepth = 99
	if got := clientIP(req); got != "203.0.113.7" {
		t.Fatalf("oversized depth = %q, want it clamped to the first entry", got)
	}
}
