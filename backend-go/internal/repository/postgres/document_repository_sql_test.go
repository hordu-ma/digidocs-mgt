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

func TestDocumentRepositoryListDocumentsDefaultsAndScans(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewDocumentRepository(db)

	mock.ExpectQuery("FROM documents d\\s+LEFT JOIN projects").
		WithArgs("", "", "", "", "", "", false, 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"title",
			"project_name",
			"folder_path",
			"current_status",
			"owner_id",
			"owner_display_name",
			"version_no",
			"updated_at",
		}).AddRow(
			"doc-1",
			"开题报告",
			"课题 A",
			"/申报材料",
			"in_progress",
			"u-1",
			"张三",
			3,
			"2026-05-01T00:00:00Z",
		))
	mock.ExpectQuery("SELECT COUNT\\(1\\)").
		WithArgs("", "", "", "", "", "", false).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	items, total, err := repo.ListDocuments(ctx, query.DocumentListFilter{})
	if err != nil {
		t.Fatalf("ListDocuments unexpected error: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("ListDocuments got total=%d items=%#v", total, items)
	}
	if items[0].CurrentOwner == nil || items[0].CurrentOwner.DisplayName != "张三" {
		t.Fatalf("CurrentOwner = %#v, want 张三", items[0].CurrentOwner)
	}
	if items[0].CurrentVersionNo == nil || *items[0].CurrentVersionNo != 3 {
		t.Fatalf("CurrentVersionNo = %#v, want 3", items[0].CurrentVersionNo)
	}
	assertExpectations(t, mock)
}

func TestDocumentRepositoryCreateAndGetDocumentNotFound(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewDocumentRepository(db)

	mock.ExpectExec("INSERT INTO documents").
		WithArgs(
			sqlmock.AnyArg(),
			"ts-1",
			"p-1",
			"",
			"开题报告",
			"",
			"u-1",
			"actor-1",
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	created, err := repo.CreateDocument(ctx, command.DocumentCreateInput{
		TeamSpaceID:    "ts-1",
		ProjectID:      "p-1",
		Title:          "开题报告",
		CurrentOwnerID: "u-1",
		ActorID:        "actor-1",
	})
	if err != nil {
		t.Fatalf("CreateDocument unexpected error: %v", err)
	}
	if created["title"] != "开题报告" || created["current_status"] != "draft" {
		t.Fatalf("CreateDocument got %#v", created)
	}

	mock.ExpectQuery("FROM documents d\\s+LEFT JOIN users").
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)
	if _, err := repo.GetDocument(ctx, "missing"); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("GetDocument err=%v, want ErrNotFound", err)
	}
	assertExpectations(t, mock)
}

func TestDocumentRepositoryUpdateDeleteRestoreNotFound(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewDocumentRepository(db)

	mock.ExpectQuery("UPDATE documents").
		WithArgs("missing", "标题", "", "").
		WillReturnError(sql.ErrNoRows)
	if _, err := repo.UpdateDocument(ctx, command.DocumentUpdateInput{
		DocumentID: "missing",
		Title:      "标题",
	}); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("UpdateDocument err=%v, want ErrNotFound", err)
	}

	mock.ExpectExec("UPDATE documents SET is_deleted = true").
		WithArgs("missing").
		WillReturnResult(sqlmock.NewResult(0, 0))
	if err := repo.DeleteDocument(ctx, command.DocumentDeleteInput{DocumentID: "missing"}); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("DeleteDocument err=%v, want ErrNotFound", err)
	}

	mock.ExpectExec("UPDATE documents SET is_deleted = false").
		WithArgs("missing").
		WillReturnResult(sqlmock.NewResult(0, 0))
	if err := repo.RestoreDocument(ctx, "missing", "actor-1"); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("RestoreDocument err=%v, want ErrNotFound", err)
	}
	assertExpectations(t, mock)
}
