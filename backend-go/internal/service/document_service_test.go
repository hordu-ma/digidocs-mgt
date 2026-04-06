package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/storage"
)

// --- mocks ---

type mockDocumentReader struct {
	items []query.DocumentListItem
	total int
	item  *query.DocumentDetail
	err   error
}

func (m *mockDocumentReader) ListDocuments(_ context.Context, _ query.DocumentListFilter) ([]query.DocumentListItem, int, error) {
	return m.items, m.total, m.err
}

func (m *mockDocumentReader) GetDocument(_ context.Context, _ string) (*query.DocumentDetail, error) {
	return m.item, m.err
}

type mockDocumentWriter struct {
	result map[string]any
	err    error
}

func (m *mockDocumentWriter) CreateDocument(_ context.Context, _ command.DocumentCreateInput) (map[string]any, error) {
	return m.result, m.err
}

func (m *mockDocumentWriter) UpdateDocument(_ context.Context, _ command.DocumentUpdateInput) (map[string]any, error) {
	return m.result, m.err
}

func (m *mockDocumentWriter) DeleteDocument(_ context.Context, _ command.DocumentDeleteInput) error {
	return m.err
}

func (m *mockDocumentWriter) RestoreDocument(_ context.Context, _ string, _ string) error {
	return m.err
}

type mockDocStorage struct {
	result storage.PutObjectResult
	err    error
}

func (m *mockDocStorage) PutObject(_ context.Context, _ storage.PutObjectInput) (storage.PutObjectResult, error) {
	return m.result, m.err
}

func (m *mockDocStorage) GetObject(_ context.Context, _ string) (*storage.GetObjectOutput, error) {
	return nil, errors.New("not implemented in mock")
}

type mockDocVersionWorkflow struct {
	result map[string]any
	err    error
}

func (m *mockDocVersionWorkflow) CreateUploadedVersion(_ context.Context, _ command.VersionCreateInput) (map[string]any, error) {
	return m.result, m.err
}

// --- tests ---

func validCreateInput() command.DocumentCreateInput {
	return command.DocumentCreateInput{
		TeamSpaceID:    "ts-1",
		ProjectID:      "p-1",
		Title:          "Test Doc",
		CurrentOwnerID: "u-1",
		ActorID:        "u-1",
	}
}

func TestDocumentService_CreateWithFirstVersion_OK(t *testing.T) {
	svc := NewDocumentService(
		&mockDocumentReader{},
		&mockDocumentWriter{result: map[string]any{"id": "doc-1", "title": "Test Doc", "current_status": "draft"}},
		&mockDocStorage{result: storage.PutObjectResult{ObjectKey: "documents/doc-1/test.pdf", Provider: "memory"}},
		&mockDocVersionWorkflow{result: map[string]any{"id": "v-1", "version_no": 1}},
	)
	data, err := svc.CreateWithFirstVersion(context.Background(), validCreateInput(), "test.pdf", 1024, "initial", strings.NewReader("content"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data["id"] != "doc-1" {
		t.Errorf("id = %v, want doc-1", data["id"])
	}
	cv, ok := data["current_version"].(map[string]any)
	if !ok || cv["id"] != "v-1" {
		t.Errorf("current_version = %v, want {id:v-1}", data["current_version"])
	}
}

func TestDocumentService_CreateWithFirstVersion_MissingTeamSpaceID(t *testing.T) {
	svc := NewDocumentService(&mockDocumentReader{}, &mockDocumentWriter{}, &mockDocStorage{}, &mockDocVersionWorkflow{})
	input := validCreateInput()
	input.TeamSpaceID = ""
	_, err := svc.CreateWithFirstVersion(context.Background(), input, "test.pdf", 1024, "", strings.NewReader(""))
	if !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}

func TestDocumentService_CreateWithFirstVersion_MissingProjectID(t *testing.T) {
	svc := NewDocumentService(&mockDocumentReader{}, &mockDocumentWriter{}, &mockDocStorage{}, &mockDocVersionWorkflow{})
	input := validCreateInput()
	input.ProjectID = ""
	_, err := svc.CreateWithFirstVersion(context.Background(), input, "test.pdf", 1024, "", strings.NewReader(""))
	if !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}

func TestDocumentService_CreateWithFirstVersion_MissingTitle(t *testing.T) {
	svc := NewDocumentService(&mockDocumentReader{}, &mockDocumentWriter{}, &mockDocStorage{}, &mockDocVersionWorkflow{})
	input := validCreateInput()
	input.Title = ""
	_, err := svc.CreateWithFirstVersion(context.Background(), input, "test.pdf", 1024, "", strings.NewReader(""))
	if !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}

func TestDocumentService_CreateWithFirstVersion_MissingFile(t *testing.T) {
	svc := NewDocumentService(&mockDocumentReader{}, &mockDocumentWriter{}, &mockDocStorage{}, &mockDocVersionWorkflow{})
	_, err := svc.CreateWithFirstVersion(context.Background(), validCreateInput(), "", 0, "", strings.NewReader(""))
	if !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}

func TestDocumentService_CreateWithFirstVersion_WriterError(t *testing.T) {
	svc := NewDocumentService(
		&mockDocumentReader{},
		&mockDocumentWriter{err: errors.New("db error")},
		&mockDocStorage{},
		&mockDocVersionWorkflow{},
	)
	_, err := svc.CreateWithFirstVersion(context.Background(), validCreateInput(), "test.pdf", 1024, "", strings.NewReader(""))
	if err == nil || !strings.Contains(err.Error(), "document creation failed") {
		t.Errorf("err = %v, want document creation failed", err)
	}
}

func TestDocumentService_CreateWithFirstVersion_StorageError(t *testing.T) {
	svc := NewDocumentService(
		&mockDocumentReader{},
		&mockDocumentWriter{result: map[string]any{"id": "doc-1"}},
		&mockDocStorage{err: errors.New("s3 down")},
		&mockDocVersionWorkflow{},
	)
	_, err := svc.CreateWithFirstVersion(context.Background(), validCreateInput(), "test.pdf", 1024, "", strings.NewReader(""))
	if err == nil || !strings.Contains(err.Error(), "file upload failed") {
		t.Errorf("err = %v, want file upload failed", err)
	}
}

func TestDocumentService_CreateWithFirstVersion_VersionError(t *testing.T) {
	svc := NewDocumentService(
		&mockDocumentReader{},
		&mockDocumentWriter{result: map[string]any{"id": "doc-1"}},
		&mockDocStorage{result: storage.PutObjectResult{ObjectKey: "k", Provider: "memory"}},
		&mockDocVersionWorkflow{err: errors.New("tx error")},
	)
	_, err := svc.CreateWithFirstVersion(context.Background(), validCreateInput(), "test.pdf", 1024, "", strings.NewReader(""))
	if err == nil || !strings.Contains(err.Error(), "first version creation failed") {
		t.Errorf("err = %v, want first version creation failed", err)
	}
}

// --- UpdateDocument tests ---

func TestDocumentService_UpdateDocument_OK(t *testing.T) {
	svc := NewDocumentService(
		&mockDocumentReader{},
		&mockDocumentWriter{result: map[string]any{"id": "doc-1", "title": "New Title"}},
		&mockDocStorage{},
		&mockDocVersionWorkflow{},
	)
	data, err := svc.UpdateDocument(context.Background(), command.DocumentUpdateInput{
		DocumentID: "doc-1", Title: "New Title", ActorID: "u-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data["title"] != "New Title" {
		t.Errorf("title = %v, want New Title", data["title"])
	}
}

func TestDocumentService_UpdateDocument_MissingDocumentID(t *testing.T) {
	svc := NewDocumentService(&mockDocumentReader{}, &mockDocumentWriter{}, &mockDocStorage{}, &mockDocVersionWorkflow{})
	_, err := svc.UpdateDocument(context.Background(), command.DocumentUpdateInput{})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}

func TestDocumentService_UpdateDocument_NoFieldsProvided(t *testing.T) {
	svc := NewDocumentService(&mockDocumentReader{}, &mockDocumentWriter{}, &mockDocStorage{}, &mockDocVersionWorkflow{})
	_, err := svc.UpdateDocument(context.Background(), command.DocumentUpdateInput{DocumentID: "doc-1"})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}

func TestDocumentService_UpdateDocument_WriterError(t *testing.T) {
	svc := NewDocumentService(
		&mockDocumentReader{},
		&mockDocumentWriter{err: errors.New("db error")},
		&mockDocStorage{},
		&mockDocVersionWorkflow{},
	)
	_, err := svc.UpdateDocument(context.Background(), command.DocumentUpdateInput{
		DocumentID: "doc-1", Title: "T", ActorID: "u-1",
	})
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// --- DeleteDocument tests ---

func TestDocumentService_DeleteDocument_OK(t *testing.T) {
	svc := NewDocumentService(&mockDocumentReader{}, &mockDocumentWriter{}, &mockDocStorage{}, &mockDocVersionWorkflow{})
	err := svc.DeleteDocument(context.Background(), command.DocumentDeleteInput{
		DocumentID: "doc-1", Reason: "test", ActorID: "u-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDocumentService_DeleteDocument_MissingDocumentID(t *testing.T) {
	svc := NewDocumentService(&mockDocumentReader{}, &mockDocumentWriter{}, &mockDocStorage{}, &mockDocVersionWorkflow{})
	err := svc.DeleteDocument(context.Background(), command.DocumentDeleteInput{})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}

// --- RestoreDocument tests ---

func TestDocumentService_RestoreDocument_OK(t *testing.T) {
	svc := NewDocumentService(&mockDocumentReader{}, &mockDocumentWriter{}, &mockDocStorage{}, &mockDocVersionWorkflow{})
	err := svc.RestoreDocument(context.Background(), "doc-1", "u-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDocumentService_RestoreDocument_MissingDocumentID(t *testing.T) {
	svc := NewDocumentService(&mockDocumentReader{}, &mockDocumentWriter{}, &mockDocStorage{}, &mockDocVersionWorkflow{})
	err := svc.RestoreDocument(context.Background(), "", "u-1")
	if !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}
