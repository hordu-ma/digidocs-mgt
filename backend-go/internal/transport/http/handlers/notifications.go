package handlers

import (
	"net/http"

	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type NotificationHandler struct {
	svc service.NotificationService
}

func NewNotificationHandler(svc service.NotificationService) NotificationHandler {
	return NotificationHandler{svc: svc}
}

// Inbox  GET /inbox — the caller's open tasks, computed live.
func (h NotificationHandler) Inbox(w http.ResponseWriter, r *http.Request) {
	actorID, actorRole := actorFrom(r)
	items, err := h.svc.Inbox(r.Context(), actorID, actorRole)
	if err != nil {
		writeServiceError(w, err, "inbox not found", "failed to load inbox")
		return
	}
	counts := map[string]int{}
	for _, item := range items {
		counts[item.Kind]++
	}
	response.WriteWithMeta(w, http.StatusOK, items, map[string]any{"total": len(items), "counts": counts})
}

// List  GET /notifications?unread_only=true&page=&page_size=
func (h NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	actorID, _ := actorFrom(r)
	page := parseIntOrDefault(r.URL.Query().Get("page"), 1)
	pageSize := parsePageSize(r.URL.Query().Get("page_size"))
	unreadOnly := r.URL.Query().Get("unread_only") == "true"
	items, total, err := h.svc.List(r.Context(), actorID, unreadOnly, page, pageSize)
	if err != nil {
		writeServiceError(w, err, "notifications not found", "failed to list notifications")
		return
	}
	unread, err := h.svc.UnreadCount(r.Context(), actorID)
	if err != nil {
		writeServiceError(w, err, "notifications not found", "failed to count notifications")
		return
	}
	response.WriteWithMeta(w, http.StatusOK, items, map[string]any{
		"pagination": query.PaginationMeta{Page: page, PageSize: pageSize, Total: total},
		"unread":     unread,
	})
}

// UnreadCount  GET /notifications/unread-count — polled by the top bar.
func (h NotificationHandler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	actorID, _ := actorFrom(r)
	n, err := h.svc.UnreadCount(r.Context(), actorID)
	if err != nil {
		writeServiceError(w, err, "notifications not found", "failed to count notifications")
		return
	}
	response.WriteData(w, http.StatusOK, map[string]int{"unread": n})
}

// MarkRead  POST /notifications/{id}/read
func (h NotificationHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	actorID, _ := actorFrom(r)
	if err := h.svc.MarkRead(r.Context(), actorID, r.PathValue("notificationID")); err != nil {
		writeServiceError(w, err, "notification not found", "failed to mark notification read")
		return
	}
	response.WriteData(w, http.StatusOK, map[string]bool{"read": true})
}

// MarkAllRead  POST /notifications/read-all
func (h NotificationHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	actorID, _ := actorFrom(r)
	n, err := h.svc.MarkAllRead(r.Context(), actorID)
	if err != nil {
		writeServiceError(w, err, "notifications not found", "failed to mark notifications read")
		return
	}
	response.WriteData(w, http.StatusOK, map[string]int{"updated": n})
}
