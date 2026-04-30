package memory

import (
	"context"
	"errors"
	"testing"

	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/domain/task"
	"digidocs-mgt/backend-go/internal/service"
)

func TestAssistantRepositoryConversationLifecycle(t *testing.T) {
	ctx := context.Background()
	repo := NewAssistantRepository()

	conv, err := repo.CreateConversation(ctx, "project", "project-1", map[string]any{"project_id": "project-1"}, "title", "actor-1")
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	if conv.ScopeType != "project" || conv.ScopeID != "project-1" || conv.CreatedBy != "actor-1" {
		t.Fatalf("unexpected conversation: %+v", conv)
	}

	got, err := repo.GetConversation(ctx, conv.ID)
	if err != nil || got.ID != conv.ID {
		t.Fatalf("get conversation = (%+v, %v)", got, err)
	}
	if _, err := repo.GetConversation(ctx, "missing"); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected missing conversation error, got %v", err)
	}

	items, err := repo.ListConversations(ctx, query.AssistantConversationFilter{CreatedBy: "actor-1"})
	if err != nil || len(items) != 1 {
		t.Fatalf("list conversations = (%+v, %v)", items, err)
	}
	items, err = repo.ListConversations(ctx, query.AssistantConversationFilter{ScopeType: "document"})
	if err != nil || len(items) != 0 {
		t.Fatalf("scope filtered conversations = (%+v, %v)", items, err)
	}

	if err := repo.ArchiveConversation(ctx, conv.ID, true); err != nil {
		t.Fatalf("archive conversation: %v", err)
	}
	items, err = repo.ListConversations(ctx, query.AssistantConversationFilter{})
	if err != nil || len(items) != 0 {
		t.Fatalf("archived conversation should be hidden: (%+v, %v)", items, err)
	}
	items, err = repo.ListConversations(ctx, query.AssistantConversationFilter{IncludeArchived: true})
	if err != nil || len(items) != 1 || items[0].ArchivedAt == "" {
		t.Fatalf("archived conversation should be included: (%+v, %v)", items, err)
	}
	if err := repo.ArchiveConversation(ctx, conv.ID, false); err != nil {
		t.Fatalf("restore conversation: %v", err)
	}
	if err := repo.ArchiveConversation(ctx, "missing", true); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected archive missing error, got %v", err)
	}
}

func TestAssistantRepositoryRequestCompletionSuggestionsAndMessages(t *testing.T) {
	ctx := context.Background()
	repo := NewAssistantRepository()
	conv, err := repo.CreateConversation(ctx, "document", "doc-1", map[string]any{"project_id": "project-1", "document_id": "doc-1"}, "title", "actor-1")
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	message := task.Message{
		RequestID:   "req-1",
		TaskType:    task.TaskTypeDocumentSummarize,
		RelatedType: "document",
		RelatedID:   "doc-1",
		Payload: map[string]any{
			"conversation_id": conv.ID,
			"question":        "总结文档",
			"scope":           map[string]any{"project_id": "project-1", "document_id": "doc-1"},
			"memory_sources": []any{
				map[string]any{"type": "conversation_messages", "count": 1},
				"bad",
			},
		},
	}
	if err := repo.CreateAssistantRequest(ctx, message, "actor-1"); err != nil {
		t.Fatalf("create request: %v", err)
	}
	messages, err := repo.ListConversationMessages(ctx, conv.ID)
	if err != nil || len(messages) != 1 || messages[0].Role != "user" || messages[0].Content != "总结文档" {
		t.Fatalf("user message = (%+v, %v)", messages, err)
	}

	if err := repo.CompleteAssistantRequest(ctx, task.Result{
		RequestID: "req-1",
		Status:    "completed",
		Output: map[string]any{
			"answer":       "回答正文",
			"summary_text": "摘要正文",
			"model":        "model-a",
			"skill_name":   "document.summarize",
			"request_id":   "upstream-1",
			"source_scope": map[string]any{"document_id": "doc-1"},
			"suggestions": []any{
				map[string]any{
					"title":           "建议一",
					"content":         "建议正文",
					"suggestion_type": "risk_alert",
					"confidence":      0.75,
				},
				map[string]any{"title": "empty-content"},
				"bad",
			},
		},
	}); err != nil {
		t.Fatalf("complete request: %v", err)
	}

	req, err := repo.GetAssistantRequest(ctx, "req-1")
	if err != nil {
		t.Fatalf("get request: %v", err)
	}
	if req.Status != "completed" || req.Model != "model-a" || req.UpstreamRequestID != "upstream-1" {
		t.Fatalf("unexpected request item: %+v", req)
	}
	if len(req.MemorySources) != 1 {
		t.Fatalf("unexpected memory sources: %+v", req.MemorySources)
	}

	requests, total, err := repo.ListAssistantRequests(ctx, query.AssistantRequestFilter{
		RequestType: string(task.TaskTypeDocumentSummarize),
		RelatedType: "document",
		RelatedID:   "doc-1",
		Status:      "completed",
		Keyword:     "总结",
		Page:        1,
		PageSize:    1,
	})
	if err != nil || total != 1 || len(requests) != 1 {
		t.Fatalf("list requests = (%+v, %d, %v)", requests, total, err)
	}
	requests, total, err = repo.ListAssistantRequests(ctx, query.AssistantRequestFilter{Page: 2, PageSize: 1})
	if err != nil || total != 1 || len(requests) != 0 {
		t.Fatalf("paged requests = (%+v, %d, %v)", requests, total, err)
	}

	suggestions, err := repo.ListSuggestions(ctx, query.AssistantSuggestionFilter{RelatedType: "document", RelatedID: "doc-1", Status: "pending"})
	if err != nil || len(suggestions) != 2 {
		t.Fatalf("list suggestions = (%+v, %v)", suggestions, err)
	}
	summaryID := "req-1-summary"
	confirmed, err := repo.ConfirmSuggestion(ctx, summaryID, "actor-2", "ok")
	if err != nil || confirmed["status"] != "confirmed" {
		t.Fatalf("confirm suggestion = (%+v, %v)", confirmed, err)
	}
	dismissed, err := repo.DismissSuggestion(ctx, "req-1-suggestion-1", "actor-2", "no")
	if err != nil || dismissed["status"] != "dismissed" {
		t.Fatalf("dismiss suggestion = (%+v, %v)", dismissed, err)
	}
	if _, err := repo.ConfirmSuggestion(ctx, "missing", "actor", ""); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected confirm missing error, got %v", err)
	}
	if _, err := repo.DismissSuggestion(ctx, "missing", "actor", ""); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected dismiss missing error, got %v", err)
	}

	messages, err = repo.ListConversationMessages(ctx, conv.ID)
	if err != nil || len(messages) != 2 || messages[1].Role != "assistant" || messages[1].Content != "回答正文" {
		t.Fatalf("conversation messages = (%+v, %v)", messages, err)
	}
	if _, err := repo.ListConversationMessages(ctx, "missing"); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected list missing messages error, got %v", err)
	}
}

func TestAssistantRepositoryFailedAndExtractedTextPaths(t *testing.T) {
	ctx := context.Background()
	repo := NewAssistantRepository()

	if err := repo.CreateAssistantRequest(ctx, task.Message{
		RequestID:   "extract-1",
		TaskType:    task.TaskTypeDocumentExtractText,
		RelatedType: "document",
		RelatedID:   "doc-1",
		Payload:     map[string]any{},
	}, "actor-1"); err != nil {
		t.Fatalf("create extract request: %v", err)
	}
	if err := repo.CompleteAssistantRequest(ctx, task.Result{
		RequestID: "extract-1",
		Status:    "completed",
		Output:    map[string]any{"extracted_text": "正文"},
	}); err != nil {
		t.Fatalf("complete extract request: %v", err)
	}
	text, err := repo.GetLatestDocumentExtractedText(ctx, "doc-1")
	if err != nil || text != "正文" {
		t.Fatalf("latest text = (%q, %v)", text, err)
	}

	conv, err := repo.CreateConversation(ctx, "project", "project-1", map[string]any{"project_id": "project-1"}, "title", "actor-1")
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	if err := repo.CreateAssistantRequest(ctx, task.Message{
		RequestID:   "failed-1",
		TaskType:    task.TaskTypeAssistantAsk,
		RelatedType: "project",
		RelatedID:   "project-1",
		Payload:     map[string]any{"conversation_id": conv.ID, "question": "Q"},
	}, "actor-1"); err != nil {
		t.Fatalf("create failed request: %v", err)
	}
	if err := repo.CompleteAssistantRequest(ctx, task.Result{RequestID: "failed-1", Status: "failed", ErrorMessage: "boom"}); err != nil {
		t.Fatalf("complete failed request: %v", err)
	}
	messages, err := repo.ListConversationMessages(ctx, conv.ID)
	if err != nil || len(messages) != 2 || messages[1].Content != "boom" {
		t.Fatalf("failed messages = (%+v, %v)", messages, err)
	}

	if err := repo.CompleteAssistantRequest(ctx, task.Result{RequestID: "missing", Status: "completed"}); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected complete missing error, got %v", err)
	}
}
