package postgres

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/query"
)

type ProjectRepository struct {
	db DBTX
}

func NewProjectRepository(db DBTX) ProjectRepository {
	return ProjectRepository{db: db}
}

func (r ProjectRepository) ListProjects(ctx context.Context, teamSpaceID string) ([]query.ProjectSummary, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`
		SELECT
			p.id::text,
			p.team_space_id::text,
			p.name,
			p.code,
			u.id::text,
			u.display_name
		FROM projects p
		JOIN users u ON u.id = p.owner_id
		WHERE ($1 = '' OR p.team_space_id::text = $1)
		ORDER BY p.name ASC
		`,
		teamSpaceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]query.ProjectSummary, 0)
	for rows.Next() {
		var item query.ProjectSummary
		if err := rows.Scan(
			&item.ID,
			&item.TeamSpaceID,
			&item.Name,
			&item.Code,
			&item.Owner.ID,
			&item.Owner.DisplayName,
		); err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, rows.Err()
}

func (r ProjectRepository) GetFolderTree(ctx context.Context, projectID string) ([]query.FolderTreeNode, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`
		SELECT
			id::text,
			COALESCE(parent_id::text, ''),
			name,
			path
		FROM folders
		WHERE project_id::text = $1
		ORDER BY path ASC
		`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodes := make(map[string]*query.FolderTreeNode)
	parentByChild := make(map[string]string)
	order := make([]string, 0)

	for rows.Next() {
		var id string
		var parentID string
		var name string
		var path string

		if err := rows.Scan(&id, &parentID, &name, &path); err != nil {
			return nil, err
		}

		nodes[id] = &query.FolderTreeNode{
			ID:       id,
			Name:     name,
			Path:     path,
			Children: []query.FolderTreeNode{},
		}
		parentByChild[id] = parentID
		order = append(order, id)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	roots := make([]query.FolderTreeNode, 0)
	for _, id := range order {
		node := nodes[id]
		parentID := parentByChild[id]
		if parentID == "" {
			roots = append(roots, *node)
			continue
		}

		parent := nodes[parentID]
		if parent == nil {
			roots = append(roots, *node)
			continue
		}

		parent.Children = append(parent.Children, *node)
	}

	return rebuildRoots(roots, nodes), nil
}

func rebuildRoots(roots []query.FolderTreeNode, nodes map[string]*query.FolderTreeNode) []query.FolderTreeNode {
	rebuilt := make([]query.FolderTreeNode, 0, len(roots))
	for _, root := range roots {
		rebuilt = append(rebuilt, cloneNode(root.ID, nodes))
	}

	return rebuilt
}

func cloneNode(nodeID string, nodes map[string]*query.FolderTreeNode) query.FolderTreeNode {
	node := nodes[nodeID]
	if node == nil {
		return query.FolderTreeNode{}
	}

	cloned := query.FolderTreeNode{
		ID:       node.ID,
		Name:     node.Name,
		Path:     node.Path,
		Children: make([]query.FolderTreeNode, 0, len(node.Children)),
	}

	for _, child := range node.Children {
		cloned.Children = append(cloned.Children, cloneNode(child.ID, nodes))
	}

	return cloned
}
