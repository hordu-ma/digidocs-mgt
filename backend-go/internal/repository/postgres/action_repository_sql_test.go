package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/shared"
)

func TestActionRepositoryCreateFlowRecordTransferCommits(t *testing.T) {
	ctx := shared.WithRequestID(context.Background(), "req-flow")
	db, mock := newMockDB(t)
	repo := NewActionRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT\\s+current_owner_id::text").
		WithArgs("doc-1").
		WillReturnRows(sqlmock.NewRows([]string{"current_owner_id", "current_status"}).AddRow("u-1", "in_progress"))
	mock.ExpectExec("INSERT INTO flow_records").
		WithArgs(
			sqlmock.AnyArg(),
			"doc-1",
			"u-1",
			"u-2",
			"in_progress",
			"pending_handover",
			"transfer",
			"请接手",
			"actor-1",
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE documents").
		WithArgs("doc-1", "u-2", "pending_handover", false, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	expectAuditInsert(mock, "doc-1", "", "actor-1", "transfer", "req-flow")
	mock.ExpectCommit()

	got, err := repo.CreateFlowRecord(ctx, command.FlowActionInput{
		DocumentID: "doc-1",
		Action:     "transfer",
		ToUserID:   "u-2",
		Note:       "请接手",
		ActorID:    "actor-1",
	})
	if err != nil {
		t.Fatalf("CreateFlowRecord unexpected error: %v", err)
	}
	if got["current_status"] != "pending_handover" || got["current_owner_id"] != "u-2" {
		t.Fatalf("CreateFlowRecord got %#v", got)
	}
	assertExpectations(t, mock)
}

func TestActionRepositoryCreateFlowRecordInvalidTransitionRollsBack(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewActionRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT\\s+current_owner_id::text").
		WithArgs("doc-1").
		WillReturnRows(sqlmock.NewRows([]string{"current_owner_id", "current_status"}).AddRow("u-1", "draft"))
	mock.ExpectRollback()

	_, err := repo.CreateFlowRecord(ctx, command.FlowActionInput{
		DocumentID: "doc-1",
		Action:     "transfer",
		ActorID:    "actor-1",
	})
	if !errors.Is(err, service.ErrInvalidTransition) {
		t.Fatalf("CreateFlowRecord err=%v, want ErrInvalidTransition", err)
	}
	assertExpectations(t, mock)
}

func TestActionRepositoryCreateHandoverCommitsAudit(t *testing.T) {
	ctx := shared.WithRequestID(context.Background(), "req-handover")
	db, mock := newMockDB(t)
	repo := NewActionRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO graduation_handovers").
		WithArgs(sqlmock.AnyArg(), "u-target", "u-receiver", "p-1", "毕业交接", "actor-1", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	expectAuditInsert(mock, "", "", "actor-1", "handover_generate", "req-handover")
	mock.ExpectCommit()

	got, err := repo.CreateHandover(ctx, command.HandoverCreateInput{
		TargetUserID:   "u-target",
		ReceiverUserID: "u-receiver",
		ProjectID:      "p-1",
		Remark:         "毕业交接",
		ActorID:        "actor-1",
	})
	if err != nil {
		t.Fatalf("CreateHandover unexpected error: %v", err)
	}
	if got["status"] != "generated" || got["target_user_id"] != "u-target" {
		t.Fatalf("CreateHandover got %#v", got)
	}
	assertExpectations(t, mock)
}

func TestActionRepositoryUpdateHandoverItemsSkipsEmptyDocumentID(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewActionRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM graduation_handover_items").
		WithArgs("handover-1").
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec("INSERT INTO graduation_handover_items").
		WithArgs(sqlmock.AnyArg(), "handover-1", "doc-1", true, "保留", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	expectAuditInsert(mock, "", "", "actor-1", "admin_update", "")
	mock.ExpectCommit()

	got, err := repo.UpdateHandoverItems(ctx, command.HandoverItemUpdateInput{
		HandoverID: "handover-1",
		ActorID:    "actor-1",
		Items: []command.HandoverItemInput{
			{DocumentID: "", Selected: true, Note: "跳过"},
			{DocumentID: "doc-1", Selected: true, Note: "保留"},
		},
	})
	if err != nil {
		t.Fatalf("UpdateHandoverItems unexpected error: %v", err)
	}
	if got["id"] != "handover-1" {
		t.Fatalf("UpdateHandoverItems got %#v", got)
	}
	assertExpectations(t, mock)
}

func TestActionRepositoryApplyHandoverCompleteUpdatesDocuments(t *testing.T) {
	ctx := shared.WithRequestID(context.Background(), "req-complete")
	db, mock := newMockDB(t)
	repo := NewActionRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT status::text").
		WithArgs("handover-1").
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("pending_confirm"))
	mock.ExpectExec("UPDATE graduation_handovers").
		WithArgs("handover-1", "completed", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT receiver_user_id::text").
		WithArgs("handover-1").
		WillReturnRows(sqlmock.NewRows([]string{"receiver_user_id"}).AddRow("u-receiver"))
	mock.ExpectExec("UPDATE documents d").
		WithArgs("handover-1", "u-receiver", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 2))
	expectAuditInsert(mock, "", "", "actor-1", "handover_complete", "req-complete")
	mock.ExpectCommit()

	got, err := repo.ApplyHandover(ctx, command.HandoverActionInput{
		HandoverID: "handover-1",
		Action:     "complete",
		Note:       "完成",
		ActorID:    "actor-1",
	})
	if err != nil {
		t.Fatalf("ApplyHandover unexpected error: %v", err)
	}
	if got["status"] != "completed" || got["note"] != "完成" {
		t.Fatalf("ApplyHandover got %#v", got)
	}
	assertExpectations(t, mock)
}

func TestActionRepositoryApplyHandoverInvalidTransitionRollsBack(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewActionRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT status::text").
		WithArgs("handover-1").
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("generated"))
	mock.ExpectRollback()

	_, err := repo.ApplyHandover(ctx, command.HandoverActionInput{
		HandoverID: "handover-1",
		Action:     "complete",
		ActorID:    "actor-1",
	})
	if !errors.Is(err, service.ErrInvalidTransition) {
		t.Fatalf("ApplyHandover err=%v, want ErrInvalidTransition", err)
	}
	assertExpectations(t, mock)
}

func expectAuditInsert(
	mock sqlmock.Sqlmock,
	documentID any,
	versionID any,
	userID string,
	actionType string,
	requestID string,
) {
	mock.ExpectExec("INSERT INTO audit_events").
		WithArgs(
			sqlmock.AnyArg(),
			documentID,
			versionID,
			userID,
			actionType,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			requestID,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
}
