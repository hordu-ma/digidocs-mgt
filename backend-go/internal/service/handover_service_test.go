package service

import (
	"context"
	"errors"
	"testing"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
)

// --- mock: HandoverActionWriter ---

type mockHandoverActionWriter struct {
	result map[string]any
	err    error
	called string
}

func (m *mockHandoverActionWriter) CreateFlowRecord(_ context.Context, _ command.FlowActionInput) (map[string]any, error) {
	return nil, nil
}

func (m *mockHandoverActionWriter) CreateHandover(_ context.Context, _ command.HandoverCreateInput) (map[string]any, error) {
	m.called = "CreateHandover"
	return m.result, m.err
}

func (m *mockHandoverActionWriter) UpdateHandoverItems(_ context.Context, _ command.HandoverItemUpdateInput) (map[string]any, error) {
	m.called = "UpdateHandoverItems"
	return m.result, m.err
}

func (m *mockHandoverActionWriter) ApplyHandover(_ context.Context, _ command.HandoverActionInput) (map[string]any, error) {
	m.called = "ApplyHandover"
	return m.result, m.err
}

// --- mock: HandoverReader ---

type mockHandoverReader struct {
	items []query.HandoverItem
	item  *query.HandoverItem
	err   error
}

func (m *mockHandoverReader) ListHandovers(_ context.Context) ([]query.HandoverItem, error) {
	return m.items, m.err
}

func (m *mockHandoverReader) GetHandover(_ context.Context, _ string) (*query.HandoverItem, error) {
	return m.item, m.err
}

// --- tests: Create ---

func TestHandoverService_Create_OK(t *testing.T) {
	w := &mockHandoverActionWriter{result: map[string]any{"id": "ho-1", "status": "generated"}}
	svc := NewHandoverService(nil, w)
	data, err := svc.Create(context.Background(), command.HandoverCreateInput{
		TargetUserID: "u-1", ReceiverUserID: "u-2", ActorID: "u-3",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.called != "CreateHandover" {
		t.Errorf("called = %q, want CreateHandover", w.called)
	}
	if data["id"] != "ho-1" {
		t.Errorf("id = %v, want ho-1", data["id"])
	}
}

func TestHandoverService_Create_MissingTargetUserID(t *testing.T) {
	svc := NewHandoverService(nil, &mockHandoverActionWriter{})
	_, err := svc.Create(context.Background(), command.HandoverCreateInput{
		ReceiverUserID: "u-2", ActorID: "u-3",
	})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}

func TestHandoverService_Create_MissingReceiverUserID(t *testing.T) {
	svc := NewHandoverService(nil, &mockHandoverActionWriter{})
	_, err := svc.Create(context.Background(), command.HandoverCreateInput{
		TargetUserID: "u-1", ActorID: "u-3",
	})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}

func TestHandoverService_Create_MissingActorID(t *testing.T) {
	svc := NewHandoverService(nil, &mockHandoverActionWriter{})
	_, err := svc.Create(context.Background(), command.HandoverCreateInput{
		TargetUserID: "u-1", ReceiverUserID: "u-2",
	})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}

// --- tests: UpdateItems ---

func TestHandoverService_UpdateItems_OK(t *testing.T) {
	w := &mockHandoverActionWriter{result: map[string]any{"id": "ho-1"}}
	svc := NewHandoverService(nil, w)
	data, err := svc.UpdateItems(context.Background(), command.HandoverItemUpdateInput{
		HandoverID: "ho-1",
		Items:      []command.HandoverItemInput{{DocumentID: "doc-1", Selected: true}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.called != "UpdateHandoverItems" {
		t.Errorf("called = %q, want UpdateHandoverItems", w.called)
	}
	if data["id"] != "ho-1" {
		t.Errorf("id = %v, want ho-1", data["id"])
	}
}

func TestHandoverService_UpdateItems_MissingHandoverID(t *testing.T) {
	svc := NewHandoverService(nil, &mockHandoverActionWriter{})
	_, err := svc.UpdateItems(context.Background(), command.HandoverItemUpdateInput{
		Items: []command.HandoverItemInput{{DocumentID: "doc-1"}},
	})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}

func TestHandoverService_UpdateItems_EmptyItems(t *testing.T) {
	svc := NewHandoverService(nil, &mockHandoverActionWriter{})
	_, err := svc.UpdateItems(context.Background(), command.HandoverItemUpdateInput{
		HandoverID: "ho-1",
	})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}

// --- tests: ApplyAction ---

func TestHandoverService_ApplyAction_OK(t *testing.T) {
	w := &mockHandoverActionWriter{result: map[string]any{"status": "pending_confirm"}}
	svc := NewHandoverService(nil, w)
	data, err := svc.ApplyAction(context.Background(), command.HandoverActionInput{
		HandoverID: "ho-1", Action: "confirm", ActorID: "u-2",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.called != "ApplyHandover" {
		t.Errorf("called = %q, want ApplyHandover", w.called)
	}
	if data["status"] != "pending_confirm" {
		t.Errorf("status = %v, want pending_confirm", data["status"])
	}
}

func TestHandoverService_ApplyAction_MissingHandoverID(t *testing.T) {
	svc := NewHandoverService(nil, &mockHandoverActionWriter{})
	_, err := svc.ApplyAction(context.Background(), command.HandoverActionInput{
		Action: "confirm", ActorID: "u-2",
	})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}

func TestHandoverService_ApplyAction_UnknownAction(t *testing.T) {
	svc := NewHandoverService(nil, &mockHandoverActionWriter{})
	_, err := svc.ApplyAction(context.Background(), command.HandoverActionInput{
		HandoverID: "ho-1", Action: "explode", ActorID: "u-2",
	})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}

func TestHandoverService_ApplyAction_MissingActorID(t *testing.T) {
	svc := NewHandoverService(nil, &mockHandoverActionWriter{})
	_, err := svc.ApplyAction(context.Background(), command.HandoverActionInput{
		HandoverID: "ho-1", Action: "confirm",
	})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}

// --- tests: Get ---

func TestHandoverService_Get_MissingID(t *testing.T) {
	svc := NewHandoverService(&mockHandoverReader{}, &mockHandoverActionWriter{})
	_, err := svc.Get(context.Background(), "")
	if !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}

func TestHandoverService_Get_OK(t *testing.T) {
	r := &mockHandoverReader{item: &query.HandoverItem{ID: "ho-1", Status: "generated"}}
	svc := NewHandoverService(r, &mockHandoverActionWriter{})
	item, err := svc.Get(context.Background(), "ho-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.ID != "ho-1" {
		t.Errorf("id = %v, want ho-1", item.ID)
	}
}
