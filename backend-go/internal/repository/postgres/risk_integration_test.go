package postgres_test

import (
	"context"
	"testing"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	pgrepo "digidocs-mgt/backend-go/internal/repository/postgres"
)

func TestPostgresIntegrationRiskRules(t *testing.T) {
	ctx := context.Background()
	database := openIntegrationDB(t)
	seedIntegrationBaseRows(t, ctx, database)
	const gone = "00000000-0000-0000-0000-000000000501"
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := database.ExecContext(ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO users(id,username,password_hash,display_name,role,status) VALUES($1,'gone','h','离组同学','member','disabled')`, gone)
	documents := pgrepo.NewDocumentRepository(database)
	newDoc := func(owner string) string {
		t.Helper()
		d, err := documents.CreateDocument(ctx, command.DocumentCreateInput{TeamSpaceID: integrationTeamSpaceID, ProjectID: integrationProjectID, Title: "doc", CurrentOwnerID: owner, ActorID: integrationUserID})
		if err != nil {
			t.Fatal(err)
		}
		return d["id"].(string)
	}
	fresh := newDoc(integrationUserID)
	stale := newDoc(integrationUserID)
	unaccepted := newDoc(integrationUserID)
	recentTransfer := newDoc(integrationUserID)
	orphan := newDoc(gone)
	staleFinal := newDoc(integrationUserID)
	exec(`UPDATE documents SET updated_at = NOW() - INTERVAL '40 days' WHERE id IN ($1, $2)`, stale, staleFinal)
	exec(`UPDATE documents SET current_status='finalized' WHERE id=$1`, staleFinal)
	exec(`UPDATE documents SET current_status='pending_handover', updated_at = NOW() - INTERVAL '8 days' WHERE id=$1`, unaccepted)
	exec(`UPDATE documents SET current_status='pending_handover', updated_at = NOW() - INTERVAL '2 days' WHERE id=$1`, recentTransfer)
	exec(`INSERT INTO graduation_handovers(id,target_user_id,receiver_user_id,project_id,status,generated_by,generated_at)
		VALUES('30000000-0000-0000-0000-000000000501',$1,$2,$3,'generated',$2,NOW()-INTERVAL '20 days'),
		      ('30000000-0000-0000-0000-000000000502',$1,$2,$3,'completed',$2,NOW()-INTERVAL '20 days')`, gone, integrationUserID, integrationProjectID)

	repo := pgrepo.NewDashboardRepository(database)
	scope := query.ActorScope{ActorID: integrationUserID}
	risks, err := repo.ListRiskDocuments(ctx, integrationProjectID, scope)
	if err != nil {
		t.Fatal(err)
	}
	byType := map[string][]query.RiskDocumentItem{}
	for _, r := range risks {
		byType[r.RiskType] = append(byType[r.RiskType], r)
	}
	if len(byType["owner_inactive"]) != 1 || byType["owner_inactive"][0].DocumentID != orphan {
		t.Fatalf("owner_inactive: %#v", byType["owner_inactive"])
	}
	if len(byType["transfer_unaccepted"]) != 1 || byType["transfer_unaccepted"][0].DocumentID != unaccepted {
		t.Fatalf("transfer_unaccepted: %#v", byType["transfer_unaccepted"])
	}
	if len(byType["stale"]) != 1 || byType["stale"][0].DocumentID != stale {
		t.Fatalf("stale (finalized must be excluded): %#v", byType["stale"])
	}
	if len(byType["handover_overdue"]) != 1 || byType["handover_overdue"][0].Link != "/handovers?id=30000000-0000-0000-0000-000000000501" {
		t.Fatalf("handover_overdue (completed must be excluded): %#v", byType["handover_overdue"])
	}
	if risks[0].RiskType != "owner_inactive" {
		t.Fatalf("most severe first, got %s", risks[0].RiskType)
	}
	for _, r := range risks {
		if r.DocumentID == fresh || r.DocumentID == recentTransfer {
			t.Fatalf("healthy document reported: %#v", r)
		}
	}
	overview, err := repo.GetOverview(ctx, integrationProjectID, scope)
	if err != nil || overview.RiskDocumentCount != len(risks) || overview.HandoverPendingCount != 1 {
		t.Fatalf("overview must agree with list: %#v list=%d err=%v", overview, len(risks), err)
	}
}
