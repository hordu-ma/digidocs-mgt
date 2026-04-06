package memory

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/domain/task"
	"digidocs-mgt/backend-go/internal/repository"
	"digidocs-mgt/backend-go/internal/service"
)

type assistantRequestRecord struct {
	RequestID   string
	RequestType string
	RelatedType string
	RelatedID   string
	Payload     map[string]any
	Status      string
	Error       string
	Output      map[string]any
	CreatedAt   string
	CompletedAt string
}

type AssistantRepository struct {
	mu          sync.Mutex
	requests    map[string]assistantRequestRecord
	suggestions map[string]query.AssistantSuggestionItem
}

func NewAssistantRepository() repository.AssistantRepository {
	return &AssistantRepository{
		requests:    map[string]assistantRequestRecord{},
		suggestions: map[string]query.AssistantSuggestionItem{},
	}
}

func (r *AssistantRepository) CreateAssistantRequest(
	_ context.Context,
	message task.Message,
	_ string,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	payload := clonePayload(message.Payload)
	r.requests[message.RequestID] = assistantRequestRecord{
		RequestID:   message.RequestID,
		RequestType: string(message.TaskType),
		RelatedType: message.RelatedType,
		RelatedID:   message.RelatedID,
		Payload:     payload,
		Status:      "pending",
		Output:      map[string]any{},
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	return nil
}

func (r *AssistantRepository) CompleteAssistantRequest(
	_ context.Context,
	result task.Result,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	req, ok := r.requests[result.RequestID]
	if !ok {
		return service.ErrNotFound
	}
	req.Status = result.Status
	req.Error = result.ErrorMessage
	req.Output = clonePayload(result.Output)
	req.CompletedAt = time.Now().UTC().Format(time.RFC3339)
	r.requests[result.RequestID] = req

	for id, item := range r.suggestions {
		if item.RequestID == result.RequestID {
			delete(r.suggestions, id)
		}
	}

	for _, item := range buildSuggestionItems(req, result) {
		r.suggestions[item.ID] = item
	}
	return nil
}

func (r *AssistantRepository) GetAssistantRequest(
	_ context.Context,
	requestID string,
) (*query.AssistantRequestItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	req, ok := r.requests[requestID]
	if !ok {
		return nil, service.ErrNotFound
	}
	item := buildAssistantRequestItem(req)
	return &item, nil
}

func (r *AssistantRepository) ListAssistantRequests(
	_ context.Context,
	filter query.AssistantRequestFilter,
) ([]query.AssistantRequestItem, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	items := make([]query.AssistantRequestItem, 0)
	keyword := strings.ToLower(strings.TrimSpace(filter.Keyword))
	for _, req := range r.requests {
		if filter.RequestType != "" && req.RequestType != filter.RequestType {
			continue
		}
		if filter.RelatedType != "" && req.RelatedType != filter.RelatedType {
			continue
		}
		if filter.RelatedID != "" && req.RelatedID != filter.RelatedID {
			continue
		}
		if filter.Status != "" && req.Status != filter.Status {
			continue
		}
		item := buildAssistantRequestItem(req)
		if keyword != "" && !strings.Contains(strings.ToLower(item.Question), keyword) {
			continue
		}
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt > items[j].CreatedAt
	})

	total := len(items)
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	if start >= total {
		return []query.AssistantRequestItem{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return items[start:end], total, nil
}

func (r *AssistantRepository) GetLatestDocumentExtractedText(
	_ context.Context,
	documentID string,
) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	latestCreatedAt := ""
	extractedText := ""
	for _, req := range r.requests {
		if req.RelatedID != documentID {
			continue
		}
		if req.RequestType != string(task.TaskTypeDocumentExtractText) &&
			req.RequestType != string(task.TaskTypeDocumentSummarize) {
			continue
		}
		if req.Status != "completed" {
			continue
		}
		text := stringValue(req.Output["extracted_text"])
		if text == "" {
			continue
		}
		if latestCreatedAt == "" || req.CompletedAt > latestCreatedAt {
			latestCreatedAt = req.CompletedAt
			extractedText = text
		}
	}
	return extractedText, nil
}

func (r *AssistantRepository) ListSuggestions(
	_ context.Context,
	filter query.AssistantSuggestionFilter,
) ([]query.AssistantSuggestionItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

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
		if filter.SuggestionType != "" && item.SuggestionType != filter.SuggestionType {
			continue
		}
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].GeneratedAt > items[j].GeneratedAt
	})
	return items, nil
}

func (r *AssistantRepository) ConfirmSuggestion(
	_ context.Context,
	suggestionID string,
	actorID string,
	note string,
) (map[string]any, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, ok := r.suggestions[suggestionID]
	if !ok {
		return nil, service.ErrNotFound
	}
	item.Status = "confirmed"
	r.suggestions[suggestionID] = item
	return map[string]any{
		"id":           suggestionID,
		"status":       "confirmed",
		"confirmed_by": actorID,
		"note":         note,
	}, nil
}

func (r *AssistantRepository) DismissSuggestion(
	_ context.Context,
	suggestionID string,
	actorID string,
	reason string,
) (map[string]any, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, ok := r.suggestions[suggestionID]
	if !ok {
		return nil, service.ErrNotFound
	}
	item.Status = "dismissed"
	r.suggestions[suggestionID] = item
	return map[string]any{
		"id":           suggestionID,
		"status":       "dismissed",
		"dismissed_by": actorID,
		"reason":       reason,
	}, nil
}

func buildSuggestionItems(
	req assistantRequestRecord,
	result task.Result,
) []query.AssistantSuggestionItem {
	if result.Status != "completed" {
		return nil
	}

	items := make([]query.AssistantSuggestionItem, 0)
	now := time.Now().UTC().Format(time.RFC3339)
	sourceScope := stringifyScope(req.Payload)

	if text := stringValue(result.Output["summary_text"]); text != "" {
		items = append(items, query.AssistantSuggestionItem{
			ID:             req.RequestID + "-summary",
			RelatedType:    req.RelatedType,
			RelatedID:      req.RelatedID,
			SuggestionType: suggestionTypeForRequest(req.RequestType),
			Status:         "pending",
			Title:          defaultSuggestionTitle(req.RequestType),
			Content:        text,
			SourceScope:    sourceScope,
			RequestID:      req.RequestID,
			GeneratedAt:    now,
		})
	}

	if rawSuggestions, ok := result.Output["suggestions"].([]any); ok {
		for i, raw := range rawSuggestions {
			m, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			content := stringValue(m["content"])
			if content == "" {
				continue
			}
			item := query.AssistantSuggestionItem{
				ID:             req.RequestID + "-suggestion-" + strconv.Itoa(i+1),
				RelatedType:    req.RelatedType,
				RelatedID:      req.RelatedID,
				SuggestionType: fallbackString(stringValue(m["suggestion_type"]), suggestionTypeForRequest(req.RequestType)),
				Status:         "pending",
				Title:          stringValue(m["title"]),
				Content:        content,
				SourceScope:    fallbackString(stringValue(m["source_scope"]), sourceScope),
				RequestID:      req.RequestID,
				GeneratedAt:    now,
			}
			if confidence, ok := floatValue(m["confidence"]); ok {
				item.Confidence = &confidence
			}
			items = append(items, item)
		}
	}

	return items
}

func suggestionTypeForRequest(requestType string) string {
	switch requestType {
	case string(task.TaskTypeDocumentSummarize):
		return "document_summary"
	case string(task.TaskTypeHandoverSummarize):
		return "handover_summary"
	case string(task.TaskTypeAssistantGenerateSuggestion):
		return "structure_recommendation"
	default:
		return "document_summary"
	}
}

func defaultSuggestionTitle(requestType string) string {
	switch requestType {
	case string(task.TaskTypeDocumentSummarize):
		return "文档摘要"
	case string(task.TaskTypeHandoverSummarize):
		return "交接摘要"
	default:
		return "AI 建议"
	}
}

func stringifyScope(payload map[string]any) string {
	scopePayload := map[string]any{}
	if scope, ok := payload["scope"].(map[string]any); ok {
		scopePayload = clonePayload(scope)
	} else {
		if projectID := stringValue(payload["project_id"]); projectID != "" {
			scopePayload["project_id"] = projectID
		}
		if documentID := stringValue(payload["document_id"]); documentID != "" {
			scopePayload["document_id"] = documentID
		}
	}
	if len(scopePayload) == 0 {
		return ""
	}
	raw, err := json.Marshal(scopePayload)
	if err != nil {
		return ""
	}
	return string(raw)
}

func clonePayload(payload map[string]any) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(payload))
	for k, v := range payload {
		cloned[k] = v
	}
	return cloned
}

func buildAssistantRequestItem(req assistantRequestRecord) query.AssistantRequestItem {
	item := query.AssistantRequestItem{
		ID:           req.RequestID,
		RequestType:  req.RequestType,
		RelatedType:  req.RelatedType,
		RelatedID:    req.RelatedID,
		Status:       req.Status,
		Question:     stringValue(req.Payload["question"]),
		SourceScope:  extractScope(req.Payload, req.Output),
		ErrorMessage: req.Error,
		Output:       clonePayload(req.Output),
		CreatedAt:    req.CreatedAt,
		CompletedAt:  req.CompletedAt,
	}
	item.Model = stringValue(req.Output["model"])
	item.UpstreamRequestID = stringValue(req.Output["request_id"])
	if usage, ok := req.Output["usage"].(map[string]any); ok {
		item.Usage = clonePayload(usage)
	}
	if item.CreatedAt != "" && item.CompletedAt != "" {
		createdAt, createdErr := time.Parse(time.RFC3339, item.CreatedAt)
		completedAt, completedErr := time.Parse(time.RFC3339, item.CompletedAt)
		if createdErr == nil && completedErr == nil {
			item.ProcessingDurationMs = completedAt.Sub(createdAt).Milliseconds()
		}
	}
	return item
}

func extractScope(payload map[string]any, output map[string]any) map[string]any {
	if scope, ok := output["source_scope"].(map[string]any); ok {
		return clonePayload(scope)
	}
	if scope, ok := payload["scope"].(map[string]any); ok {
		return clonePayload(scope)
	}
	result := map[string]any{}
	if projectID := stringValue(payload["project_id"]); projectID != "" {
		result["project_id"] = projectID
	}
	if documentID := stringValue(payload["document_id"]); documentID != "" {
		result["document_id"] = documentID
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func floatValue(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	default:
		return 0, false
	}
}

func fallbackString(value string, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func stringValue(value any) string {
	raw, _ := value.(string)
	return raw
}
