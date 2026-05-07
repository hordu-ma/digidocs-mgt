package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"digidocs-mgt/backend-go/internal/domain/task"
	"digidocs-mgt/backend-go/internal/service"
)

func TestAssistantRepositoryCreateConversationAndListMessages(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewAssistantRepository(db)

	mock.ExpectExec("INSERT INTO assistant_conversations").
		WithArgs(sqlmock.AnyArg(), "document", "doc-1", `{"document_id":"doc-1"}`, "讨论标题", "actor-1", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	conversation, err := repo.CreateConversation(ctx, "document", "doc-1", map[string]any{"document_id": "doc-1"}, "讨论标题", "actor-1")
	if err != nil || conversation.ScopeType != "document" || conversation.Title != "讨论标题" {
		t.Fatalf("CreateConversation got %#v err=%v", conversation, err)
	}

	mock.ExpectQuery("FROM assistant_conversation_messages").
		WithArgs("conv-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "conversation_id", "role", "content", "request_id", "metadata", "created_by", "created_at"}).
			AddRow("msg-1", "conv-1", "user", "请总结", "req-1", `{"source_scope":{"document_id":"doc-1"}}`, "actor-1", "2026-05-01T00:00:00Z"))
	messages, err := repo.ListConversationMessages(ctx, "conv-1")
	if err != nil || len(messages) != 1 || messages[0].Content != "请总结" {
		t.Fatalf("ListConversationMessages got %#v err=%v", messages, err)
	}
	assertExpectations(t, mock)
}

func TestAssistantRepositoryCreateRequestWithConversationMessage(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewAssistantRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO assistant_requests").
		WithArgs(
			"req-1",
			string(task.TaskTypeAssistantAsk),
			"document",
			"doc-1",
			"conv-1",
			sqlmock.AnyArg(),
			"actor-1",
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO assistant_conversation_messages").
		WithArgs(sqlmock.AnyArg(), "conv-1", "user", "如何归档？", "req-1", sqlmock.AnyArg(), "actor-1", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE assistant_conversations").
		WithArgs("conv-1", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.CreateAssistantRequest(ctx, task.Message{
		RequestID:   "req-1",
		TaskType:    task.TaskTypeAssistantAsk,
		RelatedType: "document",
		RelatedID:   "doc-1",
		Payload: map[string]any{
			"conversation_id": "conv-1",
			"question":        "如何归档？",
			"scope":           map[string]any{"document_id": "doc-1"},
		},
	}, "actor-1")
	if err != nil {
		t.Fatalf("CreateAssistantRequest unexpected err=%v", err)
	}
	assertExpectations(t, mock)
}

func TestAssistantRepositoryCompleteRequestUpdatesProjectionSuggestionsAndConversation(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewAssistantRepository(db)
	createdAt := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery("FROM assistant_requests\\s+WHERE id::text = \\$1\\s+FOR UPDATE").
		WithArgs("req-1").
		WillReturnRows(sqlmock.NewRows([]string{"request_type", "related_type", "related_id", "conversation_id", "payload", "created_at"}).
			AddRow(string(task.TaskTypeDocumentSummarize), "document", "doc-1", "conv-1", `{"version_id":"ver-1","scope":{"document_id":"doc-1"}}`, createdAt))
	mock.ExpectExec("UPDATE assistant_requests").
		WithArgs("req-1", "completed", "", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM assistant_suggestions").
		WithArgs("req-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE document_versions").
		WithArgs("ver-1", "文档摘要").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO assistant_suggestions").
		WithArgs(sqlmock.AnyArg(), "document", "doc-1", "document_summary", "pending", "文档摘要", "文档摘要", sqlmock.AnyArg(), nil, "req-1", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO assistant_conversation_messages").
		WithArgs(sqlmock.AnyArg(), "conv-1", "assistant", "这是回答", "req-1", sqlmock.AnyArg(), "", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE assistant_conversations").
		WithArgs("conv-1", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.CompleteAssistantRequest(ctx, task.Result{
		RequestID: "req-1",
		Status:    "completed",
		Output: map[string]any{
			"summary_text": "文档摘要",
			"answer":       "这是回答",
			"model":        "model-a",
		},
	})
	if err != nil {
		t.Fatalf("CompleteAssistantRequest unexpected err=%v", err)
	}
	assertExpectations(t, mock)
}

func TestAssistantRepositorySuggestionStatusAndLatestTextBranches(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewAssistantRepository(db)

	mock.ExpectQuery("UPDATE assistant_suggestions").
		WithArgs("sug-1", "actor-1", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("sug-1"))
	confirmed, err := repo.ConfirmSuggestion(ctx, "sug-1", "actor-1", "采用")
	if err != nil || confirmed["status"] != "confirmed" || confirmed["note"] != "采用" {
		t.Fatalf("ConfirmSuggestion got %#v err=%v", confirmed, err)
	}

	mock.ExpectQuery("UPDATE assistant_suggestions").
		WithArgs("missing", "actor-1", sqlmock.AnyArg()).
		WillReturnError(sqlmock.ErrCancelled)
	if _, err := repo.DismissSuggestion(ctx, "missing", "actor-1", "忽略"); !errors.Is(err, sqlmock.ErrCancelled) {
		t.Fatalf("DismissSuggestion err=%v, want propagated sql error", err)
	}

	mock.ExpectQuery("FROM assistant_requests").
		WithArgs(sqlmock.AnyArg(), "doc-1").
		WillReturnRows(sqlmock.NewRows([]string{"output"}).AddRow(`{"extracted_text":"正文"}`))
	text, err := repo.GetLatestDocumentExtractedText(ctx, "doc-1")
	if err != nil || text != "正文" {
		t.Fatalf("GetLatestDocumentExtractedText text=%q err=%v", text, err)
	}

	mock.ExpectQuery("FROM assistant_requests").
		WithArgs(sqlmock.AnyArg(), "missing").
		WillReturnError(sqlmock.ErrCancelled)
	if _, err := repo.GetLatestDocumentExtractedText(ctx, "missing"); !errors.Is(err, sqlmock.ErrCancelled) {
		t.Fatalf("GetLatestDocumentExtractedText err=%v, want propagated sql error", err)
	}

	db2, mock2 := newMockDB(t)
	repo2 := NewAssistantRepository(db2)
	mock2.ExpectQuery("UPDATE assistant_suggestions").
		WillReturnError(sqlmock.ErrCancelled)
	if _, err := repo2.ConfirmSuggestion(ctx, "sug-1", "actor-1", "采用"); !errors.Is(err, sqlmock.ErrCancelled) {
		t.Fatalf("ConfirmSuggestion err=%v, want propagated sql error", err)
	}
	assertExpectations(t, mock)
	assertExpectations(t, mock2)
}

func TestAssistantRepositoryGetRequestNotFound(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewAssistantRepository(db)

	mock.ExpectQuery("FROM assistant_requests\\s+WHERE id::text = \\$1").
		WithArgs("missing").
		WillReturnError(sqlmock.ErrCancelled)
	if _, err := repo.GetAssistantRequest(ctx, "missing"); !errors.Is(err, sqlmock.ErrCancelled) {
		t.Fatalf("GetAssistantRequest err=%v, want propagated sql error", err)
	}

	db2, mock2 := newMockDB(t)
	repo2 := NewAssistantRepository(db2)
	mock2.ExpectQuery("FROM assistant_requests\\s+WHERE id::text = \\$1").
		WithArgs("missing").
		WillReturnError(sqlmock.ErrCancelled)
	if _, err := repo2.GetAssistantRequest(ctx, "missing"); errors.Is(err, service.ErrNotFound) {
		t.Fatalf("GetAssistantRequest err=%v, did not expect ErrNotFound for non sql.ErrNoRows", err)
	}
	assertExpectations(t, mock)
	assertExpectations(t, mock2)
}
