package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPermissionRepositoryAdminShortCircuits(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewPermissionRepository(db)
	checks := []func() (bool, error){
		func() (bool, error) { return repo.CanCreateDocument(context.Background(), "u", "admin", "p") },
		func() (bool, error) { return repo.CanUpdateDocument(context.Background(), "u", "admin", "d") },
		func() (bool, error) { return repo.CanDeleteDocument(context.Background(), "u", "admin", "d") },
		func() (bool, error) { return repo.CanUploadVersion(context.Background(), "u", "admin", "d") },
		func() (bool, error) { return repo.CanFlowDocument(context.Background(), "u", "admin", "d", "archive") },
		func() (bool, error) { return repo.CanCreateHandover(context.Background(), "u", "admin", "p") },
		func() (bool, error) { return repo.CanUpdateHandoverItems(context.Background(), "u", "admin", "h") },
		func() (bool, error) { return repo.CanApplyHandover(context.Background(), "u", "admin", "h", "confirm") },
		func() (bool, error) { return repo.CanUploadDataAsset(context.Background(), "u", "admin", "p") },
		func() (bool, error) { return repo.CanManageDataAsset(context.Background(), "u", "admin", "a") },
		func() (bool, error) { return repo.CanCreateCodeRepository(context.Background(), "u", "admin", "p") },
		func() (bool, error) { return repo.CanManageCodeRepository(context.Background(), "u", "admin", "r") },
		func() (bool, error) { return repo.CanPushCodeRepository(context.Background(), "u", "admin", "r") },
	}
	for _, check := range checks {
		ok, err := check()
		if err != nil || !ok {
			t.Fatalf("admin check ok=%v err=%v, want true nil", ok, err)
		}
	}
	assertExpectations(t, mock)
}

func TestPermissionRepositoryDocumentAndProjectChecks(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewPermissionRepository(db)
	expectExists := func(ok bool) {
		mock.ExpectQuery("SELECT EXISTS").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(ok))
	}

	expectExists(true)
	if ok, err := repo.CanCreateDocument(ctx, "u-1", "member", "p-1"); err != nil || !ok {
		t.Fatalf("CanCreateDocument ok=%v err=%v", ok, err)
	}
	expectExists(true)
	if ok, err := repo.CanUpdateDocument(ctx, "u-1", "member", "d-1"); err != nil || !ok {
		t.Fatalf("CanUpdateDocument ok=%v err=%v", ok, err)
	}
	expectExists(false)
	if ok, err := repo.CanDeleteDocument(ctx, "u-1", "member", "d-1"); err != nil || ok {
		t.Fatalf("CanDeleteDocument ok=%v err=%v, want false nil", ok, err)
	}
	expectExists(true)
	if ok, err := repo.CanFlowDocument(ctx, "u-1", "member", "d-1", "archive"); err != nil || !ok {
		t.Fatalf("CanFlowDocument archive ok=%v err=%v", ok, err)
	}
	expectExists(true)
	if ok, err := repo.CanFlowDocument(ctx, "u-1", "member", "d-1", "transfer"); err != nil || !ok {
		t.Fatalf("CanFlowDocument transfer ok=%v err=%v", ok, err)
	}
	expectExists(true)
	if ok, err := repo.CanCreateCodeRepository(ctx, "u-1", "member", "p-1"); err != nil || !ok {
		t.Fatalf("CanCreateCodeRepository ok=%v err=%v", ok, err)
	}
	expectExists(true)
	if ok, err := repo.CanPushCodeRepository(ctx, "u-1", "member", "r-1"); err != nil || !ok {
		t.Fatalf("CanPushCodeRepository ok=%v err=%v", ok, err)
	}
	assertExpectations(t, mock)
}

func TestPermissionRepositoryHandoverAndAssetChecks(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewPermissionRepository(db)
	expectExists := func(ok bool) {
		mock.ExpectQuery("SELECT EXISTS").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(ok))
	}

	if ok, err := repo.CanCreateHandover(ctx, "u-1", "member", ""); err != nil || ok {
		t.Fatalf("empty project handover ok=%v err=%v, want false nil", ok, err)
	}
	expectExists(true)
	if ok, err := repo.CanCreateHandover(ctx, "u-1", "member", "p-1"); err != nil || !ok {
		t.Fatalf("CanCreateHandover ok=%v err=%v", ok, err)
	}
	mock.ExpectQuery("FROM graduation_handovers").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	if ok, err := repo.CanApplyHandover(ctx, "receiver", "member", "h-1", "confirm"); err != nil || !ok {
		t.Fatalf("receiver confirm ok=%v err=%v", ok, err)
	}
	mock.ExpectQuery("FROM graduation_handovers").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	expectExists(true)
	if ok, err := repo.CanApplyHandover(ctx, "manager", "member", "h-1", "confirm"); err != nil || !ok {
		t.Fatalf("manager confirm ok=%v err=%v", ok, err)
	}
	expectExists(true)
	if ok, err := repo.CanUpdateHandoverItems(ctx, "manager", "member", "h-1"); err != nil || !ok {
		t.Fatalf("CanUpdateHandoverItems ok=%v err=%v", ok, err)
	}
	expectExists(true)
	if ok, err := repo.CanUploadDataAsset(ctx, "u-1", "member", "p-1"); err != nil || !ok {
		t.Fatalf("CanUploadDataAsset ok=%v err=%v", ok, err)
	}
	expectExists(true)
	if ok, err := repo.CanManageDataAsset(ctx, "u-1", "member", "a-1"); err != nil || !ok {
		t.Fatalf("CanManageDataAsset ok=%v err=%v", ok, err)
	}
	expectExists(true)
	if ok, err := repo.CanManageCodeRepository(ctx, "u-1", "member", "r-1"); err != nil || !ok {
		t.Fatalf("CanManageCodeRepository ok=%v err=%v", ok, err)
	}
	assertExpectations(t, mock)
}

func TestPermissionRepositoryPropagatesDBError(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewPermissionRepository(db)
	dbErr := errors.New("db down")
	mock.ExpectQuery("SELECT EXISTS").WillReturnError(dbErr)
	if ok, err := repo.CanCreateDocument(context.Background(), "u-1", "member", "p-1"); ok || !errors.Is(err, dbErr) {
		t.Fatalf("ok=%v err=%v, want false db down", ok, err)
	}
	assertExpectations(t, mock)
}
