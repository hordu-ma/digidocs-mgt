package postgres

import "digidocs-mgt/backend-go/internal/domain/query"

// Aggregate endpoints (dashboard, audit log, handover list) have no single
// resource id to authorize against, so they narrow the result set in SQL to the
// projects the acting user belongs to.
//
// The predicates below are written as bind-parameter pairs so they can be
// appended to an existing WHERE clause without string interpolation:
//
//	$n   → bool, true when the actor is an admin (disables the narrowing)
//	$n+1 → text, the actor's user id
//
// A project column reference is the only piece that varies per table, so the
// snippets are provided per column rather than as one generic helper.

// projectScopeSQL builds the narrowing predicate for a query that can reach a
// project id through projectColumn (e.g. "d.project_id"). adminParam and
// actorParam are the placeholder tokens ("$7", "$8") assigned by the caller.
func projectScopeSQL(projectColumn string, adminParam string, actorParam string) string {
	return `(` + adminParam + ` OR EXISTS (
			SELECT 1
			FROM projects sp
			LEFT JOIN project_members spm
			  ON spm.project_id = sp.id
			 AND spm.user_id::text = ` + actorParam + `
			WHERE sp.id = ` + projectColumn + `
			  AND (sp.owner_id::text = ` + actorParam + ` OR spm.id IS NOT NULL)
		))`
}

// scopeArgs returns the two bind values matching projectScopeSQL's parameters.
func scopeArgs(scope query.ActorScope) (bool, string) {
	return scope.IsAdmin, scope.ActorID
}

// assistantRelatedProjectSQL resolves the project id behind an
// assistant_requests/assistant_suggestions row via its polymorphic
// related_type/related_id pair (document/handover/project), so the result can
// be fed into projectScopeSQL as a project id expression. Both columns must be
// qualified references the caller already has in scope (e.g. "ar.related_type",
// "ar.related_id").
func assistantRelatedProjectSQL(relatedTypeColumn string, relatedIDColumn string) string {
	return `(
		CASE ` + relatedTypeColumn + `
			WHEN 'document' THEN (SELECT d.project_id FROM documents d WHERE d.id::text = COALESCE(` + relatedIDColumn + `::text, ''))
			WHEN 'handover' THEN (SELECT h.project_id FROM graduation_handovers h WHERE h.id::text = COALESCE(` + relatedIDColumn + `::text, ''))
			WHEN 'project'  THEN (SELECT p.id FROM projects p WHERE p.id::text = COALESCE(` + relatedIDColumn + `::text, ''))
		END
	)`
}
