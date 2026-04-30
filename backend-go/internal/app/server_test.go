package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"digidocs-mgt/backend-go/internal/config"
)

func TestNewServer(t *testing.T) {
	cfg := config.Config{
		HTTPAddr:            ":0",
		APIV1Prefix:         "/api/v1",
		DataBackend:         "memory",
		StorageBackend:      "memory",
		JWTSecret:           "secret",
		WorkerCallbackToken: "worker",
		CodeRepoRoot:        t.TempDir(),
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	if server.Addr != ":0" {
		t.Fatalf("addr = %q", server.Addr)
	}
	if server.Handler == nil {
		t.Fatal("expected handler")
	}
	if server.ReadHeaderTimeout != 5*time.Second ||
		server.WriteTimeout != 5*time.Minute ||
		server.IdleTimeout != 2*time.Minute {
		t.Fatalf("unexpected timeouts: %+v", server)
	}

	rec := httptest.NewRecorder()
	server.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz status = %d", rec.Code)
	}
}
