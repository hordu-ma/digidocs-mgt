package handlers

import (
	"net/http"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/transport/http/middleware"
)

// accessEvent fills the request-derived fields (actor, IP, user agent) of an
// access audit record.
func accessEvent(r *http.Request, action string, documentID string, versionID string, extra map[string]any) command.AuditEventInput {
	actorID, _ := actorFrom(r)
	return command.AuditEventInput{
		DocumentID:   documentID,
		VersionID:    versionID,
		UserID:       actorID,
		ActionType:   action,
		IPAddress:    middleware.ClientIP(r),
		TerminalInfo: r.UserAgent(),
		Extra:        extra,
	}
}
