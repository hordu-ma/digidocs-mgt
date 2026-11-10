package postgres_test

import (
	"context"
	"testing"

	"digidocs-mgt/backend-go/internal/domain/command"
	pgrepo "digidocs-mgt/backend-go/internal/repository/postgres"
)

func TestPostgresIntegrationHandoverCandidates(t *testing.T) {
	ctx := context.Background()
	database := openIntegrationDB(t)
	seedIntegrationBaseRows(t, ctx, database)
	const leaver = "00000000-0000-0000-0000-000000000401"
	const receiver = "00000000-0000-0000-0000-000000000402"
	const otherProject = "20000000-0000-0000-0000-000000000401"
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := database.ExecContext(ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO users(id,username,password_hash,display_name,role,status) VALUES($1,'leaver','h','leaver','member','active'),($2,'receiver','h','receiver','member','active')`, leaver, receiver)
	exec(`INSERT INTO projects(id,team_space_id,name,code,owner_id) VALUES($1,$2,'other','other',$3)`, otherProject, integrationTeamSpaceID, integrationUserID)

	documents := pgrepo.NewDocumentRepository(database)
	newDoc := func(project, owner string) string {
		t.Helper()
		d, err := documents.CreateDocument(ctx, command.DocumentCreateInput{TeamSpaceID: integrationTeamSpaceID, ProjectID: project, Title: "doc", CurrentOwnerID: owner, ActorID: owner})
		if err != nil {
			t.Fatal(err)
		}
		return d["id"].(string)
	}
	owned := newDoc(integrationProjectID, leaver)         // collected, note 未挂接目录 (no folder)
	stale := newDoc(integrationProjectID, leaver)         // collected
	archived := newDoc(integrationProjectID, leaver)      // skipped: archived
	deleted := newDoc(integrationProjectID, leaver)       // skipped: deleted
	foreign := newDoc(otherProject, leaver)               // skipped: other project
	someoneElse := newDoc(integrationProjectID, receiver) // skipped: not the leaver's
	exec(`UPDATE documents SET current_status='archived', is_archived=true WHERE id=$1`, archived)
	exec(`UPDATE documents SET is_deleted=true WHERE id=$1`, deleted)
	exec(`UPDATE documents SET updated_at=NOW()-INTERVAL '40 days' WHERE id=$1`, stale)
	_ = foreign
	_ = someoneElse

	assets := pgrepo.NewDataAssetRepository(database)
	newAsset := func(project, by string) string {
		t.Helper()
		a, err := assets.CreateDataAsset(ctx, command.DataAssetCreateInput{TeamSpaceID: integrationTeamSpaceID, ProjectID: project, DisplayName: "d", FileName: "d.csv", FileSize: 1, StorageProvider: "memory", StorageObjectKey: "k", ActorID: by})
		if err != nil {
			t.Fatal(err)
		}
		return a["id"].(string)
	}
	mine := newAsset(integrationProjectID, leaver)
	newAsset(otherProject, leaver)
	newAsset(integrationProjectID, receiver)

	actions := pgrepo.NewActionRepository(database)
	got, err := actions.CreateHandover(ctx, command.HandoverCreateInput{TargetUserID: leaver, ReceiverUserID: receiver, ProjectID: integrationProjectID, ActorID: integrationUserID})
	if err != nil {
		t.Fatal(err)
	}
	if got["document_item_count"] != 2 || got["data_item_count"] != 1 {
		t.Fatalf("candidate counts: %#v", got)
	}
	handoverID := got["id"].(string)
	notes := map[string]string{}
	rows, err := database.QueryContext(ctx, `SELECT document_id::text, COALESCE(note,'') FROM graduation_handover_items WHERE handover_id=$1 AND selected`, handoverID)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id, note string
		if err := rows.Scan(&id, &note); err != nil {
			t.Fatal(err)
		}
		notes[id] = note
	}
	rows.Close()
	if len(notes) != 2 || notes[owned] != "未挂接目录" || notes[stale] != "未挂接目录" {
		t.Fatalf("collected documents/notes: %#v", notes)
	}
	var dataID string
	if err := database.QueryRowContext(ctx, `SELECT data_asset_id::text FROM graduation_handover_data_items WHERE handover_id=$1`, handoverID).Scan(&dataID); err != nil || dataID != mine {
		t.Fatalf("collected data asset %s err=%v", dataID, err)
	}

	// Cross-project (admin, no project) collects everything the leaver holds.
	all, err := actions.CreateHandover(ctx, command.HandoverCreateInput{TargetUserID: leaver, ReceiverUserID: receiver, ActorID: integrationUserID})
	if err != nil || all["document_item_count"] != 3 || all["data_item_count"] != 2 {
		t.Fatalf("cross-project candidates: %#v %v", all, err)
	}

	// Handover action audit rows now carry the project for scoped audit views.
	if _, err := actions.ApplyHandover(ctx, command.HandoverActionInput{HandoverID: handoverID, Action: "confirm", ActorID: receiver}); err != nil {
		t.Fatal(err)
	}
	var project string
	if err := database.QueryRowContext(ctx, `SELECT extra_data->>'project_id' FROM audit_events WHERE action_type='handover_confirm' AND extra_data->>'handover_id'=$1`, handoverID).Scan(&project); err != nil || project != integrationProjectID {
		t.Fatalf("handover_confirm audit project=%q err=%v", project, err)
	}
}
