package memory

import (
	"context"
	"errors"
	"testing"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/service"
)

func TestDocumentRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewDocumentRepository()

	docs, total, err := repo.ListDocuments(ctx, query.DocumentListFilter{})
	if err != nil || total != 1 || len(docs) != 1 {
		t.Fatalf("list documents = (%+v, %d, %v)", docs, total, err)
	}

	detail, err := repo.GetDocument(ctx, "doc-1")
	if err != nil || detail.ID != "doc-1" {
		t.Fatalf("get document = (%+v, %v)", detail, err)
	}

	created, err := repo.CreateDocument(ctx, command.DocumentCreateInput{Title: "New"})
	if err != nil || created["title"] != "New" {
		t.Fatalf("create document = (%+v, %v)", created, err)
	}

	updated, err := repo.UpdateDocument(ctx, command.DocumentUpdateInput{DocumentID: "doc-1"})
	if err != nil || updated["title"] != "课题申报书" {
		t.Fatalf("update document default title = (%+v, %v)", updated, err)
	}
	updated, err = repo.UpdateDocument(ctx, command.DocumentUpdateInput{DocumentID: "doc-1", Title: "Updated"})
	if err != nil || updated["title"] != "Updated" {
		t.Fatalf("update document title = (%+v, %v)", updated, err)
	}

	if err := repo.DeleteDocument(ctx, command.DocumentDeleteInput{DocumentID: "doc-1"}); err != nil {
		t.Fatalf("delete document: %v", err)
	}
	if err := repo.RestoreDocument(ctx, "doc-1", "actor-1"); err != nil {
		t.Fatalf("restore document: %v", err)
	}
}

func TestVersionRepositoryAndWorkflow(t *testing.T) {
	ctx := context.Background()
	repo := NewVersionRepository()
	workflow := NewVersionWorkflow(repo)

	seeded, err := repo.ListVersions(ctx, "00000000-0000-0000-0000-000000000100")
	if err != nil || len(seeded) != 1 {
		t.Fatalf("seeded versions = (%+v, %v)", seeded, err)
	}

	result, err := workflow.CreateUploadedVersion(ctx, command.VersionCreateInput{
		DocumentID:       "doc-1",
		CommitMessage:    "upload",
		FileName:         "report.pdf",
		FileSize:         42,
		StorageProvider:  "memory",
		StorageObjectKey: "documents/doc-1/report.pdf",
		ActorID:          "actor-1",
	})
	if err != nil {
		t.Fatalf("create uploaded version: %v", err)
	}
	versionID := result["id"].(string)
	if result["version_no"].(int) != 1 {
		t.Fatalf("unexpected version result: %+v", result)
	}

	detail, err := repo.GetVersion(ctx, versionID)
	if err != nil || detail.FileName != "report.pdf" || detail.FileSize != 42 {
		t.Fatalf("get created version = (%+v, %v)", detail, err)
	}
	items, err := repo.ListVersions(ctx, "doc-1")
	if err != nil || len(items) != 1 || items[0].ID != versionID {
		t.Fatalf("list created versions = (%+v, %v)", items, err)
	}

	fallback, err := repo.GetVersion(ctx, "unknown")
	if err != nil || fallback.ID != "unknown" {
		t.Fatalf("fallback version = (%+v, %v)", fallback, err)
	}
}

func TestActionRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewActionRepository()

	for action, status := range map[string]string{
		"archive":          "archived",
		"finalize":         "finalized",
		"transfer":         "pending_handover",
		"accept_transfer":  "in_progress",
		"mark_in_progress": "in_progress",
		"unarchive":        "finalized",
	} {
		result, err := repo.CreateFlowRecord(ctx, command.FlowActionInput{
			DocumentID: "doc-1",
			Action:     action,
			ToUserID:   "user-2",
			Note:       "note",
		})
		if err != nil {
			t.Fatalf("flow action %s: %v", action, err)
		}
		if result["current_status"] != status {
			t.Fatalf("flow action %s status = %v, want %s", action, result["current_status"], status)
		}
	}
	if _, err := repo.CreateFlowRecord(ctx, command.FlowActionInput{Action: "bad"}); !errors.Is(err, service.ErrInvalidTransition) {
		t.Fatalf("expected invalid flow transition, got %v", err)
	}

	handover, err := repo.CreateHandover(ctx, command.HandoverCreateInput{
		ProjectID:      "project-1",
		TargetUserID:   "target",
		ReceiverUserID: "receiver",
		Remark:         "remark",
	})
	if err != nil || handover["status"] != "generated" {
		t.Fatalf("create handover = (%+v, %v)", handover, err)
	}
	items, err := repo.UpdateHandoverItems(ctx, command.HandoverItemUpdateInput{
		HandoverID: "handover-1",
		Items:      []command.HandoverItemInput{{DocumentID: "doc-1", Selected: true}},
	})
	if err != nil || items["id"] != "handover-1" {
		t.Fatalf("update handover items = (%+v, %v)", items, err)
	}
	applied, err := repo.ApplyHandover(ctx, command.HandoverActionInput{HandoverID: "handover-1", Action: "confirm", Note: "ok"})
	if err != nil || applied["action"] != "confirm" {
		t.Fatalf("apply handover = (%+v, %v)", applied, err)
	}
	if _, err := repo.ApplyHandover(ctx, command.HandoverActionInput{Action: "bad"}); !errors.Is(err, service.ErrInvalidTransition) {
		t.Fatalf("expected invalid handover transition, got %v", err)
	}
	if got := memFlowActionToStatus("unknown"); got != "in_progress" {
		t.Fatalf("unknown flow action status = %s", got)
	}
}

func TestReadOnlyMemoryRepositories(t *testing.T) {
	ctx := context.Background()

	overview, err := NewDashboardRepository().GetOverview(ctx, "project-1")
	if err != nil || overview.StatusCounts == nil {
		t.Fatalf("overview = (%+v, %v)", overview, err)
	}
	if flows, err := NewDashboardRepository().ListRecentFlows(ctx, "project-1"); err != nil || len(flows) != 0 {
		t.Fatalf("recent flows = (%+v, %v)", flows, err)
	}
	if risks, err := NewDashboardRepository().ListRiskDocuments(ctx, "project-1"); err != nil || len(risks) != 0 {
		t.Fatalf("risk documents = (%+v, %v)", risks, err)
	}

	events, total, err := NewAuditRepository().ListAuditEvents(ctx, query.AuditEventFilter{})
	if err != nil || total != 0 || len(events) != 0 {
		t.Fatalf("audit events = (%+v, %d, %v)", events, total, err)
	}
	summary, err := NewAuditRepository().GetAuditSummary(ctx, "project-1")
	if err != nil || summary.ProjectID != "project-1" {
		t.Fatalf("audit summary = (%+v, %v)", summary, err)
	}

	if spaces, err := NewTeamSpaceRepository().ListTeamSpaces(ctx, "actor", "admin"); err != nil || len(spaces) != 1 {
		t.Fatalf("team spaces = (%+v, %v)", spaces, err)
	}
	if users, err := NewUserQueryRepository().ListUsers(ctx); err != nil || len(users) < 1 {
		t.Fatalf("users = (%+v, %v)", users, err)
	}
	projects, err := NewProjectRepository().ListProjects(ctx, "", "actor", "admin")
	if err != nil || len(projects) != 1 || projects[0].TeamSpaceID == "" {
		t.Fatalf("projects = (%+v, %v)", projects, err)
	}
	projects, err = NewProjectRepository().ListProjects(ctx, "team-1", "actor", "admin")
	if err != nil || projects[0].TeamSpaceID != "team-1" {
		t.Fatalf("filtered projects = (%+v, %v)", projects, err)
	}
	if folders, err := NewProjectRepository().GetFolderTree(ctx, "project-1"); err != nil || len(folders) != 1 {
		t.Fatalf("folder tree = (%+v, %v)", folders, err)
	}
	if flows, err := NewFlowRepository().ListFlows(ctx, "doc-1"); err != nil || len(flows) != 1 {
		t.Fatalf("flows = (%+v, %v)", flows, err)
	}
	if handovers, err := NewHandoverRepository().ListHandovers(ctx); err != nil || len(handovers) != 1 {
		t.Fatalf("handovers = (%+v, %v)", handovers, err)
	}
	if handover, err := NewHandoverRepository().GetHandover(ctx, "handover-1"); err != nil || handover.ID != "handover-1" {
		t.Fatalf("handover = (%+v, %v)", handover, err)
	}
}

func TestPermissionRepositoryAlwaysAllows(t *testing.T) {
	ctx := context.Background()
	repo := NewPermissionRepository()
	checks := []func() (bool, error){
		func() (bool, error) { return repo.CanCreateDocument(ctx, "", "", "") },
		func() (bool, error) { return repo.CanUpdateDocument(ctx, "", "", "") },
		func() (bool, error) { return repo.CanDeleteDocument(ctx, "", "", "") },
		func() (bool, error) { return repo.CanUploadVersion(ctx, "", "", "") },
		func() (bool, error) { return repo.CanFlowDocument(ctx, "", "", "", "") },
		func() (bool, error) { return repo.CanCreateHandover(ctx, "", "", "") },
		func() (bool, error) { return repo.CanUpdateHandoverItems(ctx, "", "", "") },
		func() (bool, error) { return repo.CanApplyHandover(ctx, "", "", "", "") },
		func() (bool, error) { return repo.CanUploadDataAsset(ctx, "", "", "") },
		func() (bool, error) { return repo.CanManageDataAsset(ctx, "", "", "") },
		func() (bool, error) { return repo.CanCreateCodeRepository(ctx, "", "", "") },
		func() (bool, error) { return repo.CanManageCodeRepository(ctx, "", "", "") },
		func() (bool, error) { return repo.CanPushCodeRepository(ctx, "", "", "") },
	}
	for idx, check := range checks {
		ok, err := check()
		if err != nil || !ok {
			t.Fatalf("check %d = (%v, %v)", idx, ok, err)
		}
	}
}

func TestDataAssetRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewDataAssetRepository()

	root, err := repo.CreateDataFolder(ctx, command.DataFolderCreateInput{ProjectID: "project-1", Name: "root"})
	if err != nil {
		t.Fatalf("create root folder: %v", err)
	}
	child, err := repo.CreateDataFolder(ctx, command.DataFolderCreateInput{ProjectID: "project-1", ParentID: root.ID, Name: "child"})
	if err != nil || child.Depth != 1 {
		t.Fatalf("create child folder = (%+v, %v)", child, err)
	}
	grandchild, err := repo.CreateDataFolder(ctx, command.DataFolderCreateInput{ProjectID: "project-1", ParentID: child.ID, Name: "grandchild"})
	if err != nil || grandchild.Depth != 2 {
		t.Fatalf("create grandchild folder = (%+v, %v)", grandchild, err)
	}
	if _, err := repo.CreateDataFolder(ctx, command.DataFolderCreateInput{ProjectID: "project-1", ParentID: grandchild.ID, Name: "too-deep"}); !errors.Is(err, service.ErrValidation) {
		t.Fatalf("expected depth validation error, got %v", err)
	}
	if _, err := repo.CreateDataFolder(ctx, command.DataFolderCreateInput{ProjectID: "project-1", ParentID: "missing", Name: "missing"}); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected missing parent error, got %v", err)
	}
	if folders, err := repo.ListDataFolders(ctx, "project-1"); err != nil || len(folders) != 3 {
		t.Fatalf("list folders = (%+v, %v)", folders, err)
	}

	created, err := repo.CreateDataAsset(ctx, command.DataAssetCreateInput{
		TeamSpaceID:      "team-1",
		ProjectID:        "project-1",
		FolderID:         child.ID,
		DisplayName:      "Dataset",
		FileName:         "dataset.bin",
		Description:      "desc",
		MimeType:         "application/octet-stream",
		FileSize:         10,
		StorageProvider:  "memory",
		StorageObjectKey: "data/dataset.bin",
		ActorID:          "actor-1",
	})
	if err != nil {
		t.Fatalf("create asset: %v", err)
	}
	assetID := created["id"].(string)
	if assets, total, err := repo.ListDataAssets(ctx, query.DataAssetListFilter{ProjectID: "project-1", FolderID: child.ID}); err != nil || total != 1 || assets[0].ID != assetID {
		t.Fatalf("list assets = (%+v, %d, %v)", assets, total, err)
	}
	detail, err := repo.GetDataAsset(ctx, assetID)
	if err != nil || detail.DisplayName != "Dataset" {
		t.Fatalf("get asset = (%+v, %v)", detail, err)
	}
	if err := repo.UpdateDataAsset(ctx, command.DataAssetUpdateInput{DataAssetID: assetID, DisplayName: "Updated", FolderID: root.ID}); err != nil {
		t.Fatalf("update asset: %v", err)
	}
	if err := repo.UpdateDataAsset(ctx, command.DataAssetUpdateInput{DataAssetID: "missing"}); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected update missing error, got %v", err)
	}
	result, err := repo.UpdateHandoverDataItems(ctx, command.HandoverDataItemUpdateInput{
		HandoverID: "handover-1",
		Items:      []command.HandoverDataItemInput{{DataAssetID: assetID, Selected: true, Note: "note"}},
	})
	if err != nil || result["count"] != 1 {
		t.Fatalf("update handover data items = (%+v, %v)", result, err)
	}
	if lines, err := repo.ListHandoverDataItems(ctx, "handover-1"); err != nil || len(lines) != 1 || !lines[0].Selected {
		t.Fatalf("list handover data items = (%+v, %v)", lines, err)
	}
	if err := repo.DeleteDataAsset(ctx, command.DataAssetDeleteInput{DataAssetID: assetID}); err != nil {
		t.Fatalf("delete asset: %v", err)
	}
	if _, err := repo.GetDataAsset(ctx, assetID); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected deleted asset not found, got %v", err)
	}
	if err := repo.DeleteDataAsset(ctx, command.DataAssetDeleteInput{DataAssetID: "missing"}); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected delete missing asset error, got %v", err)
	}
	if err := repo.DeleteDataFolder(ctx, root.ID); err != nil {
		t.Fatalf("delete folder: %v", err)
	}
	if err := repo.DeleteDataFolder(ctx, "missing"); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected delete missing folder error, got %v", err)
	}
}

func TestCodeRepositoryRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewCodeRepositoryRepository()

	created, err := repo.CreateCodeRepository(ctx, command.CodeRepositoryCreateInput{
		TeamSpaceID:      "team-1",
		ProjectID:        "project-1",
		Name:             "Analysis Repo",
		Slug:             "analysis",
		Description:      "desc",
		DefaultBranch:    "main",
		TargetFolderPath: "/code/analysis",
		RepoStoragePath:  "/tmp/repo.git",
		PushToken:        "push-token",
	})
	if err != nil {
		t.Fatalf("create code repo: %v", err)
	}
	if _, err := repo.CreateCodeRepository(ctx, command.CodeRepositoryCreateInput{Slug: "analysis"}); !errors.Is(err, service.ErrConflict) {
		t.Fatalf("expected duplicate slug conflict, got %v", err)
	}
	if items, total, err := repo.ListCodeRepositories(ctx, query.CodeRepositoryListFilter{ProjectID: "project-1", Keyword: "analysis"}); err != nil || total != 1 || items[0].ID != created.ID {
		t.Fatalf("list code repos = (%+v, %d, %v)", items, total, err)
	}
	detail, err := repo.GetCodeRepository(ctx, created.ID)
	if err != nil || detail.ID != created.ID {
		t.Fatalf("get code repo = (%+v, %v)", detail, err)
	}
	bySlug, err := repo.GetCodeRepositoryBySlug(ctx, "analysis")
	if err != nil || bySlug.PushToken != "push-token" {
		t.Fatalf("get code repo by slug = (%+v, %v)", bySlug, err)
	}
	updated, err := repo.UpdateCodeRepository(ctx, command.CodeRepositoryUpdateInput{
		RepositoryID:     created.ID,
		Name:             "Updated Repo",
		Description:      "updated",
		DefaultBranch:    "trunk",
		TargetFolderPath: "/code/updated",
	})
	if err != nil || updated.Name != "Updated Repo" || updated.DefaultBranch != "trunk" {
		t.Fatalf("update code repo = (%+v, %v)", updated, err)
	}
	if _, err := repo.UpdateCodeRepository(ctx, command.CodeRepositoryUpdateInput{RepositoryID: "missing"}); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected update missing error, got %v", err)
	}
	event, err := repo.CreateCodePushEvent(ctx, command.CodePushEventCreateInput{
		RepositoryID:  created.ID,
		Branch:        "main",
		BeforeSHA:     "before",
		AfterSHA:      "after",
		CommitMessage: "commit",
		SyncStatus:    "completed",
	})
	if err != nil || event.AfterSHA != "after" {
		t.Fatalf("create push event = (%+v, %v)", event, err)
	}
	if err := repo.UpdateCodeRepositoryAfterPush(ctx, created.ID, "after", "active"); err != nil {
		t.Fatalf("update after push: %v", err)
	}
	if err := repo.UpdateCodeRepositoryAfterPush(ctx, "missing", "after", "active"); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected update push missing error, got %v", err)
	}
	if events, err := repo.ListCodePushEvents(ctx, created.ID); err != nil || len(events) != 1 {
		t.Fatalf("list push events = (%+v, %v)", events, err)
	}
	if _, err := repo.GetCodeRepository(ctx, "missing"); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected get missing error, got %v", err)
	}
	if _, err := repo.GetCodeRepositoryBySlug(ctx, "missing"); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected get missing slug error, got %v", err)
	}
}
