package service

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/repository"
)

type QueryService struct {
	teamSpaces repository.TeamSpaceReader
	users      repository.UserReader
	projects   repository.ProjectReader
}

func NewQueryService(
	teamSpaces repository.TeamSpaceReader,
	users repository.UserReader,
	projects repository.ProjectReader,
) QueryService {
	return QueryService{
		teamSpaces: teamSpaces,
		users:      users,
		projects:   projects,
	}
}

func (s QueryService) ListTeamSpaces(ctx context.Context) ([]query.TeamSpaceSummary, error) {
	return s.teamSpaces.ListTeamSpaces(ctx)
}

func (s QueryService) ListUsers(ctx context.Context) ([]query.UserOption, error) {
	return s.users.ListUsers(ctx)
}

func (s QueryService) ListProjects(ctx context.Context, teamSpaceID string) ([]query.ProjectSummary, error) {
	return s.projects.ListProjects(ctx, teamSpaceID)
}

func (s QueryService) GetFolderTree(ctx context.Context, projectID string) ([]query.FolderTreeNode, error) {
	return s.projects.GetFolderTree(ctx, projectID)
}
