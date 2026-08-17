package handlers

import (
	"encoding/json"
	"net/http"

	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/transport/http/middleware"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type MembershipHandler struct {
	svc service.MembershipService
}

func NewMembershipHandler(svc service.MembershipService) MembershipHandler {
	return MembershipHandler{svc: svc}
}

func (h MembershipHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListMembers(r.Context(), middleware.UserIDFromContext(r.Context()), middleware.UserRoleFromContext(r.Context()), r.PathValue("projectID"))
	if err != nil {
		writeServiceError(w, err, "project not found", "failed to list members")
		return
	}
	response.WriteData(w, http.StatusOK, items)
}

func (h MembershipHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID      string `json:"user_id"`
		ProjectRole string `json:"project_role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	item, err := h.svc.AddMember(r.Context(), middleware.UserIDFromContext(r.Context()), middleware.UserRoleFromContext(r.Context()), r.PathValue("projectID"), body.UserID, body.ProjectRole)
	if err != nil {
		writeServiceError(w, err, "user not found", "failed to add member")
		return
	}
	response.WriteData(w, http.StatusCreated, item)
}

func (h MembershipHandler) UpdateMember(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProjectRole string `json:"project_role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	item, err := h.svc.UpdateMember(r.Context(), middleware.UserIDFromContext(r.Context()), middleware.UserRoleFromContext(r.Context()), r.PathValue("projectID"), r.PathValue("memberID"), body.ProjectRole)
	if err != nil {
		writeServiceError(w, err, "member not found", "failed to update member")
		return
	}
	response.WriteData(w, http.StatusOK, item)
}

func (h MembershipHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.RemoveMember(r.Context(), middleware.UserIDFromContext(r.Context()), middleware.UserRoleFromContext(r.Context()), r.PathValue("projectID"), r.PathValue("memberID")); err != nil {
		writeServiceError(w, err, "member not found", "failed to remove member")
		return
	}
	response.WriteData(w, http.StatusOK, map[string]bool{"success": true})
}

func (h MembershipHandler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	item, err := h.svc.CreateInvite(r.Context(), middleware.UserIDFromContext(r.Context()), middleware.UserRoleFromContext(r.Context()), r.PathValue("projectID"))
	if err != nil {
		writeServiceError(w, err, "project not found", "failed to create invite")
		return
	}
	response.WriteData(w, http.StatusCreated, item)
}

func (h MembershipHandler) ListInvites(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListInvites(r.Context(), middleware.UserIDFromContext(r.Context()), middleware.UserRoleFromContext(r.Context()), r.PathValue("projectID"))
	if err != nil {
		writeServiceError(w, err, "project not found", "failed to list invites")
		return
	}
	response.WriteData(w, http.StatusOK, items)
}

func (h MembershipHandler) RevokeInvite(w http.ResponseWriter, r *http.Request) {
	item, err := h.svc.RevokeInvite(r.Context(), middleware.UserIDFromContext(r.Context()), middleware.UserRoleFromContext(r.Context()), r.PathValue("projectID"), r.PathValue("inviteID"))
	if err != nil {
		writeServiceError(w, err, "invite not found", "failed to revoke invite")
		return
	}
	response.WriteData(w, http.StatusOK, item)
}

func (h MembershipHandler) PreviewInvite(w http.ResponseWriter, r *http.Request) {
	item, err := h.svc.PreviewInvite(r.Context(), r.PathValue("token"))
	if err != nil {
		writeServiceError(w, err, "invite not found", "failed to load invite")
		return
	}
	response.WriteData(w, http.StatusOK, item)
}

func (h MembershipHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	item, err := h.svc.AcceptInvite(r.Context(), middleware.UserIDFromContext(r.Context()), r.PathValue("token"))
	if err != nil {
		writeServiceError(w, err, "invite not found", "failed to accept invite")
		return
	}
	response.WriteData(w, http.StatusOK, item)
}
