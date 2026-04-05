package service

import (
	"context"
	"errors"
	"testing"

	"digidocs-mgt/backend-go/internal/domain/command"
)

// --- mock: ActionWriter ---

type mockActionWriter struct {
	result map[string]any
	err    error
	called string // last method called
}

func (m *mockActionWriter) CreateFlowRecord(_ context.Context, _ command.FlowActionInput) (map[string]any, error) {
	m.called = "CreateFlowRecord"
	return m.result, m.err
}

func (m *mockActionWriter) CreateHandover(_ context.Context, _ command.HandoverCreateInput) (map[string]any, error) {
	m.called = "CreateHandover"
	return m.result, m.err
}

func (m *mockActionWriter) UpdateHandoverItems(_ context.Context, _ command.HandoverItemUpdateInput) (map[string]any, error) {
	m.called = "UpdateHandoverItems"
	return m.result, m.err
}

func (m *mockActionWriter) ApplyHandover(_ context.Context, _ command.HandoverActionInput) (map[string]any, error) {
	m.called = "ApplyHandover"
	return m.result, m.err
}

// --- tests ---

func TestApplyFlow_Delegates(t *testing.T) {
	want := map[string]any{"id": "flow-1"}
	w := &mockActionWriter{result: want}
	svc := NewActionService(w)

	got, err := svc.ApplyFlow(context.Background(), command.FlowActionInput{
		DocumentID: "doc-1", Action: "submit", ActorID: "u-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.called != "CreateFlowRecord" {
		t.Errorf("called = %q, want CreateFlowRecord", w.called)
	}
	if got["id"] != "flow-1" {
		t.Errorf("result id = %v, want flow-1", got["id"])
	}
}

func TestApplyFlow_PropagatesError(t *testing.T) {
	repoErr := errors.New("db error")
	w := &mockActionWriter{err: repoErr}
	svc := NewActionService(w)

	_, err := svc.ApplyFlow(context.Background(), command.FlowActionInput{})
	if !errors.Is(err, repoErr) {
		t.Errorf("err = %v, want %v", err, repoErr)
	}
}

func TestCreateHandover_Delegates(t *testing.T) {
	want := map[string]any{"id": "ho-1"}
	w := &mockActionWriter{result: want}
	svc := NewActionService(w)

	got, err := svc.CreateHandover(context.Background(), command.HandoverCreateInput{
		TargetUserID: "u-1", ReceiverUserID: "u-2", ProjectID: "p-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.called != "CreateHandover" {
		t.Errorf("called = %q, want CreateHandover", w.called)
	}
	if got["id"] != "ho-1" {
		t.Errorf("result id = %v, want ho-1", got["id"])
	}
}

func TestUpdateHandoverItems_Delegates(t *testing.T) {
	want := map[string]any{"updated": true}
	w := &mockActionWriter{result: want}
	svc := NewActionService(w)

	got, err := svc.UpdateHandoverItems(context.Background(), command.HandoverItemUpdateInput{
		HandoverID: "ho-1",
		Items:      []command.HandoverItemInput{{DocumentID: "doc-1", Selected: true}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.called != "UpdateHandoverItems" {
		t.Errorf("called = %q, want UpdateHandoverItems", w.called)
	}
	if got["updated"] != true {
		t.Errorf("result = %v, want updated=true", got)
	}
}

func TestApplyHandover_Delegates(t *testing.T) {
	want := map[string]any{"status": "completed"}
	w := &mockActionWriter{result: want}
	svc := NewActionService(w)

	got, err := svc.ApplyHandover(context.Background(), command.HandoverActionInput{
		HandoverID: "ho-1", Action: "confirm", ActorID: "u-2",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.called != "ApplyHandover" {
		t.Errorf("called = %q, want ApplyHandover", w.called)
	}
	if got["status"] != "completed" {
		t.Errorf("result status = %v, want completed", got["status"])
	}
}
