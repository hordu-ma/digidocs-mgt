package memory

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/query"
)

type ProjectRepository struct{}

func NewProjectRepository() ProjectRepository {
	return ProjectRepository{}
}

func (r ProjectRepository) ListProjects(ctx context.Context, teamSpaceID, actorID, actorRole string) ([]query.ProjectSummary, error) {
	_ = ctx

	return []query.ProjectSummary{
		{
			ID:          "00000000-0000-0000-0000-000000000020",
			TeamSpaceID: pickTeamSpaceID(teamSpaceID),
			Name:        "课题A",
			Code:        "proj-a",
			Owner: query.UserSummary{
				ID:          "00000000-0000-0000-0000-000000000001",
				DisplayName: "李老师",
			},
		},
	}, nil
}

func (r ProjectRepository) GetFolderTree(ctx context.Context, projectID string) ([]query.FolderTreeNode, error) {
	_ = ctx
	_ = projectID

	return []query.FolderTreeNode{
		{
			ID:   "00000000-0000-0000-0000-000000000030",
			Name: "申报材料",
			Path: "/申报材料",
			Children: []query.FolderTreeNode{
				{
					ID:       "00000000-0000-0000-0000-000000000031",
					Name:     "历年版本",
					Path:     "/申报材料/历年版本",
					Children: []query.FolderTreeNode{},
				},
			},
		},
	}, nil
}

func pickTeamSpaceID(teamSpaceID string) string {
	if teamSpaceID != "" {
		return teamSpaceID
	}

	return "00000000-0000-0000-0000-000000000010"
}
