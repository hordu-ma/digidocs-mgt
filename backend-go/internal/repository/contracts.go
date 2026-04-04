package repository

import (
	"context"

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

type FlowReader interface {
	ListFlows(ctx context.Context, documentID string) ([]query.FlowItem, error)
}

type HandoverReader interface {
	ListHandovers(ctx context.Context) ([]query.HandoverItem, error)
	GetHandover(ctx context.Context, handoverID string) (*query.HandoverItem, error)
}

type DocumentReader interface {
	ListDocuments(ctx context.Context, filter query.DocumentListFilter) ([]query.DocumentListItem, int, error)
	GetDocument(ctx context.Context, documentID string) (*query.DocumentDetail, error)
}
