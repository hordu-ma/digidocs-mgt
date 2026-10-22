package service

import (
	"context"
	"fmt"
	"strings"

	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/domain/task"
	"digidocs-mgt/backend-go/internal/queue"
	"digidocs-mgt/backend-go/internal/repository"
)

type AssistantAskResult struct {
	RequestID      string
	ConversationID string
	Question       string
	Status         string
	SourceScope    map[string]any
	MemorySources  []map[string]any
}

type AssistantService struct {
	publisher   queue.Publisher
	repo        repository.AssistantRepository
	documents   repository.DocumentReader
	versions    repository.VersionReader
	permissions PermissionService
}

func NewAssistantService(
	publisher queue.Publisher,
	repo repository.AssistantRepository,
	documents repository.DocumentReader,
	versions repository.VersionReader,
	permissions ...PermissionService,
) AssistantService {
	permissionService := PermissionService{}
	if len(permissions) > 0 {
		permissionService = permissions[0]
	}
	return AssistantService{
		publisher:   publisher,
		repo:        repo,
		documents:   documents,
		versions:    versions,
		permissions: permissionService,
	}
}

// Configured reports whether the service was built with its dependencies.
// A zero-value AssistantService is not configured; handlers use this to skip
// optional assistant side effects (e.g. auto-queuing text extraction).
func (s AssistantService) Configured() bool {
	return s.publisher != nil
}

func (s AssistantService) Ask(
	ctx context.Context,
	payload map[string]any,
	actorID string,
	actorRole string,
) (AssistantAskResult, error) {
	question := strings.TrimSpace(stringValue(payload["question"]))
	if question == "" {
		return AssistantAskResult{}, fmt.Errorf("%w: question is required", ErrValidation)
	}

	scopeInput := extractScope(payload)
	payload = assistantPayload(payload, "question", "conversation_id", "skill_name")
	conversationID := strings.TrimSpace(stringValue(payload["conversation_id"]))

	var conversation *query.AssistantConversationItem
	if conversationID != "" {
		if err := s.permissions.EnsureViewConversation(ctx, actorID, actorRole, conversationID); err != nil {
			return AssistantAskResult{}, err
		}
		item, err := s.repo.GetConversation(ctx, conversationID)
		if err != nil {
			return AssistantAskResult{}, err
		}
		conversation = item
		scopeInput = mergeScopes(item.SourceScope, scopeInput)
	}

	scope, err := s.normalizeScope(ctx, scopeInput)
	if err != nil {
		return AssistantAskResult{}, err
	}
	if err := s.ensureViewScope(ctx, actorID, actorRole, scope); err != nil {
		return AssistantAskResult{}, err
	}

	if conversation == nil {
		item, err := s.repo.CreateConversation(
			ctx,
			scope.ScopeType,
			scope.ScopeID,
			scope.SourceScope,
			buildConversationTitle(question),
			actorID,
		)
		if err != nil {
			return AssistantAskResult{}, err
		}
		conversation = item
	} else if conversation.ScopeType != scope.ScopeType || conversation.ScopeID != scope.ScopeID {
		return AssistantAskResult{}, fmt.Errorf("%w: conversation scope mismatch", ErrValidation)
	}

	memory, sources, err := s.buildMemorySnapshot(ctx, conversation.ID, scope)
	if err != nil {
		return AssistantAskResult{}, err
	}

	payload["conversation_id"] = conversation.ID
	payload["scope"] = scope.SourceScope
	payload["memory"] = memory
	payload["memory_sources"] = sources

	message, err := s.QueueTask(
		ctx,
		task.TaskTypeAssistantAsk,
		scope.ScopeType,
		scope.ScopeID,
		payload,
		actorID,
	)
	if err != nil {
		return AssistantAskResult{}, err
	}

	return AssistantAskResult{
		RequestID:      message.RequestID,
		ConversationID: conversation.ID,
		Question:       question,
		Status:         "queued",
		SourceScope:    scope.SourceScope,
		MemorySources:  sources,
	}, nil
}

func (s AssistantService) QueueTask(
	ctx context.Context,
	taskType task.TaskType,
	relatedType string,
	relatedID string,
	payload map[string]any,
	actorID string,
) (task.Message, error) {
	message := task.Message{
		RequestID:   newRequestID(),
		TaskType:    taskType,
		RelatedType: relatedType,
		RelatedID:   relatedID,
		Payload:     payload,
	}
	if message.Payload == nil {
		message.Payload = map[string]any{}
	}

	if err := s.repo.CreateAssistantRequest(ctx, message, actorID); err != nil {
		return task.Message{}, err
	}
	if err := s.publisher.Publish(ctx, message); err != nil {
		return task.Message{}, err
	}

	return message, nil
}

func (s AssistantService) ReceiveResult(ctx context.Context, result task.Result) error {
	return s.repo.CompleteAssistantRequest(ctx, result)
}

// SummarizeDocument is the permission-checked entry point handlers must use to
// trigger a document summary — it is also the entry point of the read leak
// this queues a request that ListRequests/GetRequest can later expose, so
// queuing it for a document the actor cannot view would let them read the
// result via those endpoints regardless of how well those are locked down.
func (s AssistantService) SummarizeDocument(
	ctx context.Context,
	documentID string,
	payload map[string]any,
	actorID string,
	actorRole string,
) (task.Message, error) {
	if err := s.permissions.EnsureViewDocument(ctx, actorID, actorRole, documentID); err != nil {
		return task.Message{}, err
	}
	payload = assistantPayload(payload, "version_id", "skill_name")
	if versionID := strings.TrimSpace(stringValue(payload["version_id"])); versionID != "" {
		if s.versions == nil {
			return task.Message{}, ErrValidation
		}
		version, err := s.versions.GetVersion(ctx, versionID)
		if err != nil {
			return task.Message{}, err
		}
		if version.DocumentID != documentID {
			return task.Message{}, fmt.Errorf("%w: version does not belong to document", ErrValidation)
		}
		payload["version_id"] = versionID
	}
	return s.QueueTask(ctx, task.TaskTypeDocumentSummarize, "document", documentID, payload, actorID)
}

// SummarizeHandover mirrors SummarizeDocument for handovers.
func (s AssistantService) SummarizeHandover(
	ctx context.Context,
	handoverID string,
	payload map[string]any,
	actorID string,
	actorRole string,
) (task.Message, error) {
	if err := s.permissions.EnsureViewHandover(ctx, actorID, actorRole, handoverID); err != nil {
		return task.Message{}, err
	}
	return s.QueueTask(ctx, task.TaskTypeHandoverSummarize, "handover", handoverID, assistantPayload(payload, "skill_name"), actorID)
}

func (s AssistantService) GetRequest(
	ctx context.Context,
	requestID string,
	actorID string,
	actorRole string,
) (*query.AssistantRequestItem, error) {
	if err := s.permissions.EnsureViewAssistantRequest(ctx, actorID, actorRole, requestID); err != nil {
		return nil, err
	}
	return s.repo.GetAssistantRequest(ctx, requestID)
}

// ListRequests relies on filter.Scope (set by the handler) to narrow the
// result set in SQL — there is no single resource id here to run a
// per-request Ensure* check against, the same reasoning as ListHandovers/
// ListAuditEvents.
func (s AssistantService) ListRequests(
	ctx context.Context,
	filter query.AssistantRequestFilter,
) ([]query.AssistantRequestItem, int, error) {
	return s.repo.ListAssistantRequests(ctx, filter)
}

func (s AssistantService) CreateConversation(
	ctx context.Context,
	scopePayload map[string]any,
	title string,
	actorID string,
	actorRole string,
) (*query.AssistantConversationItem, error) {
	scope, err := s.normalizeScope(ctx, scopePayload)
	if err != nil {
		return nil, err
	}
	if err := s.ensureViewScope(ctx, actorID, actorRole, scope); err != nil {
		return nil, err
	}
	return s.repo.CreateConversation(ctx, scope.ScopeType, scope.ScopeID, scope.SourceScope, title, actorID)
}

func (s AssistantService) ListConversations(
	ctx context.Context,
	filter query.AssistantConversationFilter,
) ([]query.AssistantConversationItem, error) {
	return s.repo.ListConversations(ctx, filter)
}

func (s AssistantService) ArchiveConversation(
	ctx context.Context,
	conversationID string,
	archive bool,
	actorID string,
	actorRole string,
) error {
	if err := s.permissions.EnsureViewConversation(ctx, actorID, actorRole, conversationID); err != nil {
		return err
	}
	return s.repo.ArchiveConversation(ctx, conversationID, archive)
}

func (s AssistantService) GetConversation(
	ctx context.Context,
	conversationID string,
	actorID string,
	actorRole string,
) (*query.AssistantConversationItem, error) {
	if err := s.permissions.EnsureViewConversation(ctx, actorID, actorRole, conversationID); err != nil {
		return nil, err
	}
	return s.repo.GetConversation(ctx, conversationID)
}

func (s AssistantService) ListConversationMessages(
	ctx context.Context,
	conversationID string,
	actorID string,
	actorRole string,
) ([]query.AssistantConversationMessageItem, error) {
	if err := s.permissions.EnsureViewConversation(ctx, actorID, actorRole, conversationID); err != nil {
		return nil, err
	}
	return s.repo.ListConversationMessages(ctx, conversationID)
}

func (s AssistantService) GetDocumentVersionExtractedText(
	ctx context.Context,
	documentID, versionID string,
) (string, error) {
	return s.repo.GetDocumentVersionExtractedText(ctx, documentID, versionID)
}

// ListSuggestions relies on filter.Scope the same way ListRequests does.
func (s AssistantService) ListSuggestions(
	ctx context.Context,
	filter query.AssistantSuggestionFilter,
) ([]query.AssistantSuggestionItem, error) {
	return s.repo.ListSuggestions(ctx, filter)
}

func (s AssistantService) ConfirmSuggestion(
	ctx context.Context,
	suggestionID string,
	actorID string,
	actorRole string,
	note string,
) (map[string]any, error) {
	if err := s.permissions.EnsureViewAssistantSuggestion(ctx, actorID, actorRole, suggestionID); err != nil {
		return nil, err
	}
	return s.repo.ConfirmSuggestion(ctx, suggestionID, actorID, note)
}

func (s AssistantService) DismissSuggestion(
	ctx context.Context,
	suggestionID string,
	actorID string,
	actorRole string,
	reason string,
) (map[string]any, error) {
	if err := s.permissions.EnsureViewAssistantSuggestion(ctx, actorID, actorRole, suggestionID); err != nil {
		return nil, err
	}
	return s.repo.DismissSuggestion(ctx, suggestionID, actorID, reason)
}

// Only server-built scope and memory may influence resource reads.
func assistantPayload(input map[string]any, keys ...string) map[string]any {
	result := map[string]any{}
	for _, key := range keys {
		if value, ok := input[key]; ok {
			result[key] = value
		}
	}
	return result
}
