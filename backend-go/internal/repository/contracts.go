package repository

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/auth"
	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/domain/task"
)

type TeamSpaceReader interface {
	ListTeamSpaces(ctx context.Context) ([]query.TeamSpaceSummary, error)
}

type UserReader interface {
	ListUsers(ctx context.Context) ([]query.UserOption, error)
}

type ProjectReader interface {
	ListProjects(ctx context.Context, teamSpaceID string) ([]query.ProjectSummary, error)
	GetFolderTree(ctx context.Context, projectID string) ([]query.FolderTreeNode, error)
}

type VersionReader interface {
	ListVersions(ctx context.Context, documentID string) ([]query.VersionItem, error)
	GetVersion(ctx context.Context, versionID string) (*query.VersionDetail, error)
}

type VersionWriter interface {
	CreateVersion(ctx context.Context, input command.VersionCreateInput) (map[string]any, error)
}

type VersionWorkflow interface {
	CreateUploadedVersion(ctx context.Context, input command.VersionCreateInput) (map[string]any, error)
}

type FlowReader interface {
	ListFlows(ctx context.Context, documentID string) ([]query.FlowItem, error)
}

type HandoverReader interface {
	ListHandovers(ctx context.Context) ([]query.HandoverItem, error)
	GetHandover(ctx context.Context, handoverID string) (*query.HandoverItem, error)
}

type AuditReader interface {
	ListAuditEvents(ctx context.Context, filter query.AuditEventFilter) ([]query.AuditEventItem, int, error)
	GetAuditSummary(ctx context.Context, projectID string) (query.AuditSummary, error)
}

type DashboardReader interface {
	GetOverview(ctx context.Context, projectID string) (query.DashboardOverview, error)
	ListRecentFlows(ctx context.Context, projectID string) ([]query.RecentFlowItem, error)
	ListRiskDocuments(ctx context.Context, projectID string) ([]query.RiskDocumentItem, error)
}

type ActionWriter interface {
	CreateFlowRecord(ctx context.Context, input command.FlowActionInput) (map[string]any, error)
	CreateHandover(ctx context.Context, input command.HandoverCreateInput) (map[string]any, error)
	UpdateHandoverItems(ctx context.Context, input command.HandoverItemUpdateInput) (map[string]any, error)
	ApplyHandover(ctx context.Context, input command.HandoverActionInput) (map[string]any, error)
}

type DocumentReader interface {
	ListDocuments(ctx context.Context, filter query.DocumentListFilter) ([]query.DocumentListItem, int, error)
	GetDocument(ctx context.Context, documentID string) (*query.DocumentDetail, error)
}

type DocumentWriter interface {
	CreateDocument(ctx context.Context, input command.DocumentCreateInput) (map[string]any, error)
	UpdateDocument(ctx context.Context, input command.DocumentUpdateInput) (map[string]any, error)
	DeleteDocument(ctx context.Context, input command.DocumentDeleteInput) error
	RestoreDocument(ctx context.Context, documentID string, actorID string) error
}

type UserAuthReader interface {
	FindUserByUsername(ctx context.Context, username string) (*auth.UserRecord, error)
}

type AssistantRepository interface {
	CreateAssistantRequest(ctx context.Context, message task.Message, actorID string) error
	CompleteAssistantRequest(ctx context.Context, result task.Result) error
	ListAssistantRequests(ctx context.Context, filter query.AssistantRequestFilter) ([]query.AssistantRequestItem, int, error)
	GetAssistantRequest(ctx context.Context, requestID string) (*query.AssistantRequestItem, error)
	GetLatestDocumentExtractedText(ctx context.Context, documentID string) (string, error)
	ListSuggestions(ctx context.Context, filter query.AssistantSuggestionFilter) ([]query.AssistantSuggestionItem, error)
	ConfirmSuggestion(ctx context.Context, suggestionID string, actorID string, note string) (map[string]any, error)
	DismissSuggestion(ctx context.Context, suggestionID string, actorID string, reason string) (map[string]any, error)
}
