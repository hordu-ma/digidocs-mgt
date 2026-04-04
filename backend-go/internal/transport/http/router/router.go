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
	authHandler := handlers.NewAuthHandler(container.TokenService)
	assistantHandler := handlers.NewAssistantHandler(container.TaskService)
	flowHandler := handlers.NewFlowHandler(container.FlowQueryService, container.ActionService)
	handoverHandler := handlers.NewHandoverHandler(container.HandoverQueryService, container.ActionService)
	dashboardHandler := handlers.NewDashboardHandler()
	internalWorkerHandler := handlers.NewInternalWorkerHandler(cfg)
	teamSpaceHandler := handlers.NewTeamSpaceHandler(container.QueryService)
	projectHandler := handlers.NewProjectHandler(container.QueryService)
	documentHandler := handlers.NewDocumentHandler(container.QueryService)
	versionHandler := handlers.NewVersionHandler(
		container.UploadService,
		container.AuditService,
		container.VersionQueryService,
	)

	mux.HandleFunc("GET /healthz", systemHandler.Healthz)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/system/info", systemHandler.Info)
	mux.HandleFunc("POST "+cfg.APIV1Prefix+"/auth/login", authHandler.Login)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/auth/me", authHandler.Me)
	mux.HandleFunc("POST "+cfg.APIV1Prefix+"/auth/logout", authHandler.Logout)
	mux.HandleFunc("POST "+cfg.APIV1Prefix+"/assistant/ask", assistantHandler.Ask)
	mux.HandleFunc("POST "+cfg.APIV1Prefix+"/assistant/documents/{documentID}/summarize", assistantHandler.SummarizeDocument)
	mux.HandleFunc("POST "+cfg.APIV1Prefix+"/assistant/handovers/{handoverID}/summarize", assistantHandler.SummarizeHandover)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/assistant/suggestions", assistantHandler.ListSuggestions)
	mux.HandleFunc("POST "+cfg.APIV1Prefix+"/assistant/suggestions/{suggestionID}/confirm", assistantHandler.ConfirmSuggestion)
	mux.HandleFunc("POST "+cfg.APIV1Prefix+"/assistant/suggestions/{suggestionID}/dismiss", assistantHandler.DismissSuggestion)
	mux.HandleFunc("POST "+cfg.APIV1Prefix+"/internal/worker-results", internalWorkerHandler.ReceiveResult)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/team-spaces", teamSpaceHandler.List)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/projects", projectHandler.List)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/projects/{projectID}/folders/tree", projectHandler.FolderTree)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/documents", documentHandler.List)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/documents/{documentID}", documentHandler.Get)
	mux.HandleFunc("POST "+cfg.APIV1Prefix+"/documents/{documentID}/versions", versionHandler.Upload)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/documents/{documentID}/versions", versionHandler.List)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/versions/{versionID}", versionHandler.Get)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/versions/{versionID}/download", versionHandler.Download)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/versions/{versionID}/preview", versionHandler.Preview)
	mux.HandleFunc("POST "+cfg.APIV1Prefix+"/documents/{documentID}/flow/mark-in-progress", flowHandler.MarkInProgress)
	mux.HandleFunc("POST "+cfg.APIV1Prefix+"/documents/{documentID}/flow/transfer", flowHandler.Transfer)
	mux.HandleFunc("POST "+cfg.APIV1Prefix+"/documents/{documentID}/flow/accept-transfer", flowHandler.AcceptTransfer)
	mux.HandleFunc("POST "+cfg.APIV1Prefix+"/documents/{documentID}/flow/finalize", flowHandler.Finalize)
	mux.HandleFunc("POST "+cfg.APIV1Prefix+"/documents/{documentID}/flow/archive", flowHandler.Archive)
	mux.HandleFunc("POST "+cfg.APIV1Prefix+"/documents/{documentID}/flow/unarchive", flowHandler.Unarchive)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/documents/{documentID}/flows", flowHandler.List)
	mux.HandleFunc("POST "+cfg.APIV1Prefix+"/handovers", handoverHandler.Create)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/handovers", handoverHandler.List)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/handovers/{handoverID}", handoverHandler.Get)
	mux.HandleFunc("PATCH "+cfg.APIV1Prefix+"/handovers/{handoverID}/items", handoverHandler.UpdateItems)
	mux.HandleFunc("POST "+cfg.APIV1Prefix+"/handovers/{handoverID}/confirm", handoverHandler.Confirm)
	mux.HandleFunc("POST "+cfg.APIV1Prefix+"/handovers/{handoverID}/complete", handoverHandler.Complete)
	mux.HandleFunc("POST "+cfg.APIV1Prefix+"/handovers/{handoverID}/cancel", handoverHandler.Cancel)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/dashboard/overview", dashboardHandler.Overview)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/dashboard/recent-flows", dashboardHandler.RecentFlows)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/dashboard/risk-documents", dashboardHandler.RiskDocuments)

	return middleware.Chain(
		mux,
		middleware.RequestID,
		middleware.JSONContentType,
		middleware.AccessLog,
	)
}
