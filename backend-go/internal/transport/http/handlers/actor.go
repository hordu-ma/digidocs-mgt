package handlers

import (
	"net/http"

	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/transport/http/middleware"
)

// actorFrom returns the authenticated caller's id and role. Every handler behind
// the auth middleware should read its actor through this helper rather than
// touching the context directly, so that "which handlers authorize" is greppable.
func actorFrom(r *http.Request) (string, string) {
	return middleware.UserIDFromContext(r.Context()), middleware.UserRoleFromContext(r.Context())
}

// scopeFrom builds the project narrowing used by aggregate endpoints that have
// no single resource id to authorize against.
func scopeFrom(r *http.Request) query.ActorScope {
	actorID, actorRole := actorFrom(r)
	return query.ActorScope{ActorID: actorID, IsAdmin: actorRole == "admin"}
}
