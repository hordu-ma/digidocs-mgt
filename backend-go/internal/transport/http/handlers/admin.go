package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/transport/http/middleware"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type AdminHandler struct {
	adminService service.AdminService
}

func NewAdminHandler(adminService service.AdminService) AdminHandler {
	return AdminHandler{adminService: adminService}
}

// --- Team Spaces ---

func (h AdminHandler) CreateTeamSpace(w http.ResponseWriter, r *http.Request) {
	var input service.CreateTeamSpaceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	input.ActorID = middleware.UserIDFromContext(r.Context())

	item, err := h.adminService.CreateTeamSpace(r.Context(), input)
	if err != nil {
		writeAdminError(w, err)
		return
	}

	response.WriteData(w, http.StatusCreated, item)
}

// --- Projects ---

func (h AdminHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	var input service.CreateProjectInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	item, err := h.adminService.CreateProject(r.Context(), input)
	if err != nil {
		writeAdminError(w, err)
		return
	}

	response.WriteData(w, http.StatusCreated, item)
}

// --- Users ---

func (h AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var input service.CreateUserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	item, err := h.adminService.CreateUser(r.Context(), input)
	if err != nil {
		writeAdminError(w, err)
		return
	}

	response.WriteData(w, http.StatusCreated, item)
}

func (h AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userID")
	if userID == "" {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "missing userID")
		return
	}

	var input service.UpdateUserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	input.UserID = userID

	item, err := h.adminService.UpdateUser(r.Context(), input)
	if err != nil {
		writeAdminError(w, err)
		return
	}

	response.WriteData(w, http.StatusOK, item)
}

func (h AdminHandler) ListAllUsers(w http.ResponseWriter, r *http.Request) {
	items, err := h.adminService.ListAllUsers(r.Context())
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to list users")
		return
	}

	response.WriteData(w, http.StatusOK, items)
}

// --- Project Members ---

func (h AdminHandler) ListProjectMembers(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	if projectID == "" {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "missing projectID")
		return
	}

	items, err := h.adminService.ListProjectMembers(r.Context(), projectID)
	if err != nil {
		writeAdminError(w, err)
		return
	}

	response.WriteData(w, http.StatusOK, items)
}

func (h AdminHandler) AddProjectMember(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	if projectID == "" {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "missing projectID")
		return
	}

	var input service.AddProjectMemberInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	input.ProjectID = projectID

	item, err := h.adminService.AddProjectMember(r.Context(), input)
	if err != nil {
		writeAdminError(w, err)
		return
	}

	response.WriteData(w, http.StatusCreated, item)
}

func (h AdminHandler) UpdateProjectMember(w http.ResponseWriter, r *http.Request) {
	memberID := r.PathValue("memberID")
	if memberID == "" {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "missing memberID")
		return
	}

	var body struct {
		ProjectRole string `json:"project_role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	item, err := h.adminService.UpdateProjectMember(r.Context(), memberID, body.ProjectRole)
	if err != nil {
		writeAdminError(w, err)
		return
	}

	response.WriteData(w, http.StatusOK, item)
}

func (h AdminHandler) RemoveProjectMember(w http.ResponseWriter, r *http.Request) {
	memberID := r.PathValue("memberID")
	if memberID == "" {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "missing memberID")
		return
	}

	if err := h.adminService.RemoveProjectMember(r.Context(), memberID); err != nil {
		writeAdminError(w, err)
		return
	}

	response.WriteData(w, http.StatusOK, map[string]bool{"success": true})
}

// writeAdminError maps service errors to HTTP responses.
func writeAdminError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrValidation):
		response.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
	case errors.Is(err, service.ErrConflict):
		response.WriteError(w, http.StatusConflict, "conflict", "resource already exists")
	case errors.Is(err, service.ErrNotFound):
		response.WriteError(w, http.StatusNotFound, "not_found", "resource not found")
	case errors.Is(err, service.ErrForbidden):
		response.WriteError(w, http.StatusForbidden, "forbidden", "access denied")
	default:
		log.Printf("[admin] unhandled error: %v", err)
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "operation failed")
	}
}
