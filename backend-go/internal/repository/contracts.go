package repository

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/auth"
	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
)

type TeamSpaceReader interface {
	ListTeamSpaces(ctx context.Context) ([]query.TeamSpaceSummary, error)
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
}

type UserAuthReader interface {
	FindUserByUsername(ctx context.Context, username string) (*auth.UserRecord, error)
}
