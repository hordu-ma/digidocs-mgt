package handlers_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"digidocs-mgt/backend-go/internal/bootstrap"
	"digidocs-mgt/backend-go/internal/config"
	"digidocs-mgt/backend-go/internal/domain/auth"
	"digidocs-mgt/backend-go/internal/domain/query"
	noopqueue "digidocs-mgt/backend-go/internal/queue/noop"
	"digidocs-mgt/backend-go/internal/repository/memory"
	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/transport/http/router"
)

// These tests guard the fix for the cross-project read leak: read endpoints that
// resolve a single resource (a version, a document's history, a handover, a code
// repository's push events) used to skip the permission layer entirely, so any
// authenticated user could read — and download — another project's data by
// harvesting ids from the audit log.
//
// The memory PermissionRepository always grants, which is why the leak was
// invisible to the existing handler tests. Here the container is rebuilt with a
// reader that always denies, so a handler that forgets to consult the permission
// layer shows up as a 200 where a 403 is required.

type denyAllPermissionReader struct{}

func (denyAllPermissionReader) CanCreateDocument(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (denyAllPermissionReader) CanUpdateDocument(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (denyAllPermissionReader) CanDeleteDocument(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (denyAllPermissionReader) CanUploadVersion(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (denyAllPermissionReader) CanFlowDocument(context.Context, string, string, string, string) (bool, error) {
	return false, nil
}
func (denyAllPermissionReader) CanCreateHandover(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (denyAllPermissionReader) CanUpdateHandoverItems(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (denyAllPermissionReader) CanApplyHandover(context.Context, string, string, string, string) (bool, error) {
	return false, nil
}
func (denyAllPermissionReader) CanUploadDataAsset(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (denyAllPermissionReader) CanManageDataAsset(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (denyAllPermissionReader) CanViewDocument(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (denyAllPermissionReader) CanViewVersion(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (denyAllPermissionReader) CanViewHandover(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (denyAllPermissionReader) CanViewProject(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (denyAllPermissionReader) CanViewDataAsset(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (denyAllPermissionReader) CanViewCodeRepository(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (denyAllPermissionReader) CanCreateCodeRepository(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (denyAllPermissionReader) CanManageCodeRepository(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (denyAllPermissionReader) CanPushCodeRepository(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (denyAllPermissionReader) CanViewProjectMembers(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (denyAllPermissionReader) CanManageProjectMembers(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (denyAllPermissionReader) CanViewAssistantRequest(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (denyAllPermissionReader) CanViewAssistantSuggestion(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (denyAllPermissionReader) CanViewConversation(context.Context, string, string, string) (bool, error) {
	return false, nil
}

// scopeSpy records the ActorScope the aggregate endpoints push into the
// repository layer. Those endpoints cannot 403 (they legitimately return an
// empty page for an outsider), so the assertion is that the scope arrives.
type scopeSpy struct {
	dashboardScope query.ActorScope
	auditScope     query.ActorScope
	handoverScope  query.ActorScope
}

func (s *scopeSpy) GetOverview(_ context.Context, _ string, scope query.ActorScope) (query.DashboardOverview, error) {
	s.dashboardScope = scope
	return query.DashboardOverview{StatusCounts: map[string]int{}}, nil
}

func (s *scopeSpy) ListRecentFlows(_ context.Context, _ string, scope query.ActorScope) ([]query.RecentFlowItem, error) {
	s.dashboardScope = scope
	return []query.RecentFlowItem{}, nil
}

func (s *scopeSpy) ListRiskDocuments(_ context.Context, _ string, scope query.ActorScope) ([]query.RiskDocumentItem, error) {
	s.dashboardScope = scope
	return []query.RiskDocumentItem{}, nil
}

func (s *scopeSpy) ListAuditEvents(_ context.Context, filter query.AuditEventFilter) ([]query.AuditEventItem, int, error) {
	s.auditScope = filter.Scope
	return []query.AuditEventItem{}, 0, nil
}

func (s *scopeSpy) GetAuditSummary(_ context.Context, projectID string, scope query.ActorScope) (query.AuditSummary, error) {
	s.auditScope = scope
	return query.AuditSummary{ProjectID: projectID, TopActiveUsers: []query.AuditUserMetric{}}, nil
}

func (s *scopeSpy) ListHandovers(_ context.Context, scope query.ActorScope) ([]query.HandoverItem, error) {
	s.handoverScope = scope
	return []query.HandoverItem{}, nil
}

func (s *scopeSpy) GetHandover(_ context.Context, handoverID string) (*query.HandoverItem, error) {
	return &query.HandoverItem{ID: handoverID}, nil
}

// lockedDownServer builds the real router over memory repositories, but with a
// permission layer that denies everything and a spy on the scoped aggregates.
func lockedDownServer(t *testing.T) (http.Handler, *scopeSpy, string) {
	t.Helper()

	cfg := config.Config{
		APIV1Prefix:         "/api/v1",
		DataBackend:         "memory",
		JWTSecret:           "test-secret-key-for-handler-tests",
		WorkerCallbackToken: "worker-test-token",
		CodeRepoRoot:        t.TempDir(),
	}

	container, err := bootstrap.BuildContainer(cfg)
	if err != nil {
		t.Fatalf("failed to build container: %v", err)
	}

	permissions := service.NewPermissionService(denyAllPermissionReader{})
	spy := &scopeSpy{}

	storageProvider := memory.NewVersionRepository()
	actionRepo := memory.NewActionRepository()
	docRepo := memory.NewDocumentRepository()
	versionWorkflow := memory.NewVersionWorkflow(storageProvider)
	dataAssetRepo := memory.NewDataAssetRepository()
	codeRepo := memory.NewCodeRepositoryRepository()

	container.DocumentService = service.NewDocumentService(docRepo, docRepo, nil, versionWorkflow, permissions)
	container.AssistantService = service.NewAssistantService(noopqueue.NewPublisher(), memory.NewAssistantRepository(), docRepo, permissions)
	container.VersionService = service.NewVersionService(nil, versionWorkflow, storageProvider, permissions)
	container.FlowService = service.NewFlowService(memory.NewFlowRepository(), actionRepo, permissions)
	container.HandoverService = service.NewHandoverService(spy, actionRepo, permissions)
	container.DataAssetService = service.NewDataAssetService(dataAssetRepo, dataAssetRepo, nil, permissions)
	container.CodeRepositoryService = service.NewCodeRepositoryService(codeRepo, codeRepo, permissions, cfg.CodeRepoRoot, nil)
	container.AuditQueryService = service.NewAuditQueryService(spy)
	container.DashboardQueryService = service.NewDashboardQueryService(spy)

	token, err := container.TokenService.Generate(auth.Claims{
		UserID:      "00000000-0000-0000-0000-0000000000ff",
		Username:    "outsider",
		DisplayName: "Outsider",
		Role:        "member",
	})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	return router.New(cfg, container), spy, token
}

// TestReadEndpointsDenyNonMember is the direct regression test for the leak.
// Every entry here returned 200 to a non-member before the fix.
func TestReadEndpointsDenyNonMember(t *testing.T) {
	handler, _, token := lockedDownServer(t)

	const (
		docID      = "00000000-0000-0000-0000-000000000100"
		versionID  = "00000000-0000-0000-0000-000000000200"
		handoverID = "00000000-0000-0000-0000-000000000300"
		repoID     = "00000000-0000-0000-0000-000000000400"
	)

	cases := []struct {
		name string
		path string
	}{
		{"document detail", "/api/v1/documents/" + docID},
		{"version list", "/api/v1/documents/" + docID + "/versions"},
		{"flow history", "/api/v1/documents/" + docID + "/flows"},
		{"version detail", "/api/v1/versions/" + versionID},
		{"version download", "/api/v1/versions/" + versionID + "/download"},
		{"version preview", "/api/v1/versions/" + versionID + "/preview"},
		{"handover detail", "/api/v1/handovers/" + handoverID},
		{"code repository detail", "/api/v1/code-repositories/" + repoID},
		{"code push events", "/api/v1/code-repositories/" + repoID + "/push-events"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, authedRequest(http.MethodGet, tc.path, nil, token))
			if rec.Code != http.StatusForbidden {
				t.Fatalf("GET %s = %d, want 403 (body: %s)", tc.path, rec.Code, rec.Body.String())
			}
		})
	}
}

// TestAggregateEndpointsCarryActorScope covers the endpoints that cannot answer
// with 403 because an outsider legitimately sees an empty result: they must push
// the caller down to the repository so the SQL can narrow the rows.
func TestAggregateEndpointsCarryActorScope(t *testing.T) {
	handler, spy, token := lockedDownServer(t)

	cases := []struct {
		name  string
		path  string
		check func() query.ActorScope
	}{
		{"dashboard overview", "/api/v1/dashboard/overview", func() query.ActorScope { return spy.dashboardScope }},
		{"dashboard recent flows", "/api/v1/dashboard/recent-flows", func() query.ActorScope { return spy.dashboardScope }},
		{"dashboard risk documents", "/api/v1/dashboard/risk-documents", func() query.ActorScope { return spy.dashboardScope }},
		{"audit events", "/api/v1/audit-events", func() query.ActorScope { return spy.auditScope }},
		{"audit summary", "/api/v1/audit-events/summary", func() query.ActorScope { return spy.auditScope }},
		{"handover list", "/api/v1/handovers", func() query.ActorScope { return spy.handoverScope }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, authedRequest(http.MethodGet, tc.path, nil, token))
			if rec.Code != http.StatusOK {
				t.Fatalf("GET %s = %d, want 200 (body: %s)", tc.path, rec.Code, rec.Body.String())
			}
			scope := tc.check()
			if scope.ActorID != "00000000-0000-0000-0000-0000000000ff" {
				t.Errorf("scope.ActorID = %q, want the calling member's id", scope.ActorID)
			}
			if scope.IsAdmin {
				t.Error("scope.IsAdmin = true for a member token, want false")
			}
		})
	}
}

// TestAssistantEndpointsDenyNonMember is the regression test for the Assistant
// subsystem's own version of the cross-project leak: the P0-1 fix locked down
// document/version/handover/flow/project/dashboard/audit-events, but every one
// of these AI endpoints skipped the permission layer entirely, so an outsider
// could read AI-generated answers/suggestions, read or post into someone
// else's conversation, and even trigger a summary job against a document they
// cannot view. Every entry here returned 200 (or a queued job) before the fix.
func TestAssistantEndpointsDenyNonMember(t *testing.T) {
	handler, _, token := lockedDownServer(t)

	const (
		docID          = "00000000-0000-0000-0000-000000000100"
		handoverID     = "00000000-0000-0000-0000-000000000300"
		requestID      = "00000000-0000-0000-0000-000000000500"
		suggestionID   = "00000000-0000-0000-0000-000000000600"
		conversationID = "00000000-0000-0000-0000-000000000700"
	)

	cases := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"ask", http.MethodPost, "/api/v1/assistant/ask", map[string]any{"question": "q", "document_id": docID}},
		{"create conversation", http.MethodPost, "/api/v1/assistant/conversations", map[string]any{"document_id": docID}},
		{"get conversation", http.MethodGet, "/api/v1/assistant/conversations/" + conversationID, nil},
		{"list conversation messages", http.MethodGet, "/api/v1/assistant/conversations/" + conversationID + "/messages", nil},
		{"archive conversation", http.MethodPatch, "/api/v1/assistant/conversations/" + conversationID + "/archive", map[string]any{"archive": true}},
		{"summarize document", http.MethodPost, "/api/v1/assistant/documents/" + docID + "/summarize", map[string]any{}},
		{"summarize handover", http.MethodPost, "/api/v1/assistant/handovers/" + handoverID + "/summarize", map[string]any{}},
		{"get request", http.MethodGet, "/api/v1/assistant/requests/" + requestID, nil},
		{"confirm suggestion", http.MethodPost, "/api/v1/assistant/suggestions/" + suggestionID + "/confirm", map[string]any{}},
		{"dismiss suggestion", http.MethodPost, "/api/v1/assistant/suggestions/" + suggestionID + "/dismiss", map[string]any{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var body io.Reader
			if tc.body != nil {
				body = jsonBody(tc.body)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, authedRequest(tc.method, tc.path, body, token))
			if rec.Code != http.StatusForbidden {
				t.Fatalf("%s %s = %d, want 403 (body: %s)", tc.method, tc.path, rec.Code, rec.Body.String())
			}
		})
	}
}

// TestAggregateEndpointsScopeAdmin proves the narrowing is actually bypassed for
// platform admins, so the fix does not silently blank out the admin console.
func TestAggregateEndpointsScopeAdmin(t *testing.T) {
	handler, spy, _ := lockedDownServer(t)
	adminToken := tokenForRole(t, "admin")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, authedRequest(http.MethodGet, "/api/v1/dashboard/overview", nil, adminToken))
	if rec.Code != http.StatusOK {
		t.Fatalf("admin overview = %d, want 200", rec.Code)
	}
	if !spy.dashboardScope.IsAdmin {
		t.Error("scope.IsAdmin = false for an admin token, want true")
	}
}
