package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"digidocs-mgt/backend-go/internal/service"
)

func TestProjectRepositoryListAndFolderTree(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewProjectRepository(db)

	mock.ExpectQuery("FROM projects p\\s+JOIN users u").
		WithArgs("ts-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "team_space_id", "name", "code", "owner_id", "owner_name"}).
			AddRow("p-1", "ts-1", "Project", "proj", "u-1", "Owner"))
	projects, err := repo.ListProjects(ctx, "ts-1", "admin", "admin")
	if err != nil || len(projects) != 1 || projects[0].Owner.DisplayName != "Owner" {
		t.Fatalf("admin ListProjects got %#v err=%v", projects, err)
	}

	mock.ExpectQuery("FROM projects p\\s+JOIN users u.*JOIN project_members").
		WithArgs("u-1", "ts-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "team_space_id", "name", "code", "owner_id", "owner_name"}).
			AddRow("p-2", "ts-1", "Member Project", "member", "u-2", "Lead"))
	projects, err = repo.ListProjects(ctx, "ts-1", "u-1", "member")
	if err != nil || len(projects) != 1 || projects[0].ID != "p-2" {
		t.Fatalf("member ListProjects got %#v err=%v", projects, err)
	}

	mock.ExpectQuery("FROM folders\\s+WHERE project_id::text").
		WithArgs("p-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "parent_id", "name", "path"}).
			AddRow("f-root", "", "Root", "/root").
			AddRow("f-child", "f-root", "Child", "/root/child").
			AddRow("f-orphan", "missing", "Orphan", "/orphan"))
	tree, err := repo.GetFolderTree(ctx, "p-1")
	if err != nil || len(tree) != 2 || len(tree[0].Children) != 1 {
		t.Fatalf("GetFolderTree got %#v err=%v", tree, err)
	}
	assertExpectations(t, mock)
}

func TestFlowAndHandoverRepositories(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	flows := NewFlowRepository(db)
	handovers := NewHandoverRepository(db)

	mock.ExpectQuery("FROM flow_records").WithArgs("doc-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "action", "from_status", "to_status", "created_at"}).
			AddRow("flow-1", "transfer", "in_progress", "pending_handover", "2026-05-01T00:00:00Z"))
	flowItems, err := flows.ListFlows(ctx, "doc-1")
	if err != nil || len(flowItems) != 1 || flowItems[0].Action != "transfer" {
		t.Fatalf("ListFlows got %#v err=%v", flowItems, err)
	}

	mock.ExpectQuery("FROM graduation_handovers\\s+ORDER BY").
		WillReturnRows(sqlmock.NewRows([]string{"id", "target_user_id", "receiver_user_id", "project_id", "status", "remark"}).
			AddRow("h-1", "u-1", "u-2", "p-1", "generated", "remark"))
	handoverItems, err := handovers.ListHandovers(ctx)
	if err != nil || len(handoverItems) != 1 || handoverItems[0].Status != "generated" {
		t.Fatalf("ListHandovers got %#v err=%v", handoverItems, err)
	}

	mock.ExpectQuery("FROM graduation_handovers\\s+WHERE id::text").
		WithArgs("h-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "target_user_id", "receiver_user_id", "project_id", "status", "remark"}).
			AddRow("h-1", "u-1", "u-2", "p-1", "generated", "remark"))
	mock.ExpectQuery("FROM graduation_handover_items").
		WithArgs("h-1").
		WillReturnRows(sqlmock.NewRows([]string{"document_id", "selected", "note"}).AddRow("doc-1", true, "keep"))
	handover, err := handovers.GetHandover(ctx, "h-1")
	if err != nil || handover.ID != "h-1" || len(handover.Items) != 1 {
		t.Fatalf("GetHandover got %#v err=%v", handover, err)
	}

	mock.ExpectQuery("FROM graduation_handovers\\s+WHERE id::text").WillReturnError(sql.ErrNoRows)
	if _, err := handovers.GetHandover(ctx, "missing"); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("GetHandover err=%v, want ErrNotFound", err)
	}
	assertExpectations(t, mock)
}
