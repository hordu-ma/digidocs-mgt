package postgres

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/domain/task"
)

type fakeScanner struct {
	values []any
	err    error
}

func (s fakeScanner) Scan(dest ...any) error {
	if s.err != nil {
		return s.err
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = s.values[i].(string)
		default:
			return errors.New("unsupported scan destination")
		}
	}
	return nil
}

func TestSuggestionHelpers(t *testing.T) {
	if got := suggestionTypeForRequest(string(task.TaskTypeDocumentSummarize)); got != "document_summary" {
		t.Fatalf("document summarize type = %q", got)
	}
	if got := suggestionTypeForRequest(string(task.TaskTypeHandoverSummarize)); got != "handover_summary" {
		t.Fatalf("handover summarize type = %q", got)
	}
	if got := suggestionTypeForRequest(string(task.TaskTypeAssistantGenerateSuggestion)); got != "structure_recommendation" {
		t.Fatalf("assistant suggestion type = %q", got)
	}
	if got := suggestionTypeForRequest("unknown"); got != "document_summary" {
		t.Fatalf("unknown suggestion type = %q", got)
	}
	if got := normalizeSuggestionType("risk_alert", "document_summary"); got != "risk_alert" {
		t.Fatalf("valid normalize = %q", got)
	}
	if got := normalizeSuggestionType("bad", "document_summary"); got != "document_summary" {
		t.Fatalf("invalid normalize = %q", got)
	}
	if got := defaultSuggestionTitle(string(task.TaskTypeDocumentSummarize)); got != "文档摘要" {
		t.Fatalf("document title = %q", got)
	}
	if got := defaultSuggestionTitle(string(task.TaskTypeHandoverSummarize)); got != "交接摘要" {
		t.Fatalf("handover title = %q", got)
	}
	if got := defaultSuggestionTitle("other"); got != "AI 建议" {
		t.Fatalf("default title = %q", got)
	}
}

func TestScopeAndPayloadHelpers(t *testing.T) {
	payload := map[string]any{
		"scope":          map[string]any{"project_id": "p-1", "document_id": "d-1"},
		"memory_sources": []any{map[string]any{"type": "conversation"}, "skip"},
	}
	scope := extractScopeFromPayload(payload)
	if !reflect.DeepEqual(scope, map[string]any{"project_id": "p-1", "document_id": "d-1"}) {
		t.Fatalf("scope = %#v", scope)
	}
	scope["project_id"] = "changed"
	if payload["scope"].(map[string]any)["project_id"] != "p-1" {
		t.Fatal("extractScopeFromPayload returned aliased scope")
	}
	if got := stringifyScope(payload); !strings.Contains(got, `"project_id":"p-1"`) || !strings.Contains(got, `"document_id":"d-1"`) {
		t.Fatalf("stringifyScope = %q, want encoded scope", got)
	}
	if got := extractMemorySources(payload); len(got) != 1 || got[0]["type"] != "conversation" {
		t.Fatalf("memory sources = %#v", got)
	}

	legacyPayload := map[string]any{"project_id": "p-2", "document_id": "d-2"}
	if got := stringifyScope(legacyPayload); !strings.Contains(got, `"project_id":"p-2"`) {
		t.Fatalf("legacy stringifyScope = %q", got)
	}
	outputScope := extractScope(legacyPayload, map[string]any{"source_scope": map[string]any{"project_id": "p-out"}})
	if outputScope["project_id"] != "p-out" {
		t.Fatalf("extractScope output override = %#v", outputScope)
	}
	if got := extractScopeFromPayload(map[string]any{}); got != nil {
		t.Fatalf("empty scope = %#v, want nil", got)
	}
}

func TestPrimitiveHelpers(t *testing.T) {
	if got := clonePayload(nil); len(got) != 0 {
		t.Fatalf("clone nil = %#v, want empty map", got)
	}
	source := map[string]any{"a": "b"}
	cloned := clonePayload(source)
	cloned["a"] = "changed"
	if source["a"] != "b" {
		t.Fatal("clonePayload returned aliased map")
	}
	if got := mustJSON(map[string]any{"a": "b"}); got != `{"a":"b"}` {
		t.Fatalf("mustJSON = %q", got)
	}
	if got := mustJSON(make(chan int)); got != "{}" {
		t.Fatalf("mustJSON unsupported = %q, want {}", got)
	}
	for _, value := range []any{float64(0.8), float32(0.7), 3} {
		if _, ok := floatValue(value); !ok {
			t.Fatalf("floatValue(%T) ok=false, want true", value)
		}
	}
	if _, ok := floatValue("0.8"); ok {
		t.Fatal("floatValue string ok=true, want false")
	}
	if got := fallbackString("", "fallback"); got != "fallback" {
		t.Fatalf("fallbackString empty = %q", got)
	}
	if got := stringValue(123); got != "" {
		t.Fatalf("stringValue non-string = %q, want empty", got)
	}
}

func TestBuildSuggestionItems(t *testing.T) {
	req := assistantRequestRecord{
		RequestID:   "req-1",
		RequestType: string(task.TaskTypeAssistantGenerateSuggestion),
		RelatedType: "document",
		RelatedID:   "doc-1",
		Payload:     map[string]any{"scope": map[string]any{"document_id": "doc-1"}},
	}
	result := task.Result{Output: map[string]any{
		"suggestions": []any{
			map[string]any{"content": "归档建议", "suggestion_type": "archive_recommendation", "title": "建议", "confidence": 0.9},
			map[string]any{"content": ""},
			"skip",
		},
	}}
	result.Status = "completed"
	items := buildSuggestionItems(req, result)
	if len(items) != 1 {
		t.Fatalf("items = %#v, want one suggestion", items)
	}
	item := items[0]
	if item.RelatedID != "doc-1" || item.SuggestionType != "archive_recommendation" || item.Title != "建议" || item.Confidence == nil || *item.Confidence != 0.9 {
		t.Fatalf("item = %#v, want mapped suggestion", item)
	}
}

func TestScanAssistantRequestItem(t *testing.T) {
	created := "2026-05-01T00:00:00Z"
	completed := "2026-05-01T00:00:02Z"
	payload := `{"question":"如何归档？","scope":{"project_id":"p-1"},"memory_sources":[{"type":"history"}]}`
	output := `{"source_scope":{"document_id":"d-1"},"skill_name":"skill-a","skill_version":"v1","model":"model-a","request_id":"up-1","usage":{"total_tokens":12}}`
	item, err := scanAssistantRequestItem(fakeScanner{values: []any{
		"req-1",
		string(task.TaskTypeAssistantAsk),
		"document",
		"d-1",
		"conv-1",
		"completed",
		"",
		payload,
		output,
		created,
		completed,
	}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Question != "如何归档？" || item.SkillName != "skill-a" || item.ProcessingDurationMs != 2000 {
		t.Fatalf("item = %#v, want parsed request", item)
	}
	if item.SourceScope["document_id"] != "d-1" || len(item.MemorySources) != 1 || item.Usage["total_tokens"] != float64(12) {
		t.Fatalf("item scope/memory/usage = %#v %#v %#v", item.SourceScope, item.MemorySources, item.Usage)
	}
}

func TestScanConversationItems(t *testing.T) {
	conversation, err := scanAssistantConversationItem(fakeScanner{values: []any{
		"conv-1",
		"project",
		"p-1",
		`{"project_id":"p-1"}`,
		"课题 A",
		"标题",
		"u-1",
		"2026-05-01T00:00:00Z",
		"2026-05-01T00:01:00Z",
		"",
	}})
	if err != nil {
		t.Fatalf("conversation unexpected error: %v", err)
	}
	if conversation.SourceScope["project_id"] != "p-1" || conversation.ScopeDisplayName != "课题 A" {
		t.Fatalf("conversation = %#v", conversation)
	}

	message, err := scanAssistantConversationMessageItem(fakeScanner{values: []any{
		"msg-1",
		"conv-1",
		"assistant",
		"回答",
		"req-1",
		`{"model":"m"}`,
		"u-1",
		"2026-05-01T00:00:00Z",
	}})
	if err != nil {
		t.Fatalf("message unexpected error: %v", err)
	}
	if message.Metadata["model"] != "m" || message.Role != "assistant" {
		t.Fatalf("message = %#v", message)
	}
}

func TestParseRFC3339(t *testing.T) {
	got := parseRFC3339("2026-05-01T00:00:00Z")
	if got.Format(time.RFC3339) != "2026-05-01T00:00:00Z" {
		t.Fatalf("parseRFC3339 valid = %s", got.Format(time.RFC3339))
	}
	before := time.Now().Add(-time.Second)
	got = parseRFC3339("bad")
	after := time.Now().Add(time.Second)
	if got.Before(before) || got.After(after) {
		t.Fatalf("parseRFC3339 fallback = %s, want near now", got.Format(time.RFC3339))
	}
}

var _ = query.AssistantRequestItem{}
