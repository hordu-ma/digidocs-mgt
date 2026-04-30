package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

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
