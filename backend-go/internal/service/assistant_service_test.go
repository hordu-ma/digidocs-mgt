package service

import (
	"context"
	"errors"
	"testing"

	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/domain/task"
)

func TestAssistantServiceAskCreatesConversationQueuesTaskAndMemory(t *testing.T) {
	ctx := context.Background()
	repo := newFakeAssistantRepo()
	publisher := &fakePublisher{}
	svc := NewAssistantService(publisher, repo, nil)

	// Seed historical memory in another conversation with the same project scope.
	oldConv, _ := repo.CreateConversation(ctx, "project", "project-1", map[string]any{"project_id": "project-1"}, "old", "actor-1")
	_ = repo.CreateAssistantRequest(ctx, task.Message{
		RequestID:   "old-req",
		TaskType:    task.TaskTypeAssistantAsk,
		RelatedType: "project",
		RelatedID:   "project-1",
		Payload:     map[string]any{"conversation_id": oldConv.ID, "question": "old question"},
	}, "actor-1")
	_ = repo.CompleteAssistantRequest(ctx, task.Result{
		RequestID: "old-req",
		Status:    "completed",
		Output:    map[string]any{"answer": "old answer", "summary_text": "confirmed text"},
	})
	suggestions, _ := repo.ListSuggestions(ctx, query.AssistantSuggestionFilter{Status: "pending"})
	_, _ = repo.ConfirmSuggestion(ctx, suggestions[0].ID, "actor-1", "ok")

	result, err := svc.Ask(ctx, map[string]any{
		"question":   "  What next?  ",
		"project_id": "project-1",
	}, "actor-2")
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if result.Status != "queued" || result.ConversationID == "" || result.RequestID == "" {
		t.Fatalf("unexpected ask result: %+v", result)
	}
	if len(publisher.messages) != 1 {
		t.Fatalf("expected one published message, got %+v", publisher.messages)
	}
	payload := publisher.messages[0].Payload
	if payload["conversation_id"] != result.ConversationID {
		t.Fatalf("conversation id not injected into payload: %+v", payload)
	}
	memory := payload["memory"].(map[string]any)
	if memory["scope_id"] != "project-1" {
		t.Fatalf("unexpected memory scope: %+v", memory)
	}
	if _, ok := memory["confirmed_suggestions"]; !ok {
		t.Fatalf("expected confirmed suggestions in memory: %+v", memory)
	}
	if _, ok := memory["historical_answers"]; !ok {
		t.Fatalf("expected historical answers in memory: %+v", memory)
	}
}

func TestAssistantServiceAskExistingConversationAndErrors(t *testing.T) {
	ctx := context.Background()
	repo := newFakeAssistantRepo()
	publisher := &fakePublisher{}
	svc := NewAssistantService(publisher, repo, nil)

	if _, err := svc.Ask(ctx, map[string]any{"question": " "}, "actor"); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected question validation error, got %v", err)
	}
	if _, err := svc.Ask(ctx, map[string]any{"question": "Q"}, "actor"); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected scope validation error, got %v", err)
	}

	conv, err := svc.CreateConversation(ctx, map[string]any{"project_id": "project-1"}, "title", "actor")
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	result, err := svc.Ask(ctx, map[string]any{"question": "Q", "conversation_id": conv.ID, "project_id": "project-1"}, "actor")
	if err != nil {
		t.Fatalf("ask existing conversation: %v", err)
	}
	if result.ConversationID != conv.ID {
		t.Fatalf("expected existing conversation id, got %+v", result)
	}
	if _, err := svc.Ask(ctx, map[string]any{"question": "Q", "conversation_id": conv.ID, "project_id": "project-2"}, "actor"); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected scope mismatch validation error, got %v", err)
	}

	repo.createRequestErr = errors.New("create request failed")
	if _, err := svc.QueueTask(ctx, task.TaskTypeAssistantAsk, "project", "project-1", nil, "actor"); err == nil {
		t.Fatal("expected create request error")
	}
	repo.createRequestErr = nil
	publisher.err = errors.New("publish failed")
	if _, err := svc.QueueTask(ctx, task.TaskTypeAssistantAsk, "project", "project-1", nil, "actor"); err == nil {
		t.Fatal("expected publish error")
	}
}

func TestAssistantServiceDelegatesRepositoryMethods(t *testing.T) {
	ctx := context.Background()
	repo := newFakeAssistantRepo()
	publisher := &fakePublisher{}
	svc := NewAssistantService(publisher, repo, nil)

	conv, err := svc.CreateConversation(ctx, map[string]any{"project_id": "project-1"}, "title", "actor")
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	if _, err := svc.GetConversation(ctx, conv.ID); err != nil {
		t.Fatalf("get conversation: %v", err)
	}
	if items, err := svc.ListConversations(ctx, query.AssistantConversationFilter{}); err != nil || len(items) != 1 {
		t.Fatalf("list conversations = (%+v, %v)", items, err)
	}
	if err := svc.ArchiveConversation(ctx, conv.ID, true); err != nil {
		t.Fatalf("archive conversation: %v", err)
	}
	if _, err := svc.ListConversationMessages(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected missing conversation before listing messages, got %v", err)
	}

	message, err := svc.QueueTask(ctx, task.TaskTypeDocumentExtractText, "document", "doc-1", map[string]any{}, "actor")
	if err != nil {
		t.Fatalf("queue task: %v", err)
	}
	if err := svc.ReceiveResult(ctx, task.Result{RequestID: message.RequestID, Status: "completed", Output: map[string]any{"extracted_text": "text"}}); err != nil {
		t.Fatalf("receive result: %v", err)
	}
	if _, err := svc.GetRequest(ctx, message.RequestID); err != nil {
		t.Fatalf("get request: %v", err)
	}
	if items, total, err := svc.ListRequests(ctx, query.AssistantRequestFilter{}); err != nil || total == 0 || len(items) == 0 {
		t.Fatalf("list requests = (%+v, %d, %v)", items, total, err)
	}
	if text, err := svc.GetLatestDocumentExtractedText(ctx, "doc-1"); err != nil || text != "text" {
		t.Fatalf("latest extracted text = (%q, %v)", text, err)
	}

	repo.suggestions["suggestion-1"] = query.AssistantSuggestionItem{
		ID:          "suggestion-1",
		RelatedType: "document",
		RelatedID:   "doc-1",
		Status:      "pending",
		Content:     "content",
	}
	if suggestions, err := svc.ListSuggestions(ctx, query.AssistantSuggestionFilter{Status: "pending"}); err != nil || len(suggestions) != 1 {
		t.Fatalf("list suggestions = (%+v, %v)", suggestions, err)
	}
	if result, err := svc.ConfirmSuggestion(ctx, "suggestion-1", "actor", "ok"); err != nil || result["status"] != "confirmed" {
		t.Fatalf("confirm suggestion = (%+v, %v)", result, err)
	}
	if result, err := svc.DismissSuggestion(ctx, "suggestion-1", "actor", "no"); err != nil || result["status"] != "dismissed" {
		t.Fatalf("dismiss suggestion = (%+v, %v)", result, err)
	}
}

func TestAssistantMemoryHelpers(t *testing.T) {
	long := "这是一个很长很长很长很长很长很长很长很长很长的问题"
	if title := buildConversationTitle(long); len([]rune(title)) != 27 {
		t.Fatalf("expected truncated title, got %q", title)
	}
	if title := buildConversationTitle(" "); title != "AI 会话" {
		t.Fatalf("blank title = %q", title)
	}
	if got := tailConversationMessages(nil, 6); got != nil {
		t.Fatalf("nil message tail = %+v", got)
	}
	if got := tailConversationMessages([]query.AssistantConversationMessageItem{{Role: "user", Content: "1"}, {Role: "assistant", Content: "2"}}, 1); len(got) != 1 || got[0]["content"] != "2" {
		t.Fatalf("message tail = %+v", got)
	}
	if got := takeSuggestionTail([]query.AssistantSuggestionItem{{ID: "1"}, {ID: "2"}}, 1); len(got) != 1 || got[0].ID != "1" {
		t.Fatalf("suggestion tail = %+v", got)
	}
	if got := mergeScopes(map[string]any{"project_id": "p1"}, map[string]any{"document_id": "d1", "empty": ""}); got["project_id"] != "p1" || got["document_id"] != "d1" {
		t.Fatalf("merged scope = %+v", got)
	}
	if documentProjectID(nil) != "" {
		t.Fatal("nil document project id should be empty")
	}
}

type fakePublisher struct {
	messages []task.Message
	err      error
}

func (p *fakePublisher) Publish(_ context.Context, message task.Message) error {
	if p.err != nil {
		return p.err
	}
	p.messages = append(p.messages, message)
	return nil
}

type fakeAssistantRepo struct {
	conversations    map[string]query.AssistantConversationItem
	messages         map[string][]query.AssistantConversationMessageItem
	requests         map[string]query.AssistantRequestItem
	suggestions      map[string]query.AssistantSuggestionItem
	createRequestErr error
	seq              int
}

func newFakeAssistantRepo() *fakeAssistantRepo {
	return &fakeAssistantRepo{
		conversations: map[string]query.AssistantConversationItem{},
		messages:      map[string][]query.AssistantConversationMessageItem{},
		requests:      map[string]query.AssistantRequestItem{},
		suggestions:   map[string]query.AssistantSuggestionItem{},
	}
}

func (r *fakeAssistantRepo) nextID(prefix string) string {
	r.seq++
	return prefix + "-" + string(rune('a'+r.seq))
}

func (r *fakeAssistantRepo) CreateConversation(_ context.Context, scopeType string, scopeID string, sourceScope map[string]any, title string, actorID string) (*query.AssistantConversationItem, error) {
	item := query.AssistantConversationItem{
		ID:            r.nextID("conv"),
		ScopeType:     scopeType,
		ScopeID:       scopeID,
		SourceScope:   clonePayload(sourceScope),
		Title:         title,
		CreatedBy:     actorID,
		CreatedAt:     "2026-04-30T00:00:00Z",
		LastMessageAt: "2026-04-30T00:00:00Z",
	}
	r.conversations[item.ID] = item
	return &item, nil
}

func (r *fakeAssistantRepo) GetConversation(_ context.Context, conversationID string) (*query.AssistantConversationItem, error) {
	item, ok := r.conversations[conversationID]
	if !ok {
		return nil, ErrNotFound
	}
	return &item, nil
}

func (r *fakeAssistantRepo) ListConversations(_ context.Context, filter query.AssistantConversationFilter) ([]query.AssistantConversationItem, error) {
	items := make([]query.AssistantConversationItem, 0)
	for _, item := range r.conversations {
		if !filter.IncludeArchived && item.ArchivedAt != "" {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *fakeAssistantRepo) ArchiveConversation(_ context.Context, conversationID string, archive bool) error {
	item, ok := r.conversations[conversationID]
	if !ok {
		return ErrNotFound
	}
	if archive {
		item.ArchivedAt = "2026-04-30T00:01:00Z"
	} else {
		item.ArchivedAt = ""
	}
	r.conversations[conversationID] = item
	return nil
}

func (r *fakeAssistantRepo) ListConversationMessages(_ context.Context, conversationID string) ([]query.AssistantConversationMessageItem, error) {
	if _, ok := r.conversations[conversationID]; !ok {
		return nil, ErrNotFound
	}
	return append([]query.AssistantConversationMessageItem(nil), r.messages[conversationID]...), nil
}

func (r *fakeAssistantRepo) CreateAssistantRequest(_ context.Context, message task.Message, actorID string) error {
	if r.createRequestErr != nil {
		return r.createRequestErr
	}
	item := query.AssistantRequestItem{
		ID:             message.RequestID,
		RequestType:    string(message.TaskType),
		RelatedType:    message.RelatedType,
		RelatedID:      message.RelatedID,
		ConversationID: stringValue(message.Payload["conversation_id"]),
		Status:         "pending",
		Question:       stringValue(message.Payload["question"]),
		SourceScope:    extractScope(message.Payload),
		MemorySources:  memorySourcesFromPayload(message.Payload),
		Output:         map[string]any{},
		CreatedAt:      "2026-04-30T00:00:00Z",
	}
	r.requests[item.ID] = item
	if item.ConversationID != "" {
		r.messages[item.ConversationID] = append(r.messages[item.ConversationID], query.AssistantConversationMessageItem{
			ID:             r.nextID("msg"),
			ConversationID: item.ConversationID,
			Role:           "user",
			Content:        item.Question,
			RequestID:      item.ID,
			CreatedBy:      actorID,
			CreatedAt:      item.CreatedAt,
		})
	}
	return nil
}

func (r *fakeAssistantRepo) CompleteAssistantRequest(_ context.Context, result task.Result) error {
	item, ok := r.requests[result.RequestID]
	if !ok {
		return ErrNotFound
	}
	item.Status = result.Status
	item.Output = clonePayload(result.Output)
	item.CompletedAt = "2026-04-30T00:00:01Z"
	r.requests[item.ID] = item
	if text := stringValue(result.Output["summary_text"]); text != "" {
		r.suggestions[item.ID+"-summary"] = query.AssistantSuggestionItem{
			ID:             item.ID + "-summary",
			RelatedType:    item.RelatedType,
			RelatedID:      item.RelatedID,
			SuggestionType: "document_summary",
			Status:         "pending",
			Title:          "文档摘要",
			Content:        text,
			RequestID:      item.ID,
			GeneratedAt:    item.CompletedAt,
		}
	}
	if item.ConversationID != "" && stringValue(result.Output["answer"]) != "" {
		r.messages[item.ConversationID] = append(r.messages[item.ConversationID], query.AssistantConversationMessageItem{
			ID:             r.nextID("msg"),
			ConversationID: item.ConversationID,
			Role:           "assistant",
			Content:        stringValue(result.Output["answer"]),
			RequestID:      item.ID,
			CreatedAt:      item.CompletedAt,
		})
	}
	return nil
}

func (r *fakeAssistantRepo) ListAssistantRequests(_ context.Context, filter query.AssistantRequestFilter) ([]query.AssistantRequestItem, int, error) {
	items := make([]query.AssistantRequestItem, 0)
	for _, item := range r.requests {
		if filter.RequestType != "" && item.RequestType != filter.RequestType {
			continue
		}
		if filter.RelatedType != "" && item.RelatedType != filter.RelatedType {
			continue
		}
		if filter.RelatedID != "" && item.RelatedID != filter.RelatedID {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		items = append(items, item)
	}
	return items, len(items), nil
}

func (r *fakeAssistantRepo) GetAssistantRequest(_ context.Context, requestID string) (*query.AssistantRequestItem, error) {
	item, ok := r.requests[requestID]
	if !ok {
		return nil, ErrNotFound
	}
	return &item, nil
}

func (r *fakeAssistantRepo) GetLatestDocumentExtractedText(_ context.Context, documentID string) (string, error) {
	for _, item := range r.requests {
		if item.RelatedType == "document" && item.RelatedID == documentID {
			if text := stringValue(item.Output["extracted_text"]); text != "" {
				return text, nil
			}
		}
	}
	return "", nil
}

func (r *fakeAssistantRepo) ListSuggestions(_ context.Context, filter query.AssistantSuggestionFilter) ([]query.AssistantSuggestionItem, error) {
	items := make([]query.AssistantSuggestionItem, 0)
	for _, item := range r.suggestions {
		if filter.RelatedType != "" && item.RelatedType != filter.RelatedType {
			continue
		}
		if filter.RelatedID != "" && item.RelatedID != filter.RelatedID {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *fakeAssistantRepo) ConfirmSuggestion(_ context.Context, suggestionID string, actorID string, note string) (map[string]any, error) {
	item, ok := r.suggestions[suggestionID]
	if !ok {
		return nil, ErrNotFound
	}
	item.Status = "confirmed"
	r.suggestions[suggestionID] = item
	return map[string]any{"id": suggestionID, "status": "confirmed", "confirmed_by": actorID, "note": note}, nil
}

func (r *fakeAssistantRepo) DismissSuggestion(_ context.Context, suggestionID string, actorID string, reason string) (map[string]any, error) {
	item, ok := r.suggestions[suggestionID]
	if !ok {
		return nil, ErrNotFound
	}
	item.Status = "dismissed"
	r.suggestions[suggestionID] = item
	return map[string]any{"id": suggestionID, "status": "dismissed", "dismissed_by": actorID, "reason": reason}, nil
}

func memorySourcesFromPayload(payload map[string]any) []map[string]any {
	raw, _ := payload["memory_sources"].([]map[string]any)
	return raw
}
