package postgres_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"digidocs-mgt/backend-go/internal/metrics"
)

func TestPostgresIntegrationMetricsGauges(t *testing.T) {
	ctx := context.Background()
	database := openIntegrationDB(t)
	seedIntegrationBaseRows(t, ctx, database)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	req.Header.Set("Authorization", "Bearer t")
	metrics.NewRegistry().Handler(database, "t").ServeHTTP(rec, req)
	body := rec.Body.String()
	if strings.Contains(body, "unavailable") {
		t.Fatalf("every gauge query must run against the migrated schema:\n%s", body)
	}
	for _, name := range []string{"digidocs_ai_tokens_24h", "digidocs_handovers_open", "digidocs_text_chunks", "digidocs_assistant_requests_oldest_pending_seconds"} {
		if !strings.Contains(body, name+" ") {
			t.Fatalf("missing %s:\n%s", name, body)
		}
	}
}
