package router

import (
	"net/http"

	"digidocs-mgt/backend-go/internal/bootstrap"
	"digidocs-mgt/backend-go/internal/config"
	"digidocs-mgt/backend-go/internal/transport/http/handlers"
	"digidocs-mgt/backend-go/internal/transport/http/middleware"
)

func New(cfg config.Config, container bootstrap.Container) http.Handler {
	mux := http.NewServeMux()

	systemHandler := handlers.NewSystemHandler(cfg)
	authHandler := handlers.NewAuthHandler(container.AuthService, container.TokenService)
	assistantHandler := handlers.NewAssistantHandler(container.AssistantService)
	auditEventHandler := handlers.NewAuditEventHandler(container.AuditQueryService)
	flowHandler := handlers.NewFlowHandler(container.FlowService)
	handoverHandler := handlers.NewHandoverHandler(container.HandoverService)
	dashboardHandler := handlers.NewDashboardHandler(container.DashboardQueryService)
	internalWorkerHandler := handlers.NewInternalWorkerHandler(cfg, container.QueueConsumer, container.AssistantService)
	internalAssistantContextHandler := handlers.NewInternalAssistantContextHandler(
		cfg,
		container.AssistantService,
		container.DocumentService,
		container.VersionService,
		container.FlowService,
		container.HandoverService,
		container.DashboardQueryService,
	)
	teamSpaceHandler := handlers.NewTeamSpaceHandler(container.QueryService)
	userHandler := handlers.NewUserHandler(container.QueryService)
	projectHandler := handlers.NewProjectHandler(container.QueryService)
	documentHandler := handlers.NewDocumentHandler(container.DocumentService)
	versionHandler := handlers.NewVersionHandler(container.VersionService)

	authMw := middleware.Auth(container.TokenService)
	protect := func(h http.HandlerFunc) http.Handler {
		return authMw(h)
	}

	// --- Public routes (no JWT required) ---
	mux.HandleFunc("GET /healthz", systemHandler.Healthz)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/system/info", systemHandler.Info)
	mux.HandleFunc("POST "+cfg.APIV1Prefix+"/auth/login", authHandler.Login)
	// Worker callback uses its own shared-secret token, not user JWT.
	mux.HandleFunc("POST "+cfg.APIV1Prefix+"/internal/worker-results", internalWorkerHandler.ReceiveResult)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/internal/poll-tasks", internalWorkerHandler.PollTasks)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/internal/assistant-context/projects/{projectID}", internalAssistantContextHandler.GetProjectContext)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/internal/assistant-context/documents/{documentID}", internalAssistantContextHandler.GetDocumentContext)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/internal/assistant-context/handovers/{handoverID}", internalAssistantContextHandler.GetHandoverContext)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/internal/assistant-assets/versions/{versionID}/download", internalAssistantContextHandler.DownloadVersionFile)

	// --- Protected routes (JWT required) ---
	mux.Handle("GET "+cfg.APIV1Prefix+"/auth/me", protect(authHandler.Me))
	mux.Handle("POST "+cfg.APIV1Prefix+"/auth/logout", protect(authHandler.Logout))
	mux.Handle("GET "+cfg.APIV1Prefix+"/users", protect(userHandler.List))
	mux.Handle("POST "+cfg.APIV1Prefix+"/assistant/conversations", protect(assistantHandler.CreateConversation))
	mux.Handle("GET "+cfg.APIV1Prefix+"/assistant/conversations", protect(assistantHandler.ListConversations))
	mux.Handle("GET "+cfg.APIV1Prefix+"/assistant/conversations/{conversationID}", protect(assistantHandler.GetConversation))
	mux.Handle("GET "+cfg.APIV1Prefix+"/assistant/conversations/{conversationID}/messages", protect(assistantHandler.ListConversationMessages))
	mux.Handle("POST "+cfg.APIV1Prefix+"/assistant/ask", protect(assistantHandler.Ask))
	mux.Handle("GET "+cfg.APIV1Prefix+"/assistant/requests", protect(assistantHandler.ListRequests))
	mux.Handle("GET "+cfg.APIV1Prefix+"/assistant/requests/{requestID}", protect(assistantHandler.GetRequest))
	mux.Handle("POST "+cfg.APIV1Prefix+"/assistant/documents/{documentID}/summarize", protect(assistantHandler.SummarizeDocument))
	mux.Handle("POST "+cfg.APIV1Prefix+"/assistant/handovers/{handoverID}/summarize", protect(assistantHandler.SummarizeHandover))
	mux.Handle("GET "+cfg.APIV1Prefix+"/assistant/suggestions", protect(assistantHandler.ListSuggestions))
	mux.Handle("POST "+cfg.APIV1Prefix+"/assistant/suggestions/{suggestionID}/confirm", protect(assistantHandler.ConfirmSuggestion))
	mux.Handle("POST "+cfg.APIV1Prefix+"/assistant/suggestions/{suggestionID}/dismiss", protect(assistantHandler.DismissSuggestion))
	mux.Handle("GET "+cfg.APIV1Prefix+"/audit-events", protect(auditEventHandler.List))
	mux.Handle("GET "+cfg.APIV1Prefix+"/audit-events/summary", protect(auditEventHandler.Summary))
	mux.Handle("GET "+cfg.APIV1Prefix+"/team-spaces", protect(teamSpaceHandler.List))
	mux.Handle("GET "+cfg.APIV1Prefix+"/projects", protect(projectHandler.List))
	mux.Handle("GET "+cfg.APIV1Prefix+"/projects/{projectID}/folders/tree", protect(projectHandler.FolderTree))
	mux.Handle("POST "+cfg.APIV1Prefix+"/documents", protect(documentHandler.Create))
	mux.Handle("GET "+cfg.APIV1Prefix+"/documents", protect(documentHandler.List))
	mux.Handle("GET "+cfg.APIV1Prefix+"/documents/{documentID}", protect(documentHandler.Get))
	mux.Handle("PATCH "+cfg.APIV1Prefix+"/documents/{documentID}", protect(documentHandler.Update))
	mux.Handle("POST "+cfg.APIV1Prefix+"/documents/{documentID}/delete", protect(documentHandler.Delete))
	mux.Handle("POST "+cfg.APIV1Prefix+"/documents/{documentID}/restore", protect(documentHandler.Restore))
	mux.Handle("POST "+cfg.APIV1Prefix+"/documents/{documentID}/versions", protect(versionHandler.Upload))
	mux.Handle("GET "+cfg.APIV1Prefix+"/documents/{documentID}/versions", protect(versionHandler.List))
	mux.Handle("GET "+cfg.APIV1Prefix+"/versions/{versionID}", protect(versionHandler.Get))
	mux.Handle("GET "+cfg.APIV1Prefix+"/versions/{versionID}/download", protect(versionHandler.Download))
	mux.Handle("GET "+cfg.APIV1Prefix+"/versions/{versionID}/preview", protect(versionHandler.Preview))
	mux.Handle("POST "+cfg.APIV1Prefix+"/documents/{documentID}/flow/mark-in-progress", protect(flowHandler.MarkInProgress))
	mux.Handle("POST "+cfg.APIV1Prefix+"/documents/{documentID}/flow/transfer", protect(flowHandler.Transfer))
	mux.Handle("POST "+cfg.APIV1Prefix+"/documents/{documentID}/flow/accept-transfer", protect(flowHandler.AcceptTransfer))
	mux.Handle("POST "+cfg.APIV1Prefix+"/documents/{documentID}/flow/finalize", protect(flowHandler.Finalize))
	mux.Handle("POST "+cfg.APIV1Prefix+"/documents/{documentID}/flow/archive", protect(flowHandler.Archive))
	mux.Handle("POST "+cfg.APIV1Prefix+"/documents/{documentID}/flow/unarchive", protect(flowHandler.Unarchive))
	mux.Handle("GET "+cfg.APIV1Prefix+"/documents/{documentID}/flows", protect(flowHandler.List))
	mux.Handle("POST "+cfg.APIV1Prefix+"/handovers", protect(handoverHandler.Create))
	mux.Handle("GET "+cfg.APIV1Prefix+"/handovers", protect(handoverHandler.List))
	mux.Handle("GET "+cfg.APIV1Prefix+"/handovers/{handoverID}", protect(handoverHandler.Get))
	mux.Handle("PATCH "+cfg.APIV1Prefix+"/handovers/{handoverID}/items", protect(handoverHandler.UpdateItems))
	mux.Handle("POST "+cfg.APIV1Prefix+"/handovers/{handoverID}/confirm", protect(handoverHandler.Confirm))
	mux.Handle("POST "+cfg.APIV1Prefix+"/handovers/{handoverID}/complete", protect(handoverHandler.Complete))
	mux.Handle("POST "+cfg.APIV1Prefix+"/handovers/{handoverID}/cancel", protect(handoverHandler.Cancel))
	mux.Handle("GET "+cfg.APIV1Prefix+"/dashboard/overview", protect(dashboardHandler.Overview))
	mux.Handle("GET "+cfg.APIV1Prefix+"/dashboard/recent-flows", protect(dashboardHandler.RecentFlows))
	mux.Handle("GET "+cfg.APIV1Prefix+"/dashboard/risk-documents", protect(dashboardHandler.RiskDocuments))

	return middleware.Chain(
		mux,
		middleware.CORS(cfg.CORSAllowOrigins),
		middleware.RequestID,
		middleware.JSONContentType,
		middleware.AccessLog,
	)
}
