package postgres

import (
	"context"
	"database/sql"
)

type PermissionRepository struct {
	db *sql.DB
}

func NewPermissionRepository(db *sql.DB) PermissionRepository {
	return PermissionRepository{db: db}
}

func (r PermissionRepository) CanCreateDocument(ctx context.Context, actorID string, actorRole string, projectID string) (bool, error) {
	if isAdmin(actorRole) {
		return true, nil
	}
	return r.isProjectContributor(ctx, actorID, projectID)
}

func (r PermissionRepository) CanUpdateDocument(ctx context.Context, actorID string, actorRole string, documentID string) (bool, error) {
	if isAdmin(actorRole) {
		return true, nil
	}
	return r.isProjectManagerOrCurrentOwner(ctx, actorID, documentID)
}

func (r PermissionRepository) CanDeleteDocument(ctx context.Context, actorID string, actorRole string, documentID string) (bool, error) {
	if isAdmin(actorRole) {
		return true, nil
	}
	return r.isProjectManagerForDocument(ctx, actorID, documentID)
}

func (r PermissionRepository) CanUploadVersion(ctx context.Context, actorID string, actorRole string, documentID string) (bool, error) {
	if isAdmin(actorRole) {
		return true, nil
	}
	return r.isProjectManagerOrCurrentOwner(ctx, actorID, documentID)
}

func (r PermissionRepository) CanFlowDocument(ctx context.Context, actorID string, actorRole string, documentID string, action string) (bool, error) {
	if isAdmin(actorRole) {
		return true, nil
	}
	switch action {
	case "finalize", "archive", "unarchive":
		return r.isProjectManagerForDocument(ctx, actorID, documentID)
	default:
		return r.isProjectManagerOrCurrentOwner(ctx, actorID, documentID)
	}
}

func (r PermissionRepository) CanCreateHandover(ctx context.Context, actorID string, actorRole string, projectID string) (bool, error) {
	if isAdmin(actorRole) {
		return true, nil
	}
	if projectID == "" {
		return false, nil
	}
	return r.isProjectManager(ctx, actorID, projectID)
}

func (r PermissionRepository) CanUpdateHandoverItems(ctx context.Context, actorID string, actorRole string, handoverID string) (bool, error) {
	if isAdmin(actorRole) {
		return true, nil
	}
	return r.isProjectManagerForHandover(ctx, actorID, handoverID)
}

func (r PermissionRepository) CanApplyHandover(ctx context.Context, actorID string, actorRole string, handoverID string, action string) (bool, error) {
	if isAdmin(actorRole) {
		return true, nil
	}
	if action == "confirm" {
		var ok bool
		if err := r.db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM graduation_handovers
				WHERE id::text = $1
				  AND receiver_user_id::text = $2
			)
		`, handoverID, actorID).Scan(&ok); err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return r.isProjectManagerForHandover(ctx, actorID, handoverID)
}

func (r PermissionRepository) isProjectContributor(ctx context.Context, actorID string, projectID string) (bool, error) {
	return r.exists(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM projects p
			LEFT JOIN project_members pm
			  ON pm.project_id = p.id
			 AND pm.user_id::text = $2
			 AND pm.project_role IN ('owner', 'manager', 'contributor')
			WHERE p.id::text = $1
			  AND (p.owner_id::text = $2 OR pm.id IS NOT NULL)
		)
	`, projectID, actorID)
}

func (r PermissionRepository) isProjectManager(ctx context.Context, actorID string, projectID string) (bool, error) {
	return r.exists(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM projects p
			LEFT JOIN project_members pm
			  ON pm.project_id = p.id
			 AND pm.user_id::text = $2
			 AND pm.project_role IN ('owner', 'manager')
			WHERE p.id::text = $1
			  AND (p.owner_id::text = $2 OR pm.id IS NOT NULL)
		)
	`, projectID, actorID)
}

func (r PermissionRepository) isProjectManagerForDocument(ctx context.Context, actorID string, documentID string) (bool, error) {
	return r.exists(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM documents d
			JOIN projects p ON p.id = d.project_id
			LEFT JOIN project_members pm
			  ON pm.project_id = p.id
			 AND pm.user_id::text = $2
			 AND pm.project_role IN ('owner', 'manager')
			WHERE d.id::text = $1
			  AND d.is_deleted = false
			  AND (p.owner_id::text = $2 OR pm.id IS NOT NULL)
		)
	`, documentID, actorID)
}

func (r PermissionRepository) isProjectManagerOrCurrentOwner(ctx context.Context, actorID string, documentID string) (bool, error) {
	return r.exists(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM documents d
			JOIN projects p ON p.id = d.project_id
			LEFT JOIN project_members pm
			  ON pm.project_id = p.id
			 AND pm.user_id::text = $2
			 AND pm.project_role IN ('owner', 'manager')
			WHERE d.id::text = $1
			  AND d.is_deleted = false
			  AND (d.current_owner_id::text = $2 OR p.owner_id::text = $2 OR pm.id IS NOT NULL)
		)
	`, documentID, actorID)
}

func (r PermissionRepository) isProjectManagerForHandover(ctx context.Context, actorID string, handoverID string) (bool, error) {
	return r.exists(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM graduation_handovers h
			JOIN projects p ON p.id = h.project_id
			LEFT JOIN project_members pm
			  ON pm.project_id = p.id
			 AND pm.user_id::text = $2
			 AND pm.project_role IN ('owner', 'manager')
			WHERE h.id::text = $1
			  AND (p.owner_id::text = $2 OR pm.id IS NOT NULL)
		)
	`, handoverID, actorID)
}

func (r PermissionRepository) exists(ctx context.Context, query string, args ...any) (bool, error) {
	var ok bool
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&ok); err != nil {
		return false, err
	}
	return ok, nil
}

func isAdmin(role string) bool {
	return role == "admin"
}

func (r PermissionRepository) CanUploadDataAsset(ctx context.Context, actorID string, actorRole string, projectID string) (bool, error) {
	if isAdmin(actorRole) {
		return true, nil
	}
	return r.isProjectContributor(ctx, actorID, projectID)
}

func (r PermissionRepository) CanManageDataAsset(ctx context.Context, actorID string, actorRole string, dataAssetID string) (bool, error) {
	if isAdmin(actorRole) {
		return true, nil
	}
	// Creator OR project manager for the project the asset belongs to.
	return r.exists(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM data_assets da
			JOIN projects p ON p.id = da.project_id
			LEFT JOIN project_members pm
			  ON pm.project_id = p.id
			 AND pm.user_id::text = $2
			 AND pm.project_role IN ('owner', 'manager')
			WHERE da.id::text = $1
			  AND da.is_deleted = false
			  AND (da.created_by::text = $2 OR p.owner_id::text = $2 OR pm.id IS NOT NULL)
		)
	`, dataAssetID, actorID)
}

func (r PermissionRepository) CanViewDocument(ctx context.Context, actorID string, actorRole string, documentID string) (bool, error) {
	if isAdmin(actorRole) {
		return true, nil
	}
	return r.isProjectMemberForDocument(ctx, actorID, documentID)
}

// CanViewVersion resolves the version's owning document and reuses the document
// read rule. Version ids leak through the audit log, so this must be checked on
// every version read path (list/get/download/preview), not just on the document.
func (r PermissionRepository) CanViewVersion(ctx context.Context, actorID string, actorRole string, versionID string) (bool, error) {
	if isAdmin(actorRole) {
		return true, nil
	}
	return r.exists(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM document_versions dv
			JOIN documents d ON d.id = dv.document_id
			JOIN projects p ON p.id = d.project_id
			LEFT JOIN project_members pm
			  ON pm.project_id = p.id
			 AND pm.user_id::text = $2
			WHERE dv.id::text = $1
			  AND d.is_deleted = false
			  AND (d.current_owner_id::text = $2 OR p.owner_id::text = $2 OR pm.id IS NOT NULL)
		)
	`, versionID, actorID)
}

// CanViewHandover allows the two parties named on the handover (the departing
// member and the receiver) plus any member of the owning project.
func (r PermissionRepository) CanViewHandover(ctx context.Context, actorID string, actorRole string, handoverID string) (bool, error) {
	if isAdmin(actorRole) {
		return true, nil
	}
	return r.exists(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM graduation_handovers h
			LEFT JOIN projects p ON p.id = h.project_id
			LEFT JOIN project_members pm
			  ON pm.project_id = p.id
			 AND pm.user_id::text = $2
			WHERE h.id::text = $1
			  AND (
				h.target_user_id::text = $2
				OR h.receiver_user_id::text = $2
				OR p.owner_id::text = $2
				OR pm.id IS NOT NULL
			  )
		)
	`, handoverID, actorID)
}

func (r PermissionRepository) CanViewProject(ctx context.Context, actorID string, actorRole string, projectID string) (bool, error) {
	if isAdmin(actorRole) {
		return true, nil
	}
	if projectID == "" {
		return false, nil
	}
	return r.isProjectMember(ctx, actorID, projectID)
}

func (r PermissionRepository) CanViewDataAsset(ctx context.Context, actorID string, actorRole string, dataAssetID string) (bool, error) {
	if isAdmin(actorRole) {
		return true, nil
	}
	return r.isProjectMemberForDataAsset(ctx, actorID, dataAssetID)
}

// isProjectMemberForDocument allows any project role (including viewer),
// unlike isProjectManagerOrCurrentOwner which is restricted to write access.
func (r PermissionRepository) isProjectMemberForDocument(ctx context.Context, actorID string, documentID string) (bool, error) {
	return r.exists(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM documents d
			JOIN projects p ON p.id = d.project_id
			LEFT JOIN project_members pm
			  ON pm.project_id = p.id
			 AND pm.user_id::text = $2
			WHERE d.id::text = $1
			  AND d.is_deleted = false
			  AND (d.current_owner_id::text = $2 OR p.owner_id::text = $2 OR pm.id IS NOT NULL)
		)
	`, documentID, actorID)
}

func (r PermissionRepository) isProjectMemberForDataAsset(ctx context.Context, actorID string, dataAssetID string) (bool, error) {
	return r.exists(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM data_assets da
			JOIN projects p ON p.id = da.project_id
			LEFT JOIN project_members pm
			  ON pm.project_id = p.id
			 AND pm.user_id::text = $2
			WHERE da.id::text = $1
			  AND da.is_deleted = false
			  AND (da.created_by::text = $2 OR p.owner_id::text = $2 OR pm.id IS NOT NULL)
		)
	`, dataAssetID, actorID)
}

func (r PermissionRepository) CanViewCodeRepository(ctx context.Context, actorID string, actorRole string, repositoryID string) (bool, error) {
	if isAdmin(actorRole) {
		return true, nil
	}
	return r.exists(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM code_repositories cr
			JOIN projects p ON p.id = cr.project_id
			LEFT JOIN project_members pm
			  ON pm.project_id = p.id
			 AND pm.user_id::text = $2
			WHERE cr.id::text = $1
			  AND cr.is_deleted = false
			  AND (p.owner_id::text = $2 OR pm.id IS NOT NULL)
		)
	`, repositoryID, actorID)
}

func (r PermissionRepository) CanCreateCodeRepository(ctx context.Context, actorID string, actorRole string, projectID string) (bool, error) {
	if isAdmin(actorRole) {
		return true, nil
	}
	return r.isProjectManager(ctx, actorID, projectID)
}

func (r PermissionRepository) CanManageCodeRepository(ctx context.Context, actorID string, actorRole string, repositoryID string) (bool, error) {
	if isAdmin(actorRole) {
		return true, nil
	}
	return r.exists(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM code_repositories cr
			JOIN projects p ON p.id = cr.project_id
			LEFT JOIN project_members pm
			  ON pm.project_id = p.id
			 AND pm.user_id::text = $2
			 AND pm.project_role IN ('owner', 'manager')
			WHERE cr.id::text = $1
			  AND cr.is_deleted = false
			  AND (p.owner_id::text = $2 OR pm.id IS NOT NULL)
		)
	`, repositoryID, actorID)
}

func (r PermissionRepository) CanViewProjectMembers(ctx context.Context, actorID string, actorRole string, projectID string) (bool, error) {
	if isAdmin(actorRole) {
		return true, nil
	}
	return r.isProjectMember(ctx, actorID, projectID)
}

func (r PermissionRepository) CanManageProjectMembers(ctx context.Context, actorID string, actorRole string, projectID string) (bool, error) {
	if isAdmin(actorRole) {
		return true, nil
	}
	return r.isProjectManager(ctx, actorID, projectID)
}

func (r PermissionRepository) isProjectMember(ctx context.Context, actorID string, projectID string) (bool, error) {
	return r.exists(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM projects p
			LEFT JOIN project_members pm
			  ON pm.project_id = p.id
			 AND pm.user_id::text = $2
			WHERE p.id::text = $1
			  AND (p.owner_id::text = $2 OR pm.id IS NOT NULL)
		)
	`, projectID, actorID)
}

// CanViewAssistantRequest allows the user who created the request plus any
// member of the project that owns the request's related resource (a document,
// handover, or project). Assistant requests are the AI-facing counterpart of
// the same cross-project leak fixed for documents/versions/handovers: the
// related_id/related_type pair leaks freely through list/poll endpoints, so
// this must be checked on every read path, not just the one that was reported.
func (r PermissionRepository) CanViewAssistantRequest(ctx context.Context, actorID string, actorRole string, requestID string) (bool, error) {
	if isAdmin(actorRole) {
		return true, nil
	}
	return r.exists(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM assistant_requests ar
			WHERE ar.id::text = $1
			  AND (
				ar.created_by::text = $2
				OR EXISTS (
					SELECT 1
					FROM projects p
					LEFT JOIN project_members pm
					  ON pm.project_id = p.id
					 AND pm.user_id::text = $2
					WHERE p.id = `+assistantRelatedProjectSQL("ar.related_type", "ar.related_id")+`
					  AND (p.owner_id::text = $2 OR pm.id IS NOT NULL)
				)
			  )
		)
	`, requestID, actorID)
}

// CanViewAssistantSuggestion mirrors CanViewAssistantRequest, but suggestions
// have no creator (they are generated by the worker from a completed task), so
// visibility is purely a function of the related resource's project.
func (r PermissionRepository) CanViewAssistantSuggestion(ctx context.Context, actorID string, actorRole string, suggestionID string) (bool, error) {
	if isAdmin(actorRole) {
		return true, nil
	}
	return r.exists(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM assistant_suggestions s
			WHERE s.id::text = $1
			  AND EXISTS (
				SELECT 1
				FROM projects p
				LEFT JOIN project_members pm
				  ON pm.project_id = p.id
				 AND pm.user_id::text = $2
				WHERE p.id = `+assistantRelatedProjectSQL("s.related_type", "s.related_id")+`
				  AND (p.owner_id::text = $2 OR pm.id IS NOT NULL)
			  )
		)
	`, suggestionID, actorID)
}

// CanViewConversation matches the actor filter ListConversations already
// applies for non-admins: only the conversation's creator (or an admin) may
// read it. GetConversation/ListConversationMessages used to skip this check
// entirely, which let anyone who obtained a conversation_id (e.g. via a leaked
// assistant_requests row) bypass the list-level filter.
func (r PermissionRepository) CanViewConversation(ctx context.Context, actorID string, actorRole string, conversationID string) (bool, error) {
	if isAdmin(actorRole) {
		return true, nil
	}
	return r.exists(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM assistant_conversations c
			WHERE c.id::text = $1
			  AND COALESCE(c.created_by::text, '') = $2
		)
	`, conversationID, actorID)
}

func (r PermissionRepository) CanPushCodeRepository(ctx context.Context, actorID string, actorRole string, repositoryID string) (bool, error) {
	if isAdmin(actorRole) {
		return true, nil
	}
	return r.exists(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM code_repositories cr
			JOIN projects p ON p.id = cr.project_id
			LEFT JOIN project_members pm
			  ON pm.project_id = p.id
			 AND pm.user_id::text = $2
			 AND pm.project_role IN ('owner', 'manager', 'contributor')
			WHERE cr.id::text = $1
			  AND cr.is_deleted = false
			  AND (p.owner_id::text = $2 OR pm.id IS NOT NULL)
		)
	`, repositoryID, actorID)
}
