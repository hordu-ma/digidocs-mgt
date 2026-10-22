package postgres_test

import (
	"context"
	"errors"
	"testing"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/task"
	noopqueue "digidocs-mgt/backend-go/internal/queue/noop"
	pgrepo "digidocs-mgt/backend-go/internal/repository/postgres"
	"digidocs-mgt/backend-go/internal/service"
)

func TestPostgresIntegrationReviewSecurityAndVersionIsolation(t *testing.T) {
	ctx := context.Background()
	database := openIntegrationDB(t)
	seedIntegrationBaseRows(t, ctx, database)
	const otherUser = "00000000-0000-0000-0000-000000000102"
	const otherProject = "20000000-0000-0000-0000-000000000102"
	const handoverID = "30000000-0000-0000-0000-000000000101"
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := database.ExecContext(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO users(id,username,password_hash,display_name,role,status) VALUES($1,'other','hash','other','member','active')`, otherUser)
	exec(`INSERT INTO projects(id,team_space_id,name,code,owner_id) VALUES($1,$2,'other','other',$3)`, otherProject, integrationTeamSpaceID, otherUser)
	permissions := service.NewPermissionService(pgrepo.NewPermissionRepository(database))
	members := pgrepo.NewMembershipRepository(database)
	membership := service.NewMembershipService(members, permissions)
	member, err := members.AddProjectMember(ctx, otherProject, integrationUserID, "contributor")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = membership.UpdateMember(ctx, integrationUserID, "member", integrationProjectID, member.ID, "manager"); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("cross-project member update: %v", err)
	}
	if err = membership.RemoveMember(ctx, integrationUserID, "member", integrationProjectID, member.ID); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("cross-project member removal: %v", err)
	}
	unchanged, err := members.GetProjectMember(ctx, otherProject, member.ID)
	if err != nil || unchanged.ProjectRole != "contributor" {
		t.Fatalf("membership changed: %#v %v", unchanged, err)
	}
	owner, err := members.AddProjectMember(ctx, integrationProjectID, integrationUserID, "owner")
	if err != nil {
		t.Fatal(err)
	}
	if err = members.RemoveProjectMember(ctx, integrationProjectID, owner.ID); !errors.Is(err, service.ErrValidation) {
		t.Fatalf("last owner removed: %v", err)
	}
	if _, err = members.UpdateProjectMember(ctx, integrationProjectID, owner.ID, "viewer"); !errors.Is(err, service.ErrValidation) {
		t.Fatalf("last owner demoted: %v", err)
	}

	assets := pgrepo.NewDataAssetRepository(database)
	dataService := service.NewDataAssetService(assets, assets, nil, permissions)
	folder, err := assets.CreateDataFolder(ctx, command.DataFolderCreateInput{ProjectID: integrationProjectID, Name: "protected", ActorID: integrationUserID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = dataService.ListDataFolders(ctx, integrationProjectID, otherUser, "member"); !errors.Is(err, service.ErrForbidden) {
		t.Fatalf("folder read: %v", err)
	}
	if err = dataService.DeleteDataFolder(ctx, folder.ID, otherUser, "member"); !errors.Is(err, service.ErrForbidden) {
		t.Fatalf("folder deletion: %v", err)
	}
	if _, err = assets.GetDataFolder(ctx, folder.ID); err != nil {
		t.Fatal("denied deletion removed folder", err)
	}

	createAsset := func(project string) string {
		t.Helper()
		item, err := assets.CreateDataAsset(ctx, command.DataAssetCreateInput{TeamSpaceID: integrationTeamSpaceID, ProjectID: project, DisplayName: "data", FileName: "data.txt", FileSize: 1, StorageProvider: "memory", StorageObjectKey: "test", ActorID: integrationUserID})
		if err != nil {
			t.Fatal(err)
		}
		return item["id"].(string)
	}
	assetID := createAsset(integrationProjectID)
	foreignAsset := createAsset(otherProject)
	exec(`INSERT INTO graduation_handovers(id,target_user_id,receiver_user_id,project_id,status,generated_by) VALUES($1,$2,$2,$3,'generated',$2)`, handoverID, integrationUserID, integrationProjectID)
	input := command.HandoverDataItemUpdateInput{HandoverID: handoverID, ActorID: integrationUserID, ActorRole: "member", Items: []command.HandoverDataItemInput{{DataAssetID: assetID, Selected: true}}}
	if _, err = dataService.UpdateHandoverDataItems(ctx, input); err != nil {
		t.Fatal("valid handover edit", err)
	}
	input.ActorID = otherUser
	if _, err = dataService.UpdateHandoverDataItems(ctx, input); !errors.Is(err, service.ErrForbidden) {
		t.Fatalf("foreign handover write: %v", err)
	}
	if _, err = dataService.ListHandoverDataItems(ctx, handoverID, otherUser, "member"); !errors.Is(err, service.ErrForbidden) {
		t.Fatalf("foreign handover read: %v", err)
	}
	input.ActorID = integrationUserID
	input.Items = []command.HandoverDataItemInput{{DataAssetID: foreignAsset, Selected: true}}
	if _, err = dataService.UpdateHandoverDataItems(ctx, input); !errors.Is(err, service.ErrValidation) {
		t.Fatalf("foreign asset accepted: %v", err)
	}
	exec(`UPDATE graduation_handovers SET status='completed' WHERE id=$1`, handoverID)
	input.Items = nil
	if _, err = dataService.UpdateHandoverDataItems(ctx, input); !errors.Is(err, service.ErrValidation) {
		t.Fatalf("completed handover editable: %v", err)
	}
	items, err := assets.ListHandoverDataItems(ctx, handoverID)
	if err != nil || len(items) != 1 || items[0].DataAssetID != assetID {
		t.Fatalf("denied edit changed items: %#v %v", items, err)
	}
	var audits int
	if err = database.QueryRow(`SELECT COUNT(*) FROM audit_events WHERE extra_data->>'handover_id'=$1`, handoverID).Scan(&audits); err != nil || audits != 1 {
		t.Fatalf("audit count %d: %v", audits, err)
	}

	documents := pgrepo.NewDocumentRepository(database)
	versions := pgrepo.NewVersionRepository(database)
	workflow := pgrepo.NewVersionWorkflow(database)
	createDoc := func(project string) string {
		t.Helper()
		d, err := documents.CreateDocument(ctx, command.DocumentCreateInput{TeamSpaceID: integrationTeamSpaceID, ProjectID: project, Title: "test", CurrentOwnerID: integrationUserID, ActorID: integrationUserID})
		if err != nil {
			t.Fatal(err)
		}
		return d["id"].(string)
	}
	docID := createDoc(integrationProjectID)
	foreignDoc := createDoc(otherProject)
	createVersion := func(doc string) string {
		t.Helper()
		v, err := workflow.CreateUploadedVersion(ctx, command.VersionCreateInput{DocumentID: doc, FileName: "v.txt", FileSize: 1, StorageProvider: "memory", StorageObjectKey: "test", ActorID: integrationUserID})
		if err != nil {
			t.Fatal(err)
		}
		return v["id"].(string)
	}
	v1, v2, foreignVersion := createVersion(docID), createVersion(docID), createVersion(foreignDoc)
	assistantRepo := pgrepo.NewAssistantRepository(database)
	assistant := service.NewAssistantService(noopqueue.NewPublisher(), assistantRepo, documents, versions, permissions)
	if _, err = assistant.SummarizeDocument(ctx, docID, map[string]any{"version_id": foreignVersion}, integrationUserID, "member"); !errors.Is(err, service.ErrValidation) {
		t.Fatalf("cross-document summary accepted: %v", err)
	}
	if _, err = assistant.Ask(ctx, map[string]any{"question": "q", "scope": map[string]any{"document_id": docID, "project_id": otherProject}}, integrationUserID, "member"); !errors.Is(err, service.ErrValidation) {
		t.Fatalf("mismatched ask scope: %v", err)
	}
	conversation, err := assistantRepo.CreateConversation(ctx, "project", integrationProjectID, map[string]any{"project_id": integrationProjectID}, "private", integrationUserID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = members.AddProjectMember(ctx, integrationProjectID, otherUser, "contributor"); err != nil {
		t.Fatal(err)
	}
	if _, err = assistant.Ask(ctx, map[string]any{"question": "q", "conversation_id": conversation.ID}, otherUser, "member"); !errors.Is(err, service.ErrForbidden) {
		t.Fatalf("other user's conversation: %v", err)
	}
	message, err := assistant.QueueTask(ctx, task.TaskTypeDocumentExtractText, "document", docID, map[string]any{"version_id": v1}, integrationUserID)
	if err != nil {
		t.Fatal(err)
	}
	if err = assistant.ReceiveResult(ctx, task.Result{RequestID: message.RequestID, Status: "completed", Output: map[string]any{"extracted_text": "v1 body", "version_id": v1}}); err != nil {
		t.Fatal(err)
	}
	if text, err := assistant.GetDocumentVersionExtractedText(ctx, docID, v1); err != nil || text != "v1 body" {
		t.Fatalf("v1 cache %q %v", text, err)
	}
	if text, err := assistant.GetDocumentVersionExtractedText(ctx, docID, v2); err != nil || text != "" {
		t.Fatalf("v2 reused v1: %q %v", text, err)
	}
	if text, err := assistant.GetDocumentVersionExtractedText(ctx, foreignDoc, v1); err != nil || text != "" {
		t.Fatalf("foreign document cache: %q %v", text, err)
	}
}
