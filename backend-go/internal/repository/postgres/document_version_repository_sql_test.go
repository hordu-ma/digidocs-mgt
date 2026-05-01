package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/service"
)

func TestDocumentRepository_ListAndGet(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewDocumentRepository(db)
	ver := 3
	updatedAt := "2026-05-01T00:00:00Z"
	mock.ExpectQuery("FROM documents d\\s+LEFT JOIN projects").
		WithArgs("ts-1", "p-1", "", "", "", "report", false, 5, 5).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "project_name", "folder_path", "status", "owner_id", "owner_name", "version_no", "updated_at"}).
			AddRow("doc-1", "Report", "Project", "/docs", "in_progress", "u-1", "张三", ver, updatedAt))
	mock.ExpectQuery("SELECT COUNT\\(1\\)\\s+FROM documents d").
		WithArgs("ts-1", "p-1", "", "", "", "report", false).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(8))
	items, total, err := repo.ListDocuments(ctx, query.DocumentListFilter{
		TeamSpaceID: "ts-1", ProjectID: "p-1", Keyword: "report", Page: 2, PageSize: 5,
	})
	if err != nil || total != 8 || len(items) != 1 || *items[0].CurrentVersionNo != 3 {
		t.Fatalf("ListDocuments items=%#v total=%d err=%v", items, total, err)
	}

	mock.ExpectQuery("FROM documents d\\s+LEFT JOIN users").
		WithArgs("doc-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "status", "owner_id", "owner_name", "version_id", "archived"}).
			AddRow("doc-1", "Report", "Desc", "in_progress", "u-1", "张三", "v-1", false))
	doc, err := repo.GetDocument(ctx, "doc-1")
	if err != nil || doc.ID != "doc-1" || doc.CurrentOwner.DisplayName != "张三" {
		t.Fatalf("GetDocument doc=%#v err=%v", doc, err)
	}
	assertExpectations(t, mock)
}

func TestDocumentRepository_WriteMethods(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewDocumentRepository(db)
	mock.ExpectExec("INSERT INTO documents").
		WithArgs(sqlmock.AnyArg(), "ts-1", "p-1", "", "Report", "Desc", "u-1", "actor-1", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	created, err := repo.CreateDocument(ctx, command.DocumentCreateInput{
		TeamSpaceID: "ts-1", ProjectID: "p-1", Title: "Report", Description: "Desc", CurrentOwnerID: "u-1", ActorID: "actor-1",
	})
	if err != nil || created["title"] != "Report" || created["current_status"] != "draft" {
		t.Fatalf("CreateDocument result=%#v err=%v", created, err)
	}

	versionID := "v-1"
	mock.ExpectQuery("UPDATE documents\\s+SET title").
		WithArgs("doc-1", "New", "Desc", "").
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "status", "owner_id", "owner_name", "version_id", "archived"}).
			AddRow("doc-1", "New", "Desc", "in_progress", "u-1", "张三", &versionID, false))
	updated, err := repo.UpdateDocument(ctx, command.DocumentUpdateInput{DocumentID: "doc-1", Title: "New", Description: "Desc"})
	if err != nil || updated["title"] != "New" {
		t.Fatalf("UpdateDocument result=%#v err=%v", updated, err)
	}

	mock.ExpectExec("UPDATE documents SET is_deleted = true").WithArgs("doc-1").WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.DeleteDocument(ctx, command.DocumentDeleteInput{DocumentID: "doc-1"}); err != nil {
		t.Fatalf("DeleteDocument unexpected err=%v", err)
	}
	mock.ExpectExec("UPDATE documents SET is_deleted = false").WithArgs("doc-1").WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.RestoreDocument(ctx, "doc-1", "actor-1"); err != nil {
		t.Fatalf("RestoreDocument unexpected err=%v", err)
	}
	assertExpectations(t, mock)
}

func TestDocumentRepository_NotFoundBranches(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewDocumentRepository(db)
	mock.ExpectQuery("FROM documents d\\s+LEFT JOIN users").WillReturnError(sql.ErrNoRows)
	if _, err := repo.GetDocument(ctx, "missing"); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("GetDocument err=%v, want ErrNotFound", err)
	}
	mock.ExpectQuery("UPDATE documents\\s+SET title").WillReturnError(sql.ErrNoRows)
	if _, err := repo.UpdateDocument(ctx, command.DocumentUpdateInput{DocumentID: "missing"}); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("UpdateDocument err=%v, want ErrNotFound", err)
	}
	mock.ExpectExec("UPDATE documents SET is_deleted = true").WillReturnResult(sqlmock.NewResult(0, 0))
	if err := repo.DeleteDocument(ctx, command.DocumentDeleteInput{DocumentID: "missing"}); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("DeleteDocument err=%v, want ErrNotFound", err)
	}
	mock.ExpectExec("UPDATE documents SET is_deleted = false").WillReturnResult(sqlmock.NewResult(0, 0))
	if err := repo.RestoreDocument(ctx, "missing", "actor"); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("RestoreDocument err=%v, want ErrNotFound", err)
	}
	assertExpectations(t, mock)
}

func TestVersionRepositoryMethods(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewVersionRepository(db)
	mock.ExpectQuery("FROM document_versions\\s+WHERE document_id").
		WithArgs("doc-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "version_no", "file_name", "summary_status", "created_at"}).
			AddRow("v-2", 2, "report.pdf", "pending", "2026-05-01T00:00:00Z"))
	versions, err := repo.ListVersions(ctx, "doc-1")
	if err != nil || len(versions) != 1 || versions[0].VersionNo != 2 {
		t.Fatalf("ListVersions got %#v err=%v", versions, err)
	}

	mock.ExpectQuery("FROM document_versions\\s+WHERE id::text").
		WithArgs("v-2").
		WillReturnRows(sqlmock.NewRows([]string{"id", "document_id", "version_no", "commit_message", "file_name", "file_size", "provider", "object_key", "mime"}).
			AddRow("v-2", "doc-1", 2, "msg", "report.pdf", int64(128), "memory", "documents/doc-1/report.pdf", "application/pdf"))
	version, err := repo.GetVersion(ctx, "v-2")
	if err != nil || version.ID != "v-2" || version.StorageObjectKey == "" {
		t.Fatalf("GetVersion got %#v err=%v", version, err)
	}

	mock.ExpectExec("SELECT id\\s+FROM documents").WithArgs("doc-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT COALESCE\\(MAX\\(version_no\\), 0\\) \\+ 1").
		WithArgs("doc-1").
		WillReturnRows(sqlmock.NewRows([]string{"version_no"}).AddRow(3))
	mock.ExpectExec("INSERT INTO document_versions").
		WithArgs(sqlmock.AnyArg(), "doc-1", 3, "new.pdf", int64(256), "memory", "k", "msg", "actor-1", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	created, err := repo.CreateVersion(ctx, command.VersionCreateInput{
		DocumentID: "doc-1", FileName: "new.pdf", FileSize: 256, StorageProvider: "memory", StorageObjectKey: "k", CommitMessage: "msg", ActorID: "actor-1",
	})
	if err != nil || created["version_no"] != 3 {
		t.Fatalf("CreateVersion got %#v err=%v", created, err)
	}
	assertExpectations(t, mock)
}

func TestVersionRepositoryGetNotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewVersionRepository(db)
	mock.ExpectQuery("FROM document_versions\\s+WHERE id::text").WillReturnError(sql.ErrNoRows)
	if _, err := repo.GetVersion(context.Background(), "missing"); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("GetVersion err=%v, want ErrNotFound", err)
	}
	assertExpectations(t, mock)
}
