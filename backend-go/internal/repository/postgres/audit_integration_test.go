package postgres_test

import (
	"context"
	"testing"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	pgrepo "digidocs-mgt/backend-go/internal/repository/postgres"
	"digidocs-mgt/backend-go/internal/service"
)

func TestPostgresIntegrationAccessAudit(t *testing.T) {
	ctx := context.Background()
	database := openIntegrationDB(t)
	seedIntegrationBaseRows(t, ctx, database)
	const outsider = "00000000-0000-0000-0000-000000000199"
	if _, err := database.ExecContext(ctx, `INSERT INTO users(id,username,password_hash,display_name,role,status) VALUES($1,'outsider','hash','outsider','member','active')`, outsider); err != nil {
		t.Fatal(err)
	}

	documents := pgrepo.NewDocumentRepository(database)
	doc, err := documents.CreateDocument(ctx, command.DocumentCreateInput{TeamSpaceID: integrationTeamSpaceID, ProjectID: integrationProjectID, Title: "开题报告", CurrentOwnerID: integrationUserID, ActorID: integrationUserID})
	if err != nil {
		t.Fatal(err)
	}
	docID := doc["id"].(string)

	audit := service.NewAuditService(pgrepo.NewAuditRepository(database))
	view := command.AuditEventInput{DocumentID: docID, UserID: integrationUserID, ActionType: service.AuditActionView, IPAddress: "10.0.0.8", TerminalInfo: "jsdom"}
	for i := 0; i < 3; i++ {
		if err := audit.Record(ctx, view); err != nil {
			t.Fatal(err)
		}
	}
	download := command.AuditEventInput{DocumentID: docID, UserID: integrationUserID, ActionType: service.AuditActionDownload, IPAddress: "not-an-ip"}
	for i := 0; i < 2; i++ {
		if err := audit.Record(ctx, download); err != nil {
			t.Fatal(err)
		}
	}
	// A data asset download has no document; it must still be attributed to its project.
	if err := audit.Record(ctx, command.AuditEventInput{UserID: integrationUserID, ActionType: service.AuditActionDownload, Extra: map[string]any{"project_id": integrationProjectID, "data_asset_id": "x"}}); err != nil {
		t.Fatal(err)
	}

	count := func(q string, args ...any) int {
		t.Helper()
		var n int
		if err := database.QueryRowContext(ctx, q, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if n := count(`SELECT COUNT(*) FROM audit_events WHERE action_type='view' AND document_id=$1`, docID); n != 1 {
		t.Fatalf("view should be deduplicated to 1 row, got %d", n)
	}
	if n := count(`SELECT COUNT(*) FROM audit_events WHERE action_type='download' AND document_id=$1`, docID); n != 2 {
		t.Fatalf("downloads must never be deduplicated, got %d", n)
	}
	if n := count(`SELECT COUNT(*) FROM audit_events WHERE action_type='download' AND ip_address IS NULL AND document_id=$1`, docID); n != 2 {
		t.Fatalf("invalid IP should be stored as NULL, got %d", n)
	}

	repo := pgrepo.NewAuditRepository(database)
	member := query.ActorScope{ActorID: integrationUserID}
	summary, err := repo.GetAuditSummary(ctx, integrationProjectID, member)
	if err != nil {
		t.Fatal(err)
	}
	if summary.DownloadCount != 3 || summary.ViewCount != 1 || summary.RecentDownloadCount != 3 {
		t.Fatalf("summary counts wrong: %#v", summary)
	}
	if len(summary.TopDownloads) != 1 || summary.TopDownloads[0].DownloadCount != 2 || summary.TopDownloads[0].Title != "开题报告" {
		t.Fatalf("top downloads wrong: %#v", summary.TopDownloads)
	}
	events, total, err := repo.ListAuditEvents(ctx, query.AuditEventFilter{ProjectID: integrationProjectID, ActionType: "download", Page: 1, PageSize: 20, Scope: member})
	if err != nil || total != 3 || len(events) != 3 {
		t.Fatalf("project-scoped list must include the data asset download: total=%d err=%v", total, err)
	}
	outsiderSummary, err := repo.GetAuditSummary(ctx, "", query.ActorScope{ActorID: outsider})
	if err != nil || outsiderSummary.DownloadCount != 0 || len(outsiderSummary.TopDownloads) != 0 {
		t.Fatalf("outsider must not see project audit data: %#v %v", outsiderSummary, err)
	}
}
