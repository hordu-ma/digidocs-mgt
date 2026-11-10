package service

import (
	"context"
	"errors"
	"testing"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/domain/task"
)

func (r *fakeAssistantRepo) GetSuggestion(_ context.Context, id string) (*query.AssistantSuggestionItem, error) {
	item, ok := r.suggestions[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &item, nil
}

func (r *fakeAssistantRepo) UpdateRequestProgress(_ context.Context, id string, partial string) error {
	item, ok := r.requests[id]
	if !ok {
		return nil
	}
	if item.Output == nil {
		item.Output = map[string]any{}
	}
	item.Output["partial_answer"] = partial
	r.requests[id] = item
	return nil
}

func (r *fakeAssistantRepo) GetRequestCreator(_ context.Context, id string) (string, error) {
	if _, ok := r.requests[id]; !ok {
		return "", ErrNotFound
	}
	return "uploader", nil
}

type recordingFlowExecutor struct {
	got []command.FlowActionInput
	err error
}

func (e *recordingFlowExecutor) ApplyAction(_ context.Context, in command.FlowActionInput) (map[string]any, error) {
	e.got = append(e.got, in)
	return map[string]any{}, e.err
}

func TestNormalizeSuggestedActionWhitelist(t *testing.T) {
	for in, want := range map[string]string{"finalize": "finalize", " Archive ": "archive", "transfer": "", "delete": "", "": ""} {
		if got := NormalizeSuggestedAction(in); got != want {
			t.Fatalf("%q -> %q, want %q", in, got, want)
		}
	}
}

func TestConfirmAndExecuteRunsFlowAsUserThenConfirms(t *testing.T) {
	repo := newFakeAssistantRepo()
	repo.suggestions["s1"] = query.AssistantSuggestionItem{ID: "s1", RelatedType: "document", RelatedID: "d1", Status: "pending", Title: "建议定稿", SuggestedAction: "finalize"}
	flows := &recordingFlowExecutor{}
	svc := NewAssistantService(nil, repo, nil, nil).WithFlowExecutor(flows)
	out, err := svc.ConfirmAndExecuteSuggestion(context.Background(), "s1", "u1", "member", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(flows.got) != 1 || flows.got[0].Action != "finalize" || flows.got[0].ActorID != "u1" || flows.got[0].DocumentID != "d1" {
		t.Fatalf("flow call: %#v", flows.got)
	}
	if out["executed_action"] != "finalize" || repo.suggestions["s1"].Status != "confirmed" {
		t.Fatalf("result %#v status %s", out, repo.suggestions["s1"].Status)
	}
}

func TestConfirmAndExecuteKeepsPendingWhenFlowFails(t *testing.T) {
	repo := newFakeAssistantRepo()
	repo.suggestions["s1"] = query.AssistantSuggestionItem{ID: "s1", RelatedType: "document", RelatedID: "d1", Status: "pending", SuggestedAction: "archive"}
	svc := NewAssistantService(nil, repo, nil, nil).WithFlowExecutor(&recordingFlowExecutor{err: ErrInvalidTransition})
	if _, err := svc.ConfirmAndExecuteSuggestion(context.Background(), "s1", "u1", "member", ""); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("want ErrInvalidTransition, got %v", err)
	}
	if repo.suggestions["s1"].Status != "pending" {
		t.Fatal("a failed action must not confirm the suggestion")
	}
}

func TestConfirmAndExecuteRejectsNonActionable(t *testing.T) {
	repo := newFakeAssistantRepo()
	repo.suggestions["plain"] = query.AssistantSuggestionItem{ID: "plain", RelatedType: "document", Status: "pending"}
	repo.suggestions["done"] = query.AssistantSuggestionItem{ID: "done", RelatedType: "document", Status: "confirmed", SuggestedAction: "finalize"}
	flows := &recordingFlowExecutor{}
	svc := NewAssistantService(nil, repo, nil, nil).WithFlowExecutor(flows)
	for _, id := range []string{"plain", "done"} {
		if _, err := svc.ConfirmAndExecuteSuggestion(context.Background(), id, "u", "member", ""); !errors.Is(err, ErrValidation) {
			t.Fatalf("%s: want ErrValidation, got %v", id, err)
		}
	}
	if len(flows.got) != 0 {
		t.Fatal("no flow action may run")
	}
}

func TestAutoSummarizeChainsAfterUploadExtraction(t *testing.T) {
	repo := newFakeAssistantRepo()
	pub := &capturePublisher{}
	repo.requests["ext"] = query.AssistantRequestItem{ID: "ext", RequestType: string(task.TaskTypeDocumentExtractText), RelatedType: "document", RelatedID: "d1"}
	svc := NewAssistantService(pub, repo, nil, nil).WithAutoSummarize(true)
	if err := svc.ReceiveResult(context.Background(), task.Result{RequestID: "ext", Status: "completed", Output: map[string]any{"extracted_text": "正文", "version_id": "v2"}}); err != nil {
		t.Fatal(err)
	}
	if len(pub.messages) != 1 || pub.messages[0].TaskType != task.TaskTypeDocumentSummarize || pub.messages[0].Payload["version_id"] != "v2" {
		t.Fatalf("expected one summarize task for v2, got %#v", pub.messages)
	}
	// Disabled, failed, or empty text: nothing is chained.
	for _, tc := range []struct {
		enabled bool
		result  task.Result
	}{
		{false, task.Result{RequestID: "ext", Status: "completed", Output: map[string]any{"extracted_text": "x"}}},
		{true, task.Result{RequestID: "ext", Status: "failed"}},
		{true, task.Result{RequestID: "ext", Status: "completed", Output: map[string]any{"extracted_text": "  "}}},
	} {
		pub.messages = nil
		_ = NewAssistantService(pub, repo, nil, nil).WithAutoSummarize(tc.enabled).ReceiveResult(context.Background(), tc.result)
		if len(pub.messages) != 0 {
			t.Fatalf("unexpected chained task for %#v", tc)
		}
	}
}

type capturePublisher struct{ messages []task.Message }

func (p *capturePublisher) Publish(_ context.Context, m task.Message) error {
	p.messages = append(p.messages, m)
	return nil
}
