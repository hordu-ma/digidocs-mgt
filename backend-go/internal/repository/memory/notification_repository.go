package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/service"
)

// NotificationRepository keeps notifications in process memory. The inbox and
// the context resolvers return empty results: memory mode has no relational
// business data to compute them from.
type NotificationRepository struct {
	mu    *sync.Mutex
	items *[]memoryNotification
}

type memoryNotification struct {
	userID string
	item   query.NotificationItem
}

func NewNotificationRepository() NotificationRepository {
	return NotificationRepository{mu: &sync.Mutex{}, items: &[]memoryNotification{}}
}

func (r NotificationRepository) CreateNotifications(_ context.Context, input command.NotificationInput) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, uid := range input.Recipients {
		*r.items = append(*r.items, memoryNotification{userID: uid, item: query.NotificationItem{
			ID: newID(), EventType: input.EventType, Title: input.Title, Body: input.Body, Link: input.Link,
			RelatedType: input.RelatedType, RelatedID: input.RelatedID,
			CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
		}})
	}
	return nil
}

func (r NotificationRepository) ListNotifications(_ context.Context, userID string, unreadOnly bool, page, pageSize int) ([]query.NotificationItem, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]query.NotificationItem, 0)
	for _, n := range *r.items {
		if n.userID == userID && (!unreadOnly || !n.item.Read) {
			out = append(out, n.item)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	total := len(out)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	if start >= total {
		return []query.NotificationItem{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return out[start:end], total, nil
}

func (r NotificationRepository) CountUnread(ctx context.Context, userID string) (int, error) {
	_, n, err := r.ListNotifications(ctx, userID, true, 1, 1)
	return n, err
}

func (r NotificationRepository) MarkRead(_ context.Context, userID, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range *r.items {
		n := &(*r.items)[i]
		if n.userID == userID && n.item.ID == id {
			n.item.Read = true
			return nil
		}
	}
	return service.ErrNotFound
}

func (r NotificationRepository) MarkAllRead(_ context.Context, userID string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for i := range *r.items {
		n := &(*r.items)[i]
		if n.userID == userID && !n.item.Read {
			n.item.Read = true
			count++
		}
	}
	return count, nil
}

func (r NotificationRepository) ListInbox(context.Context, string, bool) ([]query.InboxItem, error) {
	return []query.InboxItem{}, nil
}

func (r NotificationRepository) GetUserContacts(context.Context, []string) ([]query.UserContact, error) {
	return nil, nil
}

func (r NotificationRepository) GetDocumentBrief(context.Context, string) (*query.DocumentBrief, error) {
	return nil, service.ErrNotFound
}

func (r NotificationRepository) GetHandoverBrief(context.Context, string) (*query.HandoverBrief, error) {
	return nil, service.ErrNotFound
}

func (r NotificationRepository) GetLastTransferSender(context.Context, string) (string, error) {
	return "", nil
}

func (r NotificationRepository) GetAssistantRequestBrief(context.Context, string) (*query.AssistantRequestBrief, error) {
	return nil, service.ErrNotFound
}

func (r NotificationRepository) GetProjectManagers(context.Context, string) ([]string, string, error) {
	return nil, "", nil
}
