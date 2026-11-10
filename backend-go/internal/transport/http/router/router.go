package router

import (
	"digidocs-mgt/backend-go/internal/metrics"
	"net/http"
	"time"

	"digidocs-mgt/backend-go/internal/bootstrap"
	"digidocs-mgt/backend-go/internal/config"
	"digidocs-mgt/backend-go/internal/transport/http/handlers"
	"digidocs-mgt/backend-go/internal/transport/http/middleware"
)

func New(cfg config.Config, container bootstrap.Container) http.Handler {
	mux := http.NewServeMux()

	// How many trailing X-Forwarded-For entries come from proxies we control.
	// Raising this trusts one more entry of client-supplied data, so it must
	// match the real chain — see middleware.clientIP.
	if cfg.TrustedProxyDepth > 0 {
		middleware.TrustedProxyDepth = cfg.TrustedProxyDepth
	}
	if cfg.APIV1Prefix != "" {
		middleware.APIV1Prefix = cfg.APIV1Prefix
	}

	systemHandler := handlers.NewSystemHandler(cfg)
	authHandler := handlers.NewAuthHandler(container.AuthService, container.TokenService).
		WithRegistration(cfg.SelfRegistration, container.MembershipService)
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
	).WithSearch(container.SearchService)
	teamSpaceHandler := handlers.NewTeamSpaceHandler(container.QueryService)
	userHandler := handlers.NewUserHandler(container.QueryService)
	projectHandler := handlers.NewProjectHandler(container.QueryService)
	documentHandler := handlers.NewDocumentHandler(container.DocumentService, container.AssistantService, container.AuditService)
	versionHandler := handlers.NewVersionHandler(container.VersionService, container.AssistantService, container.AuditService)
	adminHandler := handlers.NewAdminHandler(container.AdminService)
	dataAssetHandler := handlers.NewDataAssetHandler(container.DataAssetService, container.AuditService)
	codeRepositoryHandler := handlers.NewCodeRepositoryHandler(container.CodeRepositoryService, cfg.APIV1Prefix)
	membershipHandler := handlers.NewMembershipHandler(container.MembershipService)
	notificationHandler := handlers.NewNotificationHandler(container.NotificationService)
	searchHandler := handlers.NewSearchHandler(container.SearchService, container.DocumentService, container.AuditService)

	authMw := middleware.Auth(container.TokenService, container.AuthService.SessionValid)
	protect := func(h http.HandlerFunc) http.Handler {
		return authMw(h)
	}
	adminMw := middleware.RequireAdmin(container.TokenService, container.AuthService.SessionValid)
	protectAdmin := func(h http.HandlerFunc) http.Handler {
		return adminMw(h)
	}

	// --- Public routes (no JWT required) ---
	mux.HandleFunc("GET /healthz", systemHandler.Healthz)
	metricsRegistry := metrics.NewRegistry()
	mux.Handle("GET /metrics", metricsRegistry.Handler(container.DB, cfg.MetricsToken))
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/system/info", systemHandler.Info)
	loginLimiter := middleware.NewRateLimiter(cfg.LoginRateLimitPerMin, time.Minute)
	registerLimiter := middleware.NewRateLimiter(cfg.RegisterRateLimitPerMin, time.Minute)
	// Login is bucketed by the account being targeted, not only by client IP.
	//
	// Per-IP alone does not work here: public traffic arrives through an frp
	// raw-TCP tunnel, so every internet visitor shares one address and therefore
	// one bucket — measured: twelve different X-Forwarded-For values counted
	// together, 429 on the eleventh. That punishes everyone for one user's typos
	// while doing nothing extra to an attacker.
	//
	// Keying on the identifier makes the limit do what it was added for: slow
	// down guessing against a specific account, and contain the blast radius to
	// that account. It also survives the tunnel fix, at which point the IP-keyed
	// fallback below starts being meaningful again for anonymous floods.
	mux.Handle("POST "+cfg.APIV1Prefix+"/auth/login",
		loginLimiter.LimitKeyed(authHandler.Login, middleware.LoginIdentifierKey))
	mux.Handle("POST "+cfg.APIV1Prefix+"/auth/register", registerLimiter.Limit(authHandler.Register))
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/invites/{token}", membershipHandler.PreviewInvite)
	// Worker callback uses its own shared-secret token, not user JWT.
	mux.HandleFunc("POST "+cfg.APIV1Prefix+"/internal/worker-results", internalWorkerHandler.ReceiveResult)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/internal/poll-tasks", internalWorkerHandler.PollTasks)
	mux.HandleFunc("POST "+cfg.APIV1Prefix+"/internal/worker-progress", internalWorkerHandler.ReceiveProgress)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/internal/assistant-context/projects/{projectID}/search", internalAssistantContextHandler.SearchProjectText)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/internal/assistant-context/projects/{projectID}", internalAssistantContextHandler.GetProjectContext)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/internal/assistant-context/documents/{documentID}", internalAssistantContextHandler.GetDocumentContext)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/internal/assistant-context/handovers/{handoverID}", internalAssistantContextHandler.GetHandoverContext)
	mux.HandleFunc("GET "+cfg.APIV1Prefix+"/internal/assistant-assets/versions/{versionID}/download", internalAssistantContextHandler.DownloadVersionFile)

	// --- Protected routes (JWT required) ---
	mux.Handle("GET "+cfg.APIV1Prefix+"/auth/me", protect(authHandler.Me))
	mux.Handle("PATCH "+cfg.APIV1Prefix+"/auth/me", protect(authHandler.UpdateMe))
	mux.Handle("POST "+cfg.APIV1Prefix+"/auth/change-password", protect(authHandler.ChangePassword))
	mux.Handle("POST "+cfg.APIV1Prefix+"/auth/logout", protect(authHandler.Logout))
	mux.Handle("GET "+cfg.APIV1Prefix+"/users", protect(userHandler.List))
	mux.Handle("GET "+cfg.APIV1Prefix+"/inbox", protect(notificationHandler.Inbox))
	mux.Handle("GET "+cfg.APIV1Prefix+"/search", protect(searchHandler.Search))
	mux.Handle("GET "+cfg.APIV1Prefix+"/documents/{documentID}/text", protect(searchHandler.DocumentText))
	mux.Handle("GET "+cfg.APIV1Prefix+"/notifications", protect(notificationHandler.List))
	mux.Handle("GET "+cfg.APIV1Prefix+"/notifications/unread-count", protect(notificationHandler.UnreadCount))
	mux.Handle("POST "+cfg.APIV1Prefix+"/notifications/read-all", protect(notificationHandler.MarkAllRead))
	mux.Handle("POST "+cfg.APIV1Prefix+"/notifications/{notificationID}/read", protect(notificationHandler.MarkRead))
	mux.Handle("POST "+cfg.APIV1Prefix+"/assistant/conversations", protect(assistantHandler.CreateConversation))
	mux.Handle("GET "+cfg.APIV1Prefix+"/assistant/conversations", protect(assistantHandler.ListConversations))
	mux.Handle("GET "+cfg.APIV1Prefix+"/assistant/conversations/{conversationID}", protect(assistantHandler.GetConversation))
	mux.Handle("GET "+cfg.APIV1Prefix+"/assistant/conversations/{conversationID}/messages", protect(assistantHandler.ListConversationMessages))
	mux.Handle("PATCH "+cfg.APIV1Prefix+"/assistant/conversations/{conversationID}/archive", protect(assistantHandler.ArchiveConversation))
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
	mux.Handle("GET "+cfg.APIV1Prefix+"/projects/{projectID}/members", protect(membershipHandler.ListMembers))
	mux.Handle("POST "+cfg.APIV1Prefix+"/projects/{projectID}/members", protect(membershipHandler.AddMember))
	mux.Handle("PATCH "+cfg.APIV1Prefix+"/projects/{projectID}/members/{memberID}", protect(membershipHandler.UpdateMember))
	mux.Handle("DELETE "+cfg.APIV1Prefix+"/projects/{projectID}/members/{memberID}", protect(membershipHandler.RemoveMember))
	mux.Handle("GET "+cfg.APIV1Prefix+"/projects/{projectID}/invites", protect(membershipHandler.ListInvites))
	mux.Handle("POST "+cfg.APIV1Prefix+"/projects/{projectID}/invites", protect(membershipHandler.CreateInvite))
	mux.Handle("POST "+cfg.APIV1Prefix+"/projects/{projectID}/invites/{inviteID}/revoke", protect(membershipHandler.RevokeInvite))
	mux.Handle("POST "+cfg.APIV1Prefix+"/invites/{token}/accept", protect(membershipHandler.AcceptInvite))
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

	// --- Data Asset routes ---
	mux.Handle("GET "+cfg.APIV1Prefix+"/data-assets", protect(dataAssetHandler.List))
	mux.Handle("POST "+cfg.APIV1Prefix+"/data-assets", protect(dataAssetHandler.Upload))
	mux.Handle("GET "+cfg.APIV1Prefix+"/data-assets/{id}", protect(dataAssetHandler.Get))
	mux.Handle("PUT "+cfg.APIV1Prefix+"/data-assets/{id}", protect(dataAssetHandler.Update))
	mux.Handle("DELETE "+cfg.APIV1Prefix+"/data-assets/{id}", protect(dataAssetHandler.Delete))
	mux.Handle("GET "+cfg.APIV1Prefix+"/data-assets/{id}/download", protect(dataAssetHandler.Download))
	mux.Handle("GET "+cfg.APIV1Prefix+"/projects/{id}/data-folders", protect(dataAssetHandler.ListFolders))
	mux.Handle("POST "+cfg.APIV1Prefix+"/data-folders", protect(dataAssetHandler.CreateFolder))
	mux.Handle("DELETE "+cfg.APIV1Prefix+"/data-folders/{id}", protect(dataAssetHandler.DeleteFolder))
	mux.Handle("GET "+cfg.APIV1Prefix+"/handovers/{id}/data-items", protect(dataAssetHandler.ListHandoverDataItems))
	mux.Handle("PUT "+cfg.APIV1Prefix+"/handovers/{id}/data-items", protect(dataAssetHandler.UpdateHandoverDataItems))

	// --- Code routes ---
	mux.Handle("GET "+cfg.APIV1Prefix+"/code-repositories", protect(codeRepositoryHandler.List))
	mux.Handle("POST "+cfg.APIV1Prefix+"/code-repositories", protect(codeRepositoryHandler.Create))
	mux.Handle("GET "+cfg.APIV1Prefix+"/code-repositories/{id}", protect(codeRepositoryHandler.Get))
	mux.Handle("PATCH "+cfg.APIV1Prefix+"/code-repositories/{id}", protect(codeRepositoryHandler.Update))
	mux.Handle("GET "+cfg.APIV1Prefix+"/code-repositories/{id}/push-events", protect(codeRepositoryHandler.ListPushEvents))
	mux.Handle(cfg.APIV1Prefix+"/git/", http.StripPrefix("", http.HandlerFunc(codeRepositoryHandler.ServeGit)))

	mux.Handle("GET "+cfg.APIV1Prefix+"/dashboard/recent-flows", protect(dashboardHandler.RecentFlows))
	mux.Handle("GET "+cfg.APIV1Prefix+"/dashboard/risk-documents", protect(dashboardHandler.RiskDocuments))

	// --- Admin routes (admin role required) ---
	mux.Handle("POST "+cfg.APIV1Prefix+"/admin/team-spaces", protectAdmin(adminHandler.CreateTeamSpace))
	mux.Handle("POST "+cfg.APIV1Prefix+"/admin/projects", protectAdmin(adminHandler.CreateProject))
	mux.Handle("GET "+cfg.APIV1Prefix+"/admin/users", protectAdmin(adminHandler.ListAllUsers))
	mux.Handle("POST "+cfg.APIV1Prefix+"/admin/users", protectAdmin(adminHandler.CreateUser))
	mux.Handle("PATCH "+cfg.APIV1Prefix+"/admin/users/{userID}", protectAdmin(adminHandler.UpdateUser))
	mux.Handle("GET "+cfg.APIV1Prefix+"/admin/projects/{projectID}/members", protectAdmin(adminHandler.ListProjectMembers))
	mux.Handle("POST "+cfg.APIV1Prefix+"/admin/projects/{projectID}/members", protectAdmin(adminHandler.AddProjectMember))
	mux.Handle("PATCH "+cfg.APIV1Prefix+"/admin/projects/{projectID}/members/{memberID}", protectAdmin(adminHandler.UpdateProjectMember))
	mux.Handle("DELETE "+cfg.APIV1Prefix+"/admin/projects/{projectID}/members/{memberID}", protectAdmin(adminHandler.RemoveProjectMember))

	return middleware.Chain(
		mux,
		middleware.CORS(cfg.CORSAllowOrigins),
		middleware.RequestID,
		middleware.JSONContentType,
		middleware.LimitBody,
		middleware.AccessLog,
		// Innermost: it must see the request the mux annotates with r.Pattern
		// (outer middlewares replace the request via WithContext).
		metricsRegistry.Middleware,
	)
}
