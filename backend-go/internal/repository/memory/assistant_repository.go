package memory

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	RequestID      string
	RequestType    string
	RelatedType    string
	RelatedID      string
	ConversationID string
	Payload        map[string]any
	Status         string
	Error          string
	Output         map[string]any
	CreatedBy      string
	CreatedAt      string
	CompletedAt    string
}

type assistantConversationRecord struct {
	ID            string
	ScopeType     string
	ScopeID       string
	SourceScope   map[string]any
	Title         string
	CreatedBy     string
	CreatedAt     string
	LastMessageAt string
	ArchivedAt    string
}

type assistantConversationMessageRecord struct {
	ID             string
	ConversationID string
	Role           string
	Content        string
	RequestID      string
	Metadata       map[string]any
	CreatedBy      string
	CreatedAt      string
}

type AssistantRepository struct {
	mu                   sync.Mutex
	requests             map[string]assistantRequestRecord
	suggestions          map[string]query.AssistantSuggestionItem
	conversations        map[string]assistantConversationRecord
	conversationMessages map[string][]assistantConversationMessageRecord
}

func NewAssistantRepository() repository.AssistantRepository {
	return &AssistantRepository{
		requests:             map[string]assistantRequestRecord{},
		suggestions:          map[string]query.AssistantSuggestionItem{},
		conversations:        map[string]assistantConversationRecord{},
		conversationMessages: map[string][]assistantConversationMessageRecord{},
	}
}

func (r *AssistantRepository) CreateConversation(
	_ context.Context,
	scopeType string,
	scopeID string,
	sourceScope map[string]any,
	title string,
	actorID string,
) (*query.AssistantConversationItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	record := assistantConversationRecord{
		ID:            newID(),
		ScopeType:     scopeType,
		ScopeID:       scopeID,
		SourceScope:   clonePayload(sourceScope),
		Title:         title,
		CreatedBy:     actorID,
		CreatedAt:     now,
		LastMessageAt: now,
	}
	r.conversations[record.ID] = record
	item := buildConversationItem(record)
	return &item, nil
}

func (r *AssistantRepository) GetConversation(
	_ context.Context,
	conversationID string,
) (*query.AssistantConversationItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, ok := r.conversations[conversationID]
	if !ok {
		return nil, service.ErrNotFound
	}
	item := buildConversationItem(record)
	return &item, nil
}

func (r *AssistantRepository) ListConversations(
	_ context.Context,
	filter query.AssistantConversationFilter,
) ([]query.AssistantConversationItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	items := make([]query.AssistantConversationItem, 0)
	for _, record := range r.conversations {
		if !filter.IncludeArchived && record.ArchivedAt != "" {
			continue
		}
		if filter.ScopeType != "" && record.ScopeType != filter.ScopeType {
			continue
		}
		if filter.ScopeID != "" && record.ScopeID != filter.ScopeID {
			continue
		}
		if filter.CreatedBy != "" && record.CreatedBy != filter.CreatedBy {
			continue
		}
		items = append(items, buildConversationItem(record))
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].LastMessageAt > items[j].LastMessageAt
	})
	return items, nil
}

func (r *AssistantRepository) ArchiveConversation(
	_ context.Context,
	conversationID string,
	archive bool,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, ok := r.conversations[conversationID]
	if !ok {
		return service.ErrNotFound
	}
	if archive {
		record.ArchivedAt = time.Now().UTC().Format(time.RFC3339)
	} else {
		record.ArchivedAt = ""
	}
	r.conversations[conversationID] = record
	return nil
}

func (r *AssistantRepository) ListConversationMessages(
	_ context.Context,
	conversationID string,
) ([]query.AssistantConversationMessageItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.conversations[conversationID]; !ok {
		return nil, service.ErrNotFound
	}
	rawItems := r.conversationMessages[conversationID]
	items := make([]query.AssistantConversationMessageItem, 0, len(rawItems))
	for _, item := range rawItems {
		items = append(items, buildConversationMessageItem(item))
	}
	return items, nil
}

func (r *AssistantRepository) CreateAssistantRequest(
	_ context.Context,
	message task.Message,
	actorID string,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	payload := clonePayload(message.Payload)
	now := time.Now().UTC().Format(time.RFC3339)
	record := assistantRequestRecord{
		RequestID:      message.RequestID,
		RequestType:    string(message.TaskType),
		RelatedType:    message.RelatedType,
		RelatedID:      message.RelatedID,
		ConversationID: stringValue(payload["conversation_id"]),
		Payload:        payload,
		Status:         "pending",
		Output:         map[string]any{},
		CreatedBy:      actorID,
		CreatedAt:      now,
	}
	r.requests[message.RequestID] = record

	if record.ConversationID != "" {
		r.appendConversationMessageLocked(assistantConversationMessageRecord{
			ID:             newID(),
			ConversationID: record.ConversationID,
			Role:           "user",
			Content:        stringValue(payload["question"]),
			RequestID:      message.RequestID,
			Metadata: map[string]any{
				"source_scope":   extractScope(payload, nil),
				"memory_sources": extractMemorySources(payload),
			},
			CreatedBy: actorID,
			CreatedAt: now,
		})
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

	if req.ConversationID != "" {
		content := stringValue(result.Output["answer"])
		if content == "" && result.Status == "failed" {
			content = result.ErrorMessage
		}
		if content != "" {
			r.appendConversationMessageLocked(assistantConversationMessageRecord{
				ID:             newID(),
				ConversationID: req.ConversationID,
				Role:           "assistant",
				Content:        content,
				RequestID:      result.RequestID,
				Metadata: map[string]any{
					"status":                 result.Status,
					"model":                  stringValue(result.Output["model"]),
					"skill_name":             stringValue(result.Output["skill_name"]),
					"skill_version":          stringValue(result.Output["skill_version"]),
					"upstream_request_id":    stringValue(result.Output["request_id"]),
					"source_scope":           extractScope(req.Payload, req.Output),
					"memory_sources":         extractMemorySources(req.Payload),
					"processing_duration_ms": processingDurationMs(req.CreatedAt, req.CompletedAt),
				},
				CreatedAt: req.CompletedAt,
			})
		}
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
		if filter.ConversationID != "" && req.ConversationID != filter.ConversationID {
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

func (r *AssistantRepository) appendConversationMessageLocked(item assistantConversationMessageRecord) {
	if item.ConversationID == "" {
		return
	}
	r.conversationMessages[item.ConversationID] = append(r.conversationMessages[item.ConversationID], item)
	conversation, ok := r.conversations[item.ConversationID]
	if !ok {
		return
	}
	conversation.LastMessageAt = item.CreatedAt
	r.conversations[item.ConversationID] = conversation
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
	if outputScope, ok := result.Output["source_scope"].(map[string]any); ok {
		if raw, err := json.Marshal(outputScope); err == nil {
			sourceScope = string(raw)
		}
	}

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
				Title:          fallbackString(stringValue(m["title"]), defaultSuggestionTitle(req.RequestType)+"-"+strconv.Itoa(i+1)),
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

func buildAssistantRequestItem(req assistantRequestRecord) query.AssistantRequestItem {
	item := query.AssistantRequestItem{
		ID:             req.RequestID,
		RequestType:    req.RequestType,
		RelatedType:    req.RelatedType,
		RelatedID:      req.RelatedID,
		ConversationID: req.ConversationID,
		Status:         req.Status,
		Question:       stringValue(req.Payload["question"]),
		SourceScope:    extractScope(req.Payload, req.Output),
		MemorySources:  extractMemorySources(req.Payload),
		ErrorMessage:   req.Error,
		Output:         clonePayload(req.Output),
		CreatedAt:      req.CreatedAt,
		CompletedAt:    req.CompletedAt,
	}
	item.Model = stringValue(req.Output["model"])
	item.SkillName = stringValue(req.Output["skill_name"])
	item.SkillVersion = stringValue(req.Output["skill_version"])
	item.UpstreamRequestID = stringValue(req.Output["request_id"])
	if usage, ok := req.Output["usage"].(map[string]any); ok {
		item.Usage = clonePayload(usage)
	}
	item.ProcessingDurationMs = processingDurationMs(req.CreatedAt, req.CompletedAt)
	return item
}

func buildConversationItem(record assistantConversationRecord) query.AssistantConversationItem {
	return query.AssistantConversationItem{
		ID:            record.ID,
		ScopeType:     record.ScopeType,
		ScopeID:       record.ScopeID,
		SourceScope:   clonePayload(record.SourceScope),
		Title:         record.Title,
		CreatedBy:     record.CreatedBy,
		CreatedAt:     record.CreatedAt,
		LastMessageAt: record.LastMessageAt,
		ArchivedAt:    record.ArchivedAt,
	}
}

func buildConversationMessageItem(record assistantConversationMessageRecord) query.AssistantConversationMessageItem {
	return query.AssistantConversationMessageItem{
		ID:             record.ID,
		ConversationID: record.ConversationID,
		Role:           record.Role,
		Content:        record.Content,
		RequestID:      record.RequestID,
		Metadata:       clonePayload(record.Metadata),
		CreatedBy:      record.CreatedBy,
		CreatedAt:      record.CreatedAt,
	}
}

func extractScope(payload map[string]any, output map[string]any) map[string]any {
	if output != nil {
		if scope, ok := output["source_scope"].(map[string]any); ok {
			return clonePayload(scope)
		}
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

func extractMemorySources(payload map[string]any) []map[string]any {
	raw, ok := payload["memory_sources"].([]any)
	if !ok {
		return nil
	}
	items := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		items = append(items, clonePayload(m))
	}
	return items
}

func processingDurationMs(createdAt string, completedAt string) int64 {
	if createdAt == "" || completedAt == "" {
		return 0
	}
	created, createdErr := time.Parse(time.RFC3339, createdAt)
	completed, completedErr := time.Parse(time.RFC3339, completedAt)
	if createdErr != nil || completedErr != nil {
		return 0
	}
	return completed.Sub(created).Milliseconds()
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

func clonePayload(payload map[string]any) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(payload))
	for key, value := range payload {
		cloned[key] = value
	}
	return cloned
}

func stringValue(value any) string {
	raw, _ := value.(string)
	return raw
}

func newID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	hexed := hex.EncodeToString(buf)
	return fmt.Sprintf("%s-%s-%s-%s-%s", hexed[0:8], hexed[8:12], hexed[12:16], hexed[16:20], hexed[20:32])
}
