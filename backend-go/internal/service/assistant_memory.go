package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"digidocs-mgt/backend-go/internal/domain/query"
)

type assistantScope struct {
	ScopeType   string
	ScopeID     string
	SourceScope map[string]any
}

func (s AssistantService) normalizeScope(
	ctx context.Context,
	scopePayload map[string]any,
) (assistantScope, error) {
	scopePayload = clonePayload(scopePayload)
	documentID := strings.TrimSpace(stringValue(scopePayload["document_id"]))
	projectID := strings.TrimSpace(stringValue(scopePayload["project_id"]))

	if documentID == "" && projectID == "" {
		return assistantScope{}, fmt.Errorf("%w: scope is required", ErrValidation)
	}

	if documentID != "" && projectID == "" && s.documents != nil {
		if document, err := s.documents.GetDocument(ctx, documentID); err == nil {
			projectID = documentProjectID(document)
		}
	}

	sourceScope := map[string]any{}
	if projectID != "" {
		sourceScope["project_id"] = projectID
	}
	if documentID != "" {
		sourceScope["document_id"] = documentID
	}

	if documentID != "" {
		return assistantScope{
			ScopeType:   "document",
			ScopeID:     documentID,
			SourceScope: sourceScope,
		}, nil
	}

	return assistantScope{
		ScopeType:   "project",
		ScopeID:     projectID,
		SourceScope: sourceScope,
	}, nil
}

func (s AssistantService) buildMemorySnapshot(
	ctx context.Context,
	conversationID string,
	scope assistantScope,
) (map[string]any, []map[string]any, error) {
	memory := map[string]any{
		"scope_type": scope.ScopeType,
		"scope_id":   scope.ScopeID,
	}
	sources := make([]map[string]any, 0)

	messages, err := s.repo.ListConversationMessages(ctx, conversationID)
	if err != nil {
		return nil, nil, err
	}
	recentMessages := tailConversationMessages(messages, 6)
	if len(recentMessages) > 0 {
		memory["recent_messages"] = recentMessages
		sources = append(sources, map[string]any{
			"type":            "conversation_messages",
			"conversation_id": conversationID,
			"count":           len(recentMessages),
		})
	}

	confirmedSuggestions, err := s.collectConfirmedSuggestions(ctx, scope)
	if err != nil {
		return nil, nil, err
	}
	if len(confirmedSuggestions) > 0 {
		memory["confirmed_suggestions"] = confirmedSuggestions
		sources = append(sources, map[string]any{
			"type":     "confirmed_suggestions",
			"scope_id": scope.ScopeID,
			"count":    len(confirmedSuggestions),
		})
	}

	historicalAnswers, err := s.collectHistoricalAnswers(ctx, scope, conversationID)
	if err != nil {
		return nil, nil, err
	}
	if len(historicalAnswers) > 0 {
		memory["historical_answers"] = historicalAnswers
		sources = append(sources, map[string]any{
			"type":     "historical_answers",
			"scope_id": scope.ScopeID,
			"count":    len(historicalAnswers),
		})
	}

	return memory, sources, nil
}

func (s AssistantService) collectConfirmedSuggestions(
	ctx context.Context,
	scope assistantScope,
) ([]map[string]any, error) {
	items := make([]map[string]any, 0)
	seen := map[string]struct{}{}
	for _, target := range relatedTargetsForScope(scope) {
		suggestions, err := s.repo.ListSuggestions(ctx, query.AssistantSuggestionFilter{
			RelatedType: target.RelatedType,
			RelatedID:   target.RelatedID,
			Status:      "confirmed",
		})
		if err != nil {
			return nil, err
		}
		for _, item := range takeSuggestionTail(suggestions, 3) {
			if _, ok := seen[item.ID]; ok {
				continue
			}
			seen[item.ID] = struct{}{}
			items = append(items, map[string]any{
				"id":              item.ID,
				"related_type":    item.RelatedType,
				"related_id":      item.RelatedID,
				"suggestion_type": item.SuggestionType,
				"title":           item.Title,
				"content":         item.Content,
				"generated_at":    item.GeneratedAt,
			})
		}
	}
	return items, nil
}

func (s AssistantService) collectHistoricalAnswers(
	ctx context.Context,
	scope assistantScope,
	conversationID string,
) ([]map[string]any, error) {
	items := make([]map[string]any, 0)
	seen := map[string]struct{}{}
	for _, target := range relatedTargetsForScope(scope) {
		requests, _, err := s.repo.ListAssistantRequests(ctx, query.AssistantRequestFilter{
			RequestType: "assistant.ask",
			RelatedType: target.RelatedType,
			RelatedID:   target.RelatedID,
			Status:      "completed",
			Page:        1,
			PageSize:    5,
		})
		if err != nil {
			return nil, err
		}
		for _, item := range requests {
			if item.ConversationID == conversationID {
				continue
			}
			answer := strings.TrimSpace(stringValue(item.Output["answer"]))
			if answer == "" {
				continue
			}
			if _, ok := seen[item.ID]; ok {
				continue
			}
			seen[item.ID] = struct{}{}
			items = append(items, map[string]any{
				"request_id":      item.ID,
				"conversation_id": item.ConversationID,
				"related_type":    item.RelatedType,
				"related_id":      item.RelatedID,
				"question":        item.Question,
				"answer":          answer,
				"completed_at":    item.CompletedAt,
				"processing_ms":   item.ProcessingDurationMs,
			})
			if len(items) >= 3 {
				return items, nil
			}
		}
	}
	return items, nil
}

func relatedTargetsForScope(scope assistantScope) []query.AssistantSuggestionFilter {
	targets := []query.AssistantSuggestionFilter{{
		RelatedType: scope.ScopeType,
		RelatedID:   scope.ScopeID,
	}}
	projectID := strings.TrimSpace(stringValue(scope.SourceScope["project_id"]))
	if scope.ScopeType == "document" && projectID != "" {
		targets = append(targets, query.AssistantSuggestionFilter{
			RelatedType: "project",
			RelatedID:   projectID,
		})
	}
	return targets
}

func tailConversationMessages(items []query.AssistantConversationMessageItem, size int) []map[string]any {
	if size <= 0 || len(items) == 0 {
		return nil
	}
	if len(items) > size {
		items = items[len(items)-size:]
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, map[string]any{
			"role":       item.Role,
			"content":    item.Content,
			"request_id": item.RequestID,
			"created_at": item.CreatedAt,
		})
	}
	return result
}

func takeSuggestionTail(items []query.AssistantSuggestionItem, size int) []query.AssistantSuggestionItem {
	if len(items) <= size {
		return items
	}
	return items[:size]
}

func extractScope(payload map[string]any) map[string]any {
	if scope, ok := payload["scope"].(map[string]any); ok {
		return clonePayload(scope)
	}
	result := map[string]any{}
	if projectID := strings.TrimSpace(stringValue(payload["project_id"])); projectID != "" {
		result["project_id"] = projectID
	}
	if documentID := strings.TrimSpace(stringValue(payload["document_id"])); documentID != "" {
		result["document_id"] = documentID
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func mergeScopes(base map[string]any, overlay map[string]any) map[string]any {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}
	result := clonePayload(base)
	for key, value := range overlay {
		if strings.TrimSpace(stringValue(value)) == "" {
			continue
		}
		result[key] = value
	}
	return result
}

func buildConversationTitle(question string) string {
	trimmed := strings.TrimSpace(question)
	if trimmed == "" {
		return "AI 会话"
	}
	runes := []rune(trimmed)
	if len(runes) <= 24 {
		return trimmed
	}
	return string(runes[:24]) + "..."
}

func documentProjectID(document *query.DocumentDetail) string {
	if document == nil {
		return ""
	}
	// DocumentDetail 当前未暴露 project_id，保留此扩展位，避免后续 scope 组装散落修改。
	raw, err := json.Marshal(document)
	if err != nil {
		return ""
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(stringValue(payload["project_id"]))
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
