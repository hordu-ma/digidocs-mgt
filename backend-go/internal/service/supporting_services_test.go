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
	if err := NewAuditService().Record(context.Background(), "download", "u-1", "doc-1", map[string]any{"k": "v"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
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
func (m mockPermissionReader) CanCreateCodeRepository(context.Context, string, string, string) (bool, error) {
	return m.ok, m.err
}
func (m mockPermissionReader) CanManageCodeRepository(context.Context, string, string, string) (bool, error) {
	return m.ok, m.err
}
func (m mockPermissionReader) CanPushCodeRepository(context.Context, string, string, string) (bool, error) {
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
	if got, err := svc.ListDataFolders(context.Background(), "p-1"); err != nil || len(got) != 1 {
		t.Fatalf("ListDataFolders = %#v err=%v", got, err)
	}
	if _, err := svc.ListDataFolders(context.Background(), ""); !errors.Is(err, ErrValidation) {
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
	out, asset, err := svc.DownloadDataAsset(context.Background(), "asset-1")
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
	if got, err := svc.ListHandoverDataItems(context.Background(), "h-1"); err != nil || len(got) != 1 {
		t.Fatalf("ListHandoverDataItems = %#v err=%v", got, err)
	}
	if _, err := svc.ListHandoverDataItems(context.Background(), ""); !errors.Is(err, ErrValidation) {
		t.Fatalf("empty handover err = %v, want ErrValidation", err)
	}
	if got, err := svc.UpdateHandoverDataItems(context.Background(), command.HandoverDataItemUpdateInput{HandoverID: "h-1"}); err != nil || got["updated"] != true {
		t.Fatalf("UpdateHandoverDataItems = %#v err=%v", got, err)
	}
	if _, err := svc.UpdateHandoverDataItems(context.Background(), command.HandoverDataItemUpdateInput{}); !errors.Is(err, ErrValidation) {
		t.Fatalf("empty update handover err = %v, want ErrValidation", err)
	}
}
