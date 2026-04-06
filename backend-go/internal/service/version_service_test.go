package service

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/storage"
)

// --- mocks ---

type mockStorageProvider struct {
	result    storage.PutObjectResult
	err       error
	getObject *storage.GetObjectOutput
	getErr    error
}

func (m *mockStorageProvider) PutObject(_ context.Context, _ storage.PutObjectInput) (storage.PutObjectResult, error) {
	return m.result, m.err
}

func (m *mockStorageProvider) GetObject(_ context.Context, _ string) (*storage.GetObjectOutput, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.getObject != nil {
		return m.getObject, nil
	}
	return nil, errors.New("not implemented in mock")
}

func (m *mockStorageProvider) DeleteObject(_ context.Context, _ string) error {
	return nil
}

func (m *mockStorageProvider) Stat(_ context.Context, _ string) (*storage.FileInfo, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockStorageProvider) ListDir(_ context.Context, _ string) ([]storage.FileInfo, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockStorageProvider) CreateFolder(_ context.Context, _ string) error {
	return nil
}

func (m *mockStorageProvider) CreateShareLink(_ context.Context, _ string, _ int) (*storage.ShareLinkResult, error) {
	return nil, errors.New("not implemented in mock")
}

type mockVersionWorkflow struct {
	result map[string]any
	err    error
}

func (m *mockVersionWorkflow) CreateUploadedVersion(_ context.Context, _ command.VersionCreateInput) (map[string]any, error) {
	return m.result, m.err
}

type mockVersionReader struct {
	items []query.VersionItem
	item  *query.VersionDetail
	err   error
}

func (m *mockVersionReader) ListVersions(_ context.Context, _ string) ([]query.VersionItem, error) {
	return m.items, m.err
}

func (m *mockVersionReader) GetVersion(_ context.Context, _ string) (*query.VersionDetail, error) {
	return m.item, m.err
}

// --- tests ---

func TestVersionService_UploadAndCreateVersion_OK(t *testing.T) {
	sp := &mockStorageProvider{result: storage.PutObjectResult{ObjectKey: "documents/doc-1/test.pdf", Provider: "memory"}}
	wf := &mockVersionWorkflow{result: map[string]any{"id": "v-1", "version_no": 1}}
	vr := &mockVersionReader{}
	svc := NewVersionService(sp, wf, vr)

	data, err := svc.UploadAndCreateVersion(
		context.Background(), "doc-1", "test.pdf", 1024, "initial", strings.NewReader("content"), "u-1",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data["id"] != "v-1" {
		t.Errorf("id = %v, want v-1", data["id"])
	}
	if data["status"] != "uploaded" {
		t.Errorf("status = %v, want uploaded", data["status"])
	}
}

func TestVersionService_UploadAndCreateVersion_MissingDocumentID(t *testing.T) {
	svc := NewVersionService(&mockStorageProvider{}, &mockVersionWorkflow{}, &mockVersionReader{})
	_, err := svc.UploadAndCreateVersion(context.Background(), "", "test.pdf", 1024, "", strings.NewReader(""), "u-1")
	if !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}

func TestVersionService_UploadAndCreateVersion_MissingFileName(t *testing.T) {
	svc := NewVersionService(&mockStorageProvider{}, &mockVersionWorkflow{}, &mockVersionReader{})
	_, err := svc.UploadAndCreateVersion(context.Background(), "doc-1", "", 1024, "", strings.NewReader(""), "u-1")
	if !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}

func TestVersionService_UploadAndCreateVersion_MissingActorID(t *testing.T) {
	svc := NewVersionService(&mockStorageProvider{}, &mockVersionWorkflow{}, &mockVersionReader{})
	_, err := svc.UploadAndCreateVersion(context.Background(), "doc-1", "test.pdf", 1024, "", strings.NewReader(""), "")
	if !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}

func TestVersionService_UploadAndCreateVersion_StorageError(t *testing.T) {
	sp := &mockStorageProvider{err: errors.New("storage down")}
	svc := NewVersionService(sp, &mockVersionWorkflow{}, &mockVersionReader{})
	_, err := svc.UploadAndCreateVersion(context.Background(), "doc-1", "test.pdf", 1024, "", strings.NewReader(""), "u-1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "upload failed") {
		t.Errorf("err = %v, want upload failed", err)
	}
}

func TestVersionService_UploadAndCreateVersion_WorkflowError(t *testing.T) {
	sp := &mockStorageProvider{result: storage.PutObjectResult{ObjectKey: "k", Provider: "memory"}}
	wf := &mockVersionWorkflow{err: errors.New("db error")}
	svc := NewVersionService(sp, wf, &mockVersionReader{})
	_, err := svc.UploadAndCreateVersion(context.Background(), "doc-1", "test.pdf", 1024, "", strings.NewReader(""), "u-1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "version creation failed") {
		t.Errorf("err = %v, want version creation failed", err)
	}
}

func TestVersionService_List_Delegates(t *testing.T) {
	items := []query.VersionItem{{ID: "v-1", VersionNo: 1}}
	svc := NewVersionService(&mockStorageProvider{}, &mockVersionWorkflow{}, &mockVersionReader{items: items})
	got, err := svc.List(context.Background(), "doc-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "v-1" {
		t.Errorf("got = %v, want [{ID:v-1}]", got)
	}
}

func TestVersionService_GetFile_OK(t *testing.T) {
	output := &storage.GetObjectOutput{
		Reader:      io.NopCloser(strings.NewReader("hello version")),
		ContentType: "text/plain",
		Size:        13,
	}
	reader := &mockVersionReader{item: &query.VersionDetail{
		ID:               "v-1",
		FileName:         "report.txt",
		StorageObjectKey: "documents/doc-1/report.txt",
	}}
	svc := NewVersionService(&mockStorageProvider{getObject: output}, &mockVersionWorkflow{}, reader)

	ver, obj, err := svc.GetFile(context.Background(), "v-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver.ID != "v-1" {
		t.Fatalf("version id = %s, want v-1", ver.ID)
	}
	if obj.ContentType != "text/plain" {
		t.Fatalf("content type = %s, want text/plain", obj.ContentType)
	}
}

func TestVersionService_GetFile_StorageError(t *testing.T) {
	reader := &mockVersionReader{item: &query.VersionDetail{
		ID:               "v-1",
		FileName:         "report.txt",
		StorageObjectKey: "documents/doc-1/report.txt",
	}}
	svc := NewVersionService(&mockStorageProvider{getErr: errors.New("storage down")}, &mockVersionWorkflow{}, reader)

	ver, obj, err := svc.GetFile(context.Background(), "v-1")
	if err == nil {
		t.Fatal("expected error")
	}
	if ver == nil || ver.ID != "v-1" {
		t.Fatalf("version = %#v, want v-1", ver)
	}
	if obj != nil {
		t.Fatal("expected nil object on storage error")
	}
	if !strings.Contains(err.Error(), "storage get failed") {
		t.Fatalf("err = %v, want storage get failed", err)
	}
}

func newReader(s string) io.Reader { return strings.NewReader(s) }
