package service

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/repository"
)

type QueryService struct {
	teamSpaces repository.TeamSpaceReader
	projects   repository.ProjectReader
	documents  repository.DocumentReader
}

func NewQueryService(
	teamSpaces repository.TeamSpaceReader,
	projects repository.ProjectReader,
	documents repository.DocumentReader,
) QueryService {
	return QueryService{
		teamSpaces: teamSpaces,
		projects:   projects,
		documents:  documents,
	}
}

func (s QueryService) ListTeamSpaces(ctx context.Context) ([]query.TeamSpaceSummary, error) {
	return s.teamSpaces.ListTeamSpaces(ctx)
}

func (s QueryService) ListProjects(ctx context.Context, teamSpaceID string) ([]query.ProjectSummary, error) {
	return s.projects.ListProjects(ctx, teamSpaceID)
}

func (s QueryService) GetFolderTree(ctx context.Context, projectID string) ([]query.FolderTreeNode, error) {
	return s.projects.GetFolderTree(ctx, projectID)
}

func (s QueryService) ListDocuments(
	ctx context.Context,
	filter query.DocumentListFilter,
) ([]query.DocumentListItem, int, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	return s.documents.ListDocuments(ctx, filter)
}

func (s QueryService) GetDocument(ctx context.Context, documentID string) (*query.DocumentDetail, error) {
	return s.documents.GetDocument(ctx, documentID)
}
