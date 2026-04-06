package service

import (
	"context"
	"errors"
	"testing"

	"digidocs-mgt/backend-go/internal/domain/command"
)

// --- mock: FlowActionWriter ---

type mockFlowActionWriter struct {
	result map[string]any
	err    error
	called string
}

func (m *mockFlowActionWriter) CreateFlowRecord(_ context.Context, _ command.FlowActionInput) (map[string]any, error) {
	m.called = "CreateFlowRecord"
	return m.result, m.err
}

func (m *mockFlowActionWriter) CreateHandover(_ context.Context, _ command.HandoverCreateInput) (map[string]any, error) {
	return nil, nil
}

func (m *mockFlowActionWriter) UpdateHandoverItems(_ context.Context, _ command.HandoverItemUpdateInput) (map[string]any, error) {
	return nil, nil
}

func (m *mockFlowActionWriter) ApplyHandover(_ context.Context, _ command.HandoverActionInput) (map[string]any, error) {
	return nil, nil
}

// --- tests ---

func TestFlowService_ApplyAction_OK(t *testing.T) {
	w := &mockFlowActionWriter{result: map[string]any{"action": "finalize"}}
	svc := NewFlowService(nil, w)
	data, err := svc.ApplyAction(context.Background(), command.FlowActionInput{
		DocumentID: "doc-1", Action: "finalize", ActorID: "u-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.called != "CreateFlowRecord" {
		t.Errorf("called = %q, want CreateFlowRecord", w.called)
	}
	if data["action"] != "finalize" {
		t.Errorf("action = %v, want finalize", data["action"])
	}
}

func TestFlowService_ApplyAction_MissingDocumentID(t *testing.T) {
	svc := NewFlowService(nil, &mockFlowActionWriter{})
	_, err := svc.ApplyAction(context.Background(), command.FlowActionInput{
		Action: "finalize", ActorID: "u-1",
	})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}

func TestFlowService_ApplyAction_UnknownAction(t *testing.T) {
	svc := NewFlowService(nil, &mockFlowActionWriter{})
	_, err := svc.ApplyAction(context.Background(), command.FlowActionInput{
		DocumentID: "doc-1", Action: "explode", ActorID: "u-1",
	})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}

func TestFlowService_ApplyAction_TransferMissingToUserID(t *testing.T) {
	svc := NewFlowService(nil, &mockFlowActionWriter{})
	_, err := svc.ApplyAction(context.Background(), command.FlowActionInput{
		DocumentID: "doc-1", Action: "transfer", ActorID: "u-1",
	})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}

func TestFlowService_ApplyAction_TransferToSelf(t *testing.T) {
	svc := NewFlowService(nil, &mockFlowActionWriter{})
	_, err := svc.ApplyAction(context.Background(), command.FlowActionInput{
		DocumentID: "doc-1", Action: "transfer", ActorID: "u-1", ToUserID: "u-1",
	})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}

func TestFlowService_ApplyAction_MissingActorID(t *testing.T) {
	svc := NewFlowService(nil, &mockFlowActionWriter{})
	_, err := svc.ApplyAction(context.Background(), command.FlowActionInput{
		DocumentID: "doc-1", Action: "finalize",
	})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}

func TestFlowService_ApplyAction_PropagatesRepoError(t *testing.T) {
	repoErr := errors.New("db error")
	w := &mockFlowActionWriter{err: repoErr}
	svc := NewFlowService(nil, w)
	_, err := svc.ApplyAction(context.Background(), command.FlowActionInput{
		DocumentID: "doc-1", Action: "finalize", ActorID: "u-1",
	})
	if !errors.Is(err, repoErr) {
		t.Errorf("err = %v, want %v", err, repoErr)
	}
}
