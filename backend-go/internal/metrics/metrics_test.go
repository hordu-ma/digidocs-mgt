package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMiddlewareRecordsMatchedPatternAndStatus(t *testing.T) {
	reg := NewRegistry()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/documents/{id}", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) })
	h := reg.Middleware(mux)
	for _, id := range []string{"a", "b", "c"} {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/api/v1/documents/"+id, nil))
	}
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/nope", nil))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	req.Header.Set("Authorization", "Bearer t")
	reg.Handler(nil, "t").ServeHTTP(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, `digidocs_http_requests_total{route="GET /api/v1/documents/{id}",status="4xx"} 3`) {
		t.Fatalf("ids must collapse into one route series:\n%s", body)
	}
	if !strings.Contains(body, `route="unmatched"`) {
		t.Fatalf("unmatched requests should be counted:\n%s", body)
	}
}

func TestHandlerAuth(t *testing.T) {
	reg := NewRegistry()
	rec := httptest.NewRecorder()
	reg.Handler(nil, "").ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("no token configured must disable the endpoint, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	reg.Handler(nil, "t").ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("missing bearer must be 401, got %d", rec.Code)
	}
}
