package service

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/domain/task"
	"digidocs-mgt/backend-go/internal/repository"
	"digidocs-mgt/backend-go/internal/storage"
)

type mockTaskPublisher struct {
	message task.Message
	err     error
}

func (m *mockTaskPublisher) Publish(_ context.Context, message task.Message) error {
	m.message = message
	return m.err
}

func TestTaskServicePublishBuildsMessageAndDefaultsPayload(t *testing.T) {
	publisher := &mockTaskPublisher{}
	svc := NewTaskService(publisher)
	message, err := svc.Publish(context.Background(), task.TaskTypeAssistantAsk, "document", "doc-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if message.RequestID == "" || message.TaskType != task.TaskTypeAssistantAsk || message.Payload == nil {
		t.Fatalf("message = %#v, want request id, task type and default payload", message)
	}
	if publisher.message.RequestID != message.RequestID {
		t.Fatalf("published message = %#v, want returned message", publisher.message)
	}
}

func TestTaskServicePublishPropagatesPublisherError(t *testing.T) {
	svc := NewTaskService(&mockTaskPublisher{err: errors.New("queue down")})
	if _, err := svc.Publish(context.Background(), task.TaskTypeAssistantAsk, "", "", map[string]any{}); err == nil {
		t.Fatal("expected publish error, got nil")
	}
}

func TestAuditServiceRecord(t *testing.T) {
	if err := NewAuditService().Record(context.Background(), command.AuditEventInput{
		ActionType: "download", UserID: "u-1", DocumentID: "doc-1", Extra: map[string]any{"k": "v"},
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

type recordingAuditWriter struct {
	events []command.AuditEventInput
	err    error
}

func (w *recordingAuditWriter) RecordAuditEvent(_ context.Context, e command.AuditEventInput) error {
	w.events = append(w.events, e)
	return w.err
}

func TestAuditServiceRecordDelegatesAndDefaultsViewDedup(t *testing.T) {
	w := &recordingAuditWriter{}
	svc := NewAuditService(w)
	if err := svc.Record(context.Background(), command.AuditEventInput{ActionType: AuditActionView, UserID: "u", DocumentID: "d"}); err != nil {
		t.Fatal(err)
	}
	if len(w.events) != 1 || w.events[0].DedupWindowSeconds != viewDedupWindowSeconds {
		t.Fatalf("view should default to dedup window, got %+v", w.events)
	}
	if err := svc.Record(context.Background(), command.AuditEventInput{ActionType: AuditActionDownload}); err != nil {
		t.Fatal(err)
	}
	if w.events[1].DedupWindowSeconds != 0 {
		t.Fatal("download must never be deduplicated")
	}
}

func TestAuditServiceRecordPropagatesWriterError(t *testing.T) {
	svc := NewAuditService(&recordingAuditWriter{err: errors.New("db down")})
	if err := svc.Record(context.Background(), command.AuditEventInput{ActionType: AuditActionDownload}); err == nil {
		t.Fatal("download audit failure must surface so the handler can fail closed")
	}
	if err := svc.Record(context.Background(), command.AuditEventInput{}); err == nil {
		t.Fatal("empty action must be rejected")
	}
}

type mockQueryRepos struct{}

func (m mockQueryRepos) ListTeamSpaces(_ context.Context, actorID, actorRole string) ([]query.TeamSpaceSummary, error) {
	return []query.TeamSpaceSummary{{ID: actorID + "-" + actorRole, Name: "Lab"}}, nil
}

func (m mockQueryRepos) ListUsers(_ context.Context) ([]query.UserOption, error) {
	return []query.UserOption{{ID: "u-1", Username: "zhangsan"}}, nil
}

func (m mockQueryRepos) ListProjects(_ context.Context, teamSpaceID, actorID, actorRole string) ([]query.ProjectSummary, error) {
	return []query.ProjectSummary{{ID: "p-1", TeamSpaceID: teamSpaceID, Owner: query.UserSummary{ID: actorID, DisplayName: actorRole}}}, nil
}

func (m mockQueryRepos) GetFolderTree(_ context.Context, projectID string) ([]query.FolderTreeNode, error) {
	return []query.FolderTreeNode{{ID: "f-1", Name: projectID}}, nil
}

func TestQueryServiceDelegates(t *testing.T) {
	repos := mockQueryRepos{}
	svc := NewQueryService(repos, repos, repos)
	if got, _ := svc.ListTeamSpaces(context.Background(), "u-1", "admin"); len(got) != 1 || got[0].ID != "u-1-admin" {
		t.Fatalf("ListTeamSpaces = %#v", got)
	}
	if got, _ := svc.ListUsers(context.Background()); len(got) != 1 || got[0].Username != "zhangsan" {
		t.Fatalf("ListUsers = %#v", got)
	}
	if got, _ := svc.ListProjects(context.Background(), "ts-1", "u-1", "admin"); len(got) != 1 || got[0].TeamSpaceID != "ts-1" {
		t.Fatalf("ListProjects = %#v", got)
	}
	if got, _ := svc.GetFolderTree(context.Background(), "p-1"); len(got) != 1 || got[0].Name != "p-1" {
		t.Fatalf("GetFolderTree = %#v", got)
	}
}

type mockPermissionReader struct {
	ok  bool
	err error
}

func (m mockPermissionReader) CanCreateDocument(context.Context, string, string, string) (bool, error) {
	return m.ok, m.err
}
func (m mockPermissionReader) CanUpdateDocument(context.Context, string, string, string) (bool, error) {
	return m.ok, m.err
}
func (m mockPermissionReader) CanDeleteDocument(context.Context, string, string, string) (bool, error) {
	return m.ok, m.err
}
func (m mockPermissionReader) CanUploadVersion(context.Context, string, string, string) (bool, error) {
	return m.ok, m.err
}
func (m mockPermissionReader) CanFlowDocument(context.Context, string, string, string, string) (bool, error) {
	return m.ok, m.err
}
func (m mockPermissionReader) CanCreateHandover(context.Context, string, string, string) (bool, error) {
	return m.ok, m.err
}
func (m mockPermissionReader) CanUpdateHandoverItems(context.Context, string, string, string) (bool, error) {
	return m.ok, m.err
}
func (m mockPermissionReader) CanApplyHandover(context.Context, string, string, string, string) (bool, error) {
	return m.ok, m.err
}
func (m mockPermissionReader) CanUploadDataAsset(context.Context, string, string, string) (bool, error) {
	return m.ok, m.err
}
func (m mockPermissionReader) CanManageDataAsset(context.Context, string, string, string) (bool, error) {
	return m.ok, m.err
}
func (m mockPermissionReader) CanViewDocument(context.Context, string, string, string) (bool, error) {
	return m.ok, m.err
}
func (m mockPermissionReader) CanViewVersion(context.Context, string, string, string) (bool, error) {
	return m.ok, m.err
}
func (m mockPermissionReader) CanViewHandover(context.Context, string, string, string) (bool, error) {
	return m.ok, m.err
}
func (m mockPermissionReader) CanViewProject(context.Context, string, string, string) (bool, error) {
	return m.ok, m.err
}
func (m mockPermissionReader) CanViewDataAsset(context.Context, string, string, string) (bool, error) {
	return m.ok, m.err
}
func (m mockPermissionReader) CanViewCodeRepository(context.Context, string, string, string) (bool, error) {
	return m.ok, m.err
}
func (m mockPermissionReader) CanCreateCodeRepository(context.Context, string, string, string) (bool, error) {
	return m.ok, m.err
}
func (m mockPermissionReader) CanManageCodeRepository(context.Context, string, string, string) (bool, error) {
	return m.ok, m.err
}
func (m mockPermissionReader) CanPushCodeRepository(context.Context, string, string, string) (bool, error) {
	return m.ok, m.err
}
func (m mockPermissionReader) CanViewProjectMembers(context.Context, string, string, string) (bool, error) {
	return m.ok, m.err
}
func (m mockPermissionReader) CanManageProjectMembers(context.Context, string, string, string) (bool, error) {
	return m.ok, m.err
}
func (m mockPermissionReader) CanViewAssistantRequest(context.Context, string, string, string) (bool, error) {
	return m.ok, m.err
}
func (m mockPermissionReader) CanViewAssistantSuggestion(context.Context, string, string, string) (bool, error) {
	return m.ok, m.err
}
func (m mockPermissionReader) CanViewConversation(context.Context, string, string, string) (bool, error) {
	return m.ok, m.err
}

func TestPermissionServiceAllowsNilReaderAndGrantedActions(t *testing.T) {
	if err := (PermissionService{}).EnsureCreateDocument(context.Background(), "u", "member", "p"); err != nil {
		t.Fatalf("nil reader should allow action: %v", err)
	}
	svc := NewPermissionService(mockPermissionReader{ok: true})
	checks := []func() error{
		func() error { return svc.EnsureCreateDocument(context.Background(), "u", "member", "p") },
		func() error { return svc.EnsureUpdateDocument(context.Background(), "u", "member", "d") },
		func() error { return svc.EnsureDeleteDocument(context.Background(), "u", "member", "d") },
		func() error { return svc.EnsureUploadVersion(context.Background(), "u", "member", "d") },
		func() error { return svc.EnsureFlowDocument(context.Background(), "u", "member", "d", "transfer") },
		func() error { return svc.EnsureCreateHandover(context.Background(), "u", "member", "p") },
		func() error { return svc.EnsureUpdateHandoverItems(context.Background(), "u", "member", "h") },
		func() error { return svc.EnsureApplyHandover(context.Background(), "u", "member", "h", "confirm") },
		func() error { return svc.EnsureUploadDataAsset(context.Background(), "u", "member", "p") },
		func() error { return svc.EnsureManageDataAsset(context.Background(), "u", "member", "a") },
		func() error { return svc.EnsureCreateCodeRepository(context.Background(), "u", "member", "p") },
		func() error { return svc.EnsureManageCodeRepository(context.Background(), "u", "member", "r") },
		func() error { return svc.EnsurePushCodeRepository(context.Background(), "u", "member", "r") },
		func() error { return svc.EnsureViewProjectMembers(context.Background(), "u", "member", "p") },
		func() error { return svc.EnsureManageProjectMembers(context.Background(), "u", "member", "p") },
	}
	for _, check := range checks {
		if err := check(); err != nil {
			t.Fatalf("granted permission returned error: %v", err)
		}
	}
}

func TestPermissionServiceDeniedAndReaderError(t *testing.T) {
	if err := NewPermissionService(mockPermissionReader{ok: false}).EnsurePushCodeRepository(context.Background(), "u", "member", "r"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("denied err = %v, want ErrForbidden", err)
	}
	readerErr := errors.New("db down")
	if err := NewPermissionService(mockPermissionReader{err: readerErr}).EnsurePushCodeRepository(context.Background(), "u", "member", "r"); !errors.Is(err, readerErr) {
		t.Fatalf("reader err = %v, want db down", err)
	}
}

type mockDataAssetRepo struct {
	asset       *query.DataAssetDetail
	folders     []query.DataFolderItem
	assets      []query.DataAssetListItem
	handover    []query.HandoverDataLine
	result      map[string]any
	err         error
	deletedID   string
	updateInput command.DataAssetUpdateInput
}

func (m *mockDataAssetRepo) ListDataAssets(context.Context, query.DataAssetListFilter) ([]query.DataAssetListItem, int, error) {
	return m.assets, len(m.assets), m.err
}
func (m *mockDataAssetRepo) GetDataAsset(context.Context, string) (*query.DataAssetDetail, error) {
	return m.asset, m.err
}
func (m *mockDataAssetRepo) ListDataFolders(context.Context, string) ([]query.DataFolderItem, error) {
	return m.folders, m.err
}
func (m *mockDataAssetRepo) ListHandoverDataItems(context.Context, string) ([]query.HandoverDataLine, error) {
	return m.handover, m.err
}
func (m *mockDataAssetRepo) CreateDataAsset(context.Context, command.DataAssetCreateInput) (map[string]any, error) {
	return m.result, m.err
}
func (m *mockDataAssetRepo) UpdateDataAsset(_ context.Context, input command.DataAssetUpdateInput) error {
	m.updateInput = input
	return m.err
}
func (m *mockDataAssetRepo) DeleteDataAsset(context.Context, command.DataAssetDeleteInput) error {
	return m.err
}
func (m *mockDataAssetRepo) CreateDataFolder(_ context.Context, input command.DataFolderCreateInput) (*query.DataFolderItem, error) {
	return &query.DataFolderItem{ID: "folder-1", ProjectID: input.ProjectID, Name: input.Name}, m.err
}
func (m *mockDataAssetRepo) DeleteDataFolder(_ context.Context, id string) error {
	m.deletedID = id
	return m.err
}
func (m *mockDataAssetRepo) UpdateHandoverDataItems(context.Context, command.HandoverDataItemUpdateInput) (map[string]any, error) {
	return m.result, m.err
}

func TestDataAssetServiceFoldersAndValidation(t *testing.T) {
	repo := &mockDataAssetRepo{folders: []query.DataFolderItem{{ID: "folder-1"}}, result: map[string]any{"ok": true}}
	svc := NewDataAssetService(repo, repo, &mockStorageProvider{})
	if got, err := svc.ListDataFolders(context.Background(), "p-1", "u-1", "admin"); err != nil || len(got) != 1 {
		t.Fatalf("ListDataFolders = %#v err=%v", got, err)
	}
	if _, err := svc.ListDataFolders(context.Background(), "", "u-1", "admin"); !errors.Is(err, ErrValidation) {
		t.Fatalf("empty project err = %v, want ErrValidation", err)
	}
	folder, err := svc.CreateDataFolder(context.Background(), command.DataFolderCreateInput{ProjectID: "p-1", Name: "raw"})
	if err != nil || folder.ID != "folder-1" {
		t.Fatalf("CreateDataFolder = %#v err=%v", folder, err)
	}
	if _, err := svc.CreateDataFolder(context.Background(), command.DataFolderCreateInput{ProjectID: "p-1"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("missing name err = %v, want ErrValidation", err)
	}
	if err := svc.DeleteDataFolder(context.Background(), "folder-1", "u-1", "admin"); err != nil || repo.deletedID != "folder-1" {
		t.Fatalf("DeleteDataFolder err=%v deletedID=%q", err, repo.deletedID)
	}
}

func TestDataAssetServiceUploadDownloadAndMutations(t *testing.T) {
	repo := &mockDataAssetRepo{
		asset:  &query.DataAssetDetail{ID: "asset-1", StorageObjectKey: "data/p-1/file.txt", FileName: "file.txt"},
		result: map[string]any{"id": "asset-1"},
	}
	storageProvider := &mockStorageProvider{
		result:    storage.PutObjectResult{ObjectKey: "data/p-1/tmp/file.txt", Provider: "memory"},
		getObject: &storage.GetObjectOutput{Reader: io.NopCloser(strings.NewReader("hello")), Size: 5},
	}
	svc := NewDataAssetService(repo, repo, storageProvider)
	created, err := svc.UploadDataAsset(context.Background(), command.DataAssetCreateInput{ProjectID: "p-1", DisplayName: "File"}, strings.NewReader("hello"), "file.txt")
	if err != nil || created["id"] != "asset-1" {
		t.Fatalf("UploadDataAsset = %#v err=%v", created, err)
	}
	if _, err := svc.UploadDataAsset(context.Background(), command.DataAssetCreateInput{DisplayName: "File"}, strings.NewReader("hello"), "file.txt"); !errors.Is(err, ErrValidation) {
		t.Fatalf("missing project err = %v, want ErrValidation", err)
	}
	if err := svc.UpdateDataAsset(context.Background(), command.DataAssetUpdateInput{DataAssetID: "asset-1", DisplayName: "New"}); err != nil || repo.updateInput.DisplayName != "New" {
		t.Fatalf("UpdateDataAsset err=%v input=%#v", err, repo.updateInput)
	}
	if err := svc.UpdateDataAsset(context.Background(), command.DataAssetUpdateInput{DataAssetID: "asset-1"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("empty update err = %v, want ErrValidation", err)
	}
	if err := svc.DeleteDataAsset(context.Background(), command.DataAssetDeleteInput{DataAssetID: "asset-1"}); err != nil {
		t.Fatalf("DeleteDataAsset unexpected err=%v", err)
	}
	out, asset, err := svc.DownloadDataAsset(context.Background(), "asset-1", "user-1", "member")
	if err != nil || asset.ID != "asset-1" || out.Size != 5 {
		t.Fatalf("DownloadDataAsset out=%#v asset=%#v err=%v", out, asset, err)
	}
	defer out.Reader.Close()
}

func TestDataAssetServiceHandoverDataItems(t *testing.T) {
	repo := &mockDataAssetRepo{
		handover: []query.HandoverDataLine{{DataAssetID: "asset-1"}},
		result:   map[string]any{"updated": true},
	}
	svc := NewDataAssetService(repo, repo, &mockStorageProvider{})
	if got, err := svc.ListHandoverDataItems(context.Background(), "h-1", "u-1", "admin"); err != nil || len(got) != 1 {
		t.Fatalf("ListHandoverDataItems = %#v err=%v", got, err)
	}
	if _, err := svc.ListHandoverDataItems(context.Background(), "", "u-1", "admin"); !errors.Is(err, ErrValidation) {
		t.Fatalf("empty handover err = %v, want ErrValidation", err)
	}
	if got, err := svc.UpdateHandoverDataItems(context.Background(), command.HandoverDataItemUpdateInput{HandoverID: "h-1"}); err != nil || got["updated"] != true {
		t.Fatalf("UpdateHandoverDataItems = %#v err=%v", got, err)
	}
	if _, err := svc.UpdateHandoverDataItems(context.Background(), command.HandoverDataItemUpdateInput{}); !errors.Is(err, ErrValidation) {
		t.Fatalf("empty update handover err = %v, want ErrValidation", err)
	}
}

func (m *mockDataAssetRepo) GetDataFolder(context.Context, string) (*query.DataFolderItem, error) {
	return &query.DataFolderItem{ID: "folder-1", ProjectID: "p-1"}, nil
}

type auditFilterSpy struct{ got query.AuditEventFilter }

func (s *auditFilterSpy) ListAuditEvents(_ context.Context, f query.AuditEventFilter) ([]query.AuditEventItem, int, error) {
	s.got = f
	return nil, 0, nil
}
func (s *auditFilterSpy) GetAuditSummary(context.Context, string, query.ActorScope) (query.AuditSummary, error) {
	return query.AuditSummary{}, nil
}

type docViewPermission struct {
	repository.PermissionReader
	allowed string
}

func (p docViewPermission) CanViewDocument(_ context.Context, _ string, _ string, documentID string) (bool, error) {
	return documentID == p.allowed, nil
}

func TestAuditQueryWidensScopeOnlyForViewableDocument(t *testing.T) {
	spy := &auditFilterSpy{}
	svc := NewAuditQueryService(spy).WithPermissions(NewPermissionService(docViewPermission{allowed: "doc-ok"}))
	member := query.ActorScope{ActorID: "u"}
	_, _, _ = svc.List(context.Background(), query.AuditEventFilter{DocumentID: "doc-ok", Scope: member})
	if !spy.got.Scope.IsAdmin || spy.got.DocumentID != "doc-ok" {
		t.Fatalf("viewable document should widen scope but stay pinned: %#v", spy.got)
	}
	_, _, _ = svc.List(context.Background(), query.AuditEventFilter{DocumentID: "doc-no", Scope: member})
	if spy.got.Scope.IsAdmin {
		t.Fatal("non-viewable document must keep project scoping")
	}
	_, _, _ = svc.List(context.Background(), query.AuditEventFilter{Scope: member})
	if spy.got.Scope.IsAdmin {
		t.Fatal("no document filter must keep project scoping")
	}
}

func TestValidFlowTransitionMatchesProductRules(t *testing.T) {
	cases := map[[2]string]bool{
		{"handed_over", "finalize"}:         true,
		{"handed_over", "mark_in_progress"}: true,
		{"handed_over", "archive"}:          true,
		{"draft", "archive"}:                false,
		{"pending_handover", "finalize"}:    false,
		{"archived", "unarchive"}:           true,
	}
	for c, want := range cases {
		if got := ValidFlowTransition(c[0], c[1]); got != want {
			t.Fatalf("%s/%s = %v, want %v", c[0], c[1], got, want)
		}
	}
}
