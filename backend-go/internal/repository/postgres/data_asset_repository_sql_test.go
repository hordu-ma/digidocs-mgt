package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"digidocs-mgt/backend-go/internal/domain/command"
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
