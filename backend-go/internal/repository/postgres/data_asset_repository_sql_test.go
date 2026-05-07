package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/service"
)

func TestDataAssetRepositoryFolderMethods(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewDataAssetRepository(db)

	mock.ExpectQuery("FROM data_folders").
		WithArgs("p-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "project_id", "parent_id", "depth", "name", "created_at"}).
			AddRow("folder-1", "p-1", "", 0, "Raw", "2026-05-01T00:00:00Z"))
	folders, err := repo.ListDataFolders(ctx, "p-1")
	if err != nil || len(folders) != 1 || folders[0].Name != "Raw" {
		t.Fatalf("ListDataFolders got %#v err=%v", folders, err)
	}

	mock.ExpectExec("INSERT INTO data_folders").
		WithArgs(sqlmock.AnyArg(), "p-1", nil, 0, "Raw", "u-1", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	created, err := repo.CreateDataFolder(ctx, command.DataFolderCreateInput{ProjectID: "p-1", Name: "Raw", ActorID: "u-1"})
	if err != nil || created.ProjectID != "p-1" || created.Depth != 0 {
		t.Fatalf("CreateDataFolder root got %#v err=%v", created, err)
	}

	mock.ExpectQuery("SELECT depth FROM data_folders").
		WithArgs("parent-1", "p-1").
		WillReturnRows(sqlmock.NewRows([]string{"depth"}).AddRow(1))
	mock.ExpectExec("INSERT INTO data_folders").
		WithArgs(sqlmock.AnyArg(), "p-1", "parent-1", 2, "Leaf", "u-1", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	created, err = repo.CreateDataFolder(ctx, command.DataFolderCreateInput{ProjectID: "p-1", ParentID: "parent-1", Name: "Leaf", ActorID: "u-1"})
	if err != nil || created.Depth != 2 || created.ParentID != "parent-1" {
		t.Fatalf("CreateDataFolder child got %#v err=%v", created, err)
	}

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM data_assets").
		WithArgs("folder-1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM data_folders").
		WithArgs("folder-1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectExec("DELETE FROM data_folders").WithArgs("folder-1").WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.DeleteDataFolder(ctx, "folder-1"); err != nil {
		t.Fatalf("DeleteDataFolder unexpected err=%v", err)
	}
	assertExpectations(t, mock)
}

func TestDataAssetRepositoryFolderValidationBranches(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewDataAssetRepository(db)
	mock.ExpectQuery("SELECT depth FROM data_folders").
		WithArgs("parent-1", "p-1").
		WillReturnRows(sqlmock.NewRows([]string{"depth"}).AddRow(2))
	if _, err := repo.CreateDataFolder(ctx, command.DataFolderCreateInput{ProjectID: "p-1", ParentID: "parent-1", Name: "TooDeep"}); !errors.Is(err, service.ErrValidation) {
		t.Fatalf("CreateDataFolder depth err=%v, want ErrValidation", err)
	}
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM data_assets").
		WithArgs("folder-1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	if err := repo.DeleteDataFolder(ctx, "folder-1"); !errors.Is(err, service.ErrValidation) {
		t.Fatalf("DeleteDataFolder non-empty err=%v, want ErrValidation", err)
	}
	assertExpectations(t, mock)
}

func TestDataAssetRepositoryAssetMethods(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewDataAssetRepository(db)

	mock.ExpectQuery("FROM data_assets da").
		WithArgs("p-1", "folder-1", "dataset", 2, 2).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "project_id", "project_name", "folder_id", "folder_name",
			"display_name", "file_name", "mime_type", "file_size", "created_by_name", "created_at",
		}).AddRow("asset-1", "p-1", "Project", "folder-1", "Raw", "Dataset", "dataset.csv", "text/csv", int64(12), "张三", "2026-05-01T00:00:00Z"))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\)").
		WithArgs("p-1", "folder-1", "dataset").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
	items, total, err := repo.ListDataAssets(ctx, commandToDataAssetFilter("p-1", "folder-1", "dataset", 2, 2))
	if err != nil || total != 5 || len(items) != 1 || items[0].DisplayName != "Dataset" {
		t.Fatalf("ListDataAssets got items=%#v total=%d err=%v", items, total, err)
	}

	mock.ExpectQuery("FROM data_assets da").
		WithArgs("asset-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "team_space_id", "project_id", "project_name", "folder_id", "folder_name",
			"display_name", "file_name", "description", "mime_type", "file_size",
			"storage_provider", "storage_object_key", "created_by_name", "created_at", "updated_at",
		}).AddRow("asset-1", "ts-1", "p-1", "Project", "folder-1", "Raw", "Dataset", "dataset.csv", "desc", "text/csv", int64(12), "memory", "obj", "张三", "2026-05-01T00:00:00Z", "2026-05-01T00:01:00Z"))
	detail, err := repo.GetDataAsset(ctx, "asset-1")
	if err != nil || detail.StorageObjectKey != "obj" || detail.Description != "desc" {
		t.Fatalf("GetDataAsset got %#v err=%v", detail, err)
	}

	mock.ExpectExec("INSERT INTO data_assets").
		WithArgs(sqlmock.AnyArg(), "ts-1", "p-1", nil, "Dataset", "dataset.csv", "", "text/csv; charset=utf-8", int64(12), "memory", nil, "obj", "actor-1", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	created, err := repo.CreateDataAsset(ctx, command.DataAssetCreateInput{
		TeamSpaceID:      "ts-1",
		ProjectID:        "p-1",
		DisplayName:      "Dataset",
		FileName:         "dataset.csv",
		FileSize:         12,
		StorageProvider:  "memory",
		StorageObjectKey: "obj",
		ActorID:          "actor-1",
	})
	if err != nil || created["mime_type"] != "text/csv; charset=utf-8" {
		t.Fatalf("CreateDataAsset got %#v err=%v", created, err)
	}

	mock.ExpectExec("UPDATE data_assets").
		WithArgs("asset-1", "Renamed", "desc", nil).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.UpdateDataAsset(ctx, command.DataAssetUpdateInput{DataAssetID: "asset-1", DisplayName: "Renamed", Description: "desc"}); err != nil {
		t.Fatalf("UpdateDataAsset unexpected err=%v", err)
	}

	mock.ExpectExec("UPDATE data_assets").
		WithArgs("asset-1", "actor-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.DeleteDataAsset(ctx, command.DataAssetDeleteInput{DataAssetID: "asset-1", ActorID: "actor-1"}); err != nil {
		t.Fatalf("DeleteDataAsset unexpected err=%v", err)
	}
	assertExpectations(t, mock)
}

func TestDataAssetRepositoryAssetNotFoundAndHandoverTransaction(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewDataAssetRepository(db)

	mock.ExpectQuery("FROM data_assets da").
		WithArgs("missing").
		WillReturnError(sqlmock.ErrCancelled)
	if _, err := repo.GetDataAsset(ctx, "missing"); !errors.Is(err, sqlmock.ErrCancelled) {
		t.Fatalf("GetDataAsset err=%v, want propagated sql error", err)
	}

	mock.ExpectExec("UPDATE data_assets").
		WithArgs("missing", "Missing", "", nil).
		WillReturnResult(sqlmock.NewResult(0, 0))
	if err := repo.UpdateDataAsset(ctx, command.DataAssetUpdateInput{DataAssetID: "missing", DisplayName: "Missing"}); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("UpdateDataAsset err=%v, want ErrNotFound", err)
	}

	mock.ExpectExec("UPDATE data_assets").
		WithArgs("missing", "actor-1").
		WillReturnResult(sqlmock.NewResult(0, 0))
	if err := repo.DeleteDataAsset(ctx, command.DataAssetDeleteInput{DataAssetID: "missing", ActorID: "actor-1"}); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("DeleteDataAsset err=%v, want ErrNotFound", err)
	}

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM graduation_handover_data_items").
		WithArgs("handover-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO graduation_handover_data_items").
		WithArgs(sqlmock.AnyArg(), "handover-1", "asset-1", true, "移交").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO graduation_handover_data_items").
		WithArgs(sqlmock.AnyArg(), "handover-1", "asset-2", false, nil).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	result, err := repo.UpdateHandoverDataItems(ctx, command.HandoverDataItemUpdateInput{
		HandoverID: "handover-1",
		Items: []command.HandoverDataItemInput{
			{DataAssetID: "asset-1", Selected: true, Note: "移交"},
			{DataAssetID: "asset-2"},
		},
	})
	if err != nil || result["count"] != 2 {
		t.Fatalf("UpdateHandoverDataItems got %#v err=%v", result, err)
	}
	assertExpectations(t, mock)
}

func commandToDataAssetFilter(projectID, folderID, keyword string, page, pageSize int) query.DataAssetListFilter {
	return query.DataAssetListFilter{
		ProjectID: projectID,
		FolderID:  folderID,
		Keyword:   keyword,
		Page:      page,
		PageSize:  pageSize,
	}
}
