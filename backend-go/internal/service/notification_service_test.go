package service

import (
	"context"
	"errors"
	"testing"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/domain/task"
	"digidocs-mgt/backend-go/internal/notify"
)

type fakeNotificationRepo struct {
	created   []command.NotificationInput
	createErr error
	doc       *query.DocumentBrief
	handover  *query.HandoverBrief
	sender    string
	request   *query.AssistantRequestBrief
}

func (f *fakeNotificationRepo) CreateNotifications(_ context.Context, in command.NotificationInput) error {
	f.created = append(f.created, in)
	return f.createErr
}
func (f *fakeNotificationRepo) ListNotifications(context.Context, string, bool, int, int) ([]query.NotificationItem, int, error) {
	return nil, 0, nil
}
func (f *fakeNotificationRepo) CountUnread(context.Context, string) (int, error) { return 0, nil }
func (f *fakeNotificationRepo) MarkRead(context.Context, string, string) error   { return nil }
func (f *fakeNotificationRepo) MarkAllRead(context.Context, string) (int, error) { return 0, nil }
func (f *fakeNotificationRepo) ListInbox(context.Context, string, bool) ([]query.InboxItem, error) {
	return nil, nil
}
func (f *fakeNotificationRepo) GetUserContacts(_ context.Context, ids []string) ([]query.UserContact, error) {
	out := make([]query.UserContact, 0, len(ids))
	for _, id := range ids {
		out = append(out, query.UserContact{ID: id, DisplayName: "name-" + id, Email: id + "@x"})
	}
	return out, nil
}
func (f *fakeNotificationRepo) GetDocumentBrief(context.Context, string) (*query.DocumentBrief, error) {
	if f.doc == nil {
		return nil, ErrNotFound
	}
	return f.doc, nil
}
func (f *fakeNotificationRepo) GetHandoverBrief(context.Context, string) (*query.HandoverBrief, error) {
	if f.handover == nil {
		return nil, ErrNotFound
	}
	return f.handover, nil
}
func (f *fakeNotificationRepo) GetLastTransferSender(context.Context, string) (string, error) {
	return f.sender, nil
}
func (f *fakeNotificationRepo) GetAssistantRequestBrief(context.Context, string) (*query.AssistantRequestBrief, error) {
	if f.request == nil {
		return nil, ErrNotFound
	}
	return f.request, nil
}
func (f *fakeNotificationRepo) GetProjectManagers(context.Context, string) ([]string, string, error) {
	return []string{"lead", "actor"}, "课题A", nil
}

type fakeChannel struct{ got []notify.Message }

func (c *fakeChannel) Name() string { return "fake" }
func (c *fakeChannel) Deliver(_ context.Context, m notify.Message) error {
	c.got = append(c.got, m)
	return errors.New("delivery failures are logged only")
}

func newTestNotifier(repo *fakeNotificationRepo, ch *fakeChannel) NotificationService {
	s := NewNotificationService(repo, "https://docs.example.org/", ch)
	s.async = false
	return s
}

func TestNotifyExcludesActorAndDuplicates(t *testing.T) {
	repo := &fakeNotificationRepo{}
	s := newTestNotifier(repo, &fakeChannel{})
	s.Notify(context.Background(), command.NotificationInput{Recipients: []string{"a", "actor", "a", " "}, ActorID: "actor", Title: "t"})
	if len(repo.created) != 1 || len(repo.created[0].Recipients) != 1 || repo.created[0].Recipients[0] != "a" {
		t.Fatalf("got %#v", repo.created)
	}
	s.Notify(context.Background(), command.NotificationInput{Recipients: []string{"actor"}, ActorID: "actor"})
	if len(repo.created) != 1 {
		t.Fatal("self-only notification must not be stored")
	}
}

func TestTransferNotifiesReceiverExternallyWithAbsoluteLink(t *testing.T) {
	repo := &fakeNotificationRepo{doc: &query.DocumentBrief{ID: "d1", Title: "开题报告"}}
	ch := &fakeChannel{}
	s := newTestNotifier(repo, ch)
	s.OnFlowAction(context.Background(), command.FlowActionInput{DocumentID: "d1", Action: "transfer", ToUserID: "bob", ActorID: "amy", Note: "请周五前完成"})
	if len(repo.created) != 1 || repo.created[0].Recipients[0] != "bob" || repo.created[0].EventType != "document_transfer" {
		t.Fatalf("got %#v", repo.created)
	}
	if len(ch.got) != 1 || ch.got[0].Link != "https://docs.example.org/documents/d1" || ch.got[0].Recipients[0].Email != "bob@x" {
		t.Fatalf("external message wrong: %#v", ch.got)
	}
}

func TestAcceptTransferNotifiesSenderInAppOnly(t *testing.T) {
	repo := &fakeNotificationRepo{doc: &query.DocumentBrief{ID: "d1", Title: "开题报告"}, sender: "amy"}
	ch := &fakeChannel{}
	newTestNotifier(repo, ch).OnFlowAction(context.Background(), command.FlowActionInput{DocumentID: "d1", Action: "accept_transfer", ActorID: "bob"})
	if len(repo.created) != 1 || repo.created[0].Recipients[0] != "amy" || repo.created[0].External {
		t.Fatalf("got %#v", repo.created)
	}
	if len(ch.got) != 0 {
		t.Fatal("acceptance is in-app only")
	}
}

func TestOtherFlowActionsDoNotNotify(t *testing.T) {
	repo := &fakeNotificationRepo{doc: &query.DocumentBrief{ID: "d1"}}
	newTestNotifier(repo, &fakeChannel{}).OnFlowAction(context.Background(), command.FlowActionInput{DocumentID: "d1", Action: "finalize", ActorID: "x"})
	if len(repo.created) != 0 {
		t.Fatal("finalize must not notify")
	}
}

func TestHandoverConfirmNotifiesManagers(t *testing.T) {
	repo := &fakeNotificationRepo{handover: &query.HandoverBrief{ID: "h1", TargetName: "甲", ReceiverName: "乙", ReceiverUserID: "rcv", TargetUserID: "tgt", GeneratedBy: "lead", ManagerIDs: []string{"lead", "mgr"}}}
	s := newTestNotifier(repo, &fakeChannel{})
	s.OnHandoverAction(context.Background(), command.HandoverActionInput{HandoverID: "h1", Action: "confirm", ActorID: "rcv"})
	if len(repo.created) != 1 || len(repo.created[0].Recipients) != 2 {
		t.Fatalf("confirm should reach lead+mgr once each: %#v", repo.created)
	}
	s.OnHandoverCreated(context.Background(), "h1", "lead")
	if len(repo.created) != 3 {
		t.Fatalf("create notifies receiver and target: %#v", repo.created)
	}
}

func TestAssistantResultOnlyForSummaries(t *testing.T) {
	repo := &fakeNotificationRepo{doc: &query.DocumentBrief{ID: "d1", Title: "报告"}, request: &query.AssistantRequestBrief{RequestType: "assistant.ask", CreatedBy: "u"}}
	s := newTestNotifier(repo, &fakeChannel{})
	s.OnAssistantResult(context.Background(), task.Result{RequestID: "r", Status: "completed"})
	if len(repo.created) != 0 {
		t.Fatal("plain ask must not notify")
	}
	repo.request = &query.AssistantRequestBrief{RequestType: "document.summarize", RelatedType: "document", RelatedID: "d1", CreatedBy: "u"}
	s.OnAssistantResult(context.Background(), task.Result{RequestID: "r", Status: "failed", ErrorMessage: "timeout"})
	if len(repo.created) != 1 || repo.created[0].EventType != "ai_summary_failed" || repo.created[0].Title != "《报告》的 AI 摘要生成失败" {
		t.Fatalf("got %#v", repo.created)
	}
}

func TestNotificationStoreFailureIsSwallowed(t *testing.T) {
	repo := &fakeNotificationRepo{createErr: errors.New("db down")}
	ch := &fakeChannel{}
	newTestNotifier(repo, ch).Notify(context.Background(), command.NotificationInput{Recipients: []string{"a"}, External: true})
	if len(ch.got) != 0 {
		t.Fatal("external delivery must not run when the in-app row failed")
	}
}

func TestZeroValueNotificationServiceIsInert(t *testing.T) {
	var s NotificationService
	s.OnFlowAction(context.Background(), command.FlowActionInput{Action: "transfer"})
	if items, err := s.Inbox(context.Background(), "u", "member"); err != nil || len(items) != 0 {
		t.Fatal("zero value must be a no-op")
	}
}
