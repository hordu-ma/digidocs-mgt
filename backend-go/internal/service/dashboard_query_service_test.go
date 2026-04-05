package service

import (
	"context"
	"errors"
	"testing"

	"digidocs-mgt/backend-go/internal/domain/query"
)

// --- mock: DashboardReader ---

type mockDashboardReader struct {
	overview  query.DashboardOverview
	flows     []query.RecentFlowItem
	risks     []query.RiskDocumentItem
	err       error
}

func (m *mockDashboardReader) GetOverview(_ context.Context, _ string) (query.DashboardOverview, error) {
	return m.overview, m.err
}

func (m *mockDashboardReader) ListRecentFlows(_ context.Context, _ string) ([]query.RecentFlowItem, error) {
	return m.flows, m.err
}

func (m *mockDashboardReader) ListRiskDocuments(_ context.Context, _ string) ([]query.RiskDocumentItem, error) {
	return m.risks, m.err
}

// --- tests ---

func TestOverview_Delegates(t *testing.T) {
	want := query.DashboardOverview{
		DocumentTotal:        42,
		StatusCounts:         map[string]int{"draft": 10, "active": 32},
		HandoverPendingCount: 3,
		RiskDocumentCount:    5,
	}
	r := &mockDashboardReader{overview: want}
	svc := NewDashboardQueryService(r)

	got, err := svc.Overview(context.Background(), "p-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.DocumentTotal != 42 {
		t.Errorf("DocumentTotal = %d, want 42", got.DocumentTotal)
	}
	if got.StatusCounts["draft"] != 10 {
		t.Errorf("StatusCounts[draft] = %d, want 10", got.StatusCounts["draft"])
	}
	if got.HandoverPendingCount != 3 {
		t.Errorf("HandoverPendingCount = %d, want 3", got.HandoverPendingCount)
	}
	if got.RiskDocumentCount != 5 {
		t.Errorf("RiskDocumentCount = %d, want 5", got.RiskDocumentCount)
	}
}

func TestOverview_PropagatesError(t *testing.T) {
	repoErr := errors.New("db timeout")
	r := &mockDashboardReader{err: repoErr}
	svc := NewDashboardQueryService(r)

	_, err := svc.Overview(context.Background(), "p-1")
	if !errors.Is(err, repoErr) {
		t.Errorf("err = %v, want %v", err, repoErr)
	}
}

func TestRecentFlows_Delegates(t *testing.T) {
	want := []query.RecentFlowItem{
		{DocumentID: "doc-1", Title: "Doc A", Action: "submit", ToStatus: "review"},
		{DocumentID: "doc-2", Title: "Doc B", Action: "approve", ToStatus: "active"},
	}
	r := &mockDashboardReader{flows: want}
	svc := NewDashboardQueryService(r)

	got, err := svc.RecentFlows(context.Background(), "p-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].DocumentID != "doc-1" {
		t.Errorf("got[0].DocumentID = %q, want doc-1", got[0].DocumentID)
	}
	if got[1].Action != "approve" {
		t.Errorf("got[1].Action = %q, want approve", got[1].Action)
	}
}

func TestRiskDocuments_Delegates(t *testing.T) {
	want := []query.RiskDocumentItem{
		{DocumentID: "doc-3", Title: "Risky", RiskType: "expired", RiskMessage: "overdue"},
	}
	r := &mockDashboardReader{risks: want}
	svc := NewDashboardQueryService(r)

	got, err := svc.RiskDocuments(context.Background(), "p-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].RiskType != "expired" {
		t.Errorf("RiskType = %q, want expired", got[0].RiskType)
	}
}
