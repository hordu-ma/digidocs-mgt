package service

import (
	"context"
	"log"
	"strings"
	"time"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/domain/task"
	"digidocs-mgt/backend-go/internal/notify"
	"digidocs-mgt/backend-go/internal/repository"
)

// NotificationService turns business events into in-app notifications and,
// for important events, optional external copies (webhook / email).
//
// Notifications are reminders, never business facts: every On* hook runs after
// the business write has committed, and a failure here is logged, not
// returned, so it can never roll back or fail a transfer or handover.
type NotificationService struct {
	repo          repository.NotificationRepository
	channels      []notify.Channel
	publicBaseURL string
	// async controls external delivery; tests set it false to deliver inline.
	async bool
}

func NewNotificationService(repo repository.NotificationRepository, publicBaseURL string, channels ...notify.Channel) NotificationService {
	return NotificationService{
		repo:          repo,
		channels:      channels,
		publicBaseURL: strings.TrimRight(strings.TrimSpace(publicBaseURL), "/"),
		async:         true,
	}
}

func (s NotificationService) enabled() bool { return s.repo != nil }

// Channels reports the configured external channel names (for startup logs).
func (s NotificationService) Channels() []string {
	names := make([]string, 0, len(s.channels))
	for _, c := range s.channels {
		names = append(names, c.Name())
	}
	return names
}

// ─────────────────────────── read side ───────────────────────────

func (s NotificationService) Inbox(ctx context.Context, actorID, actorRole string) ([]query.InboxItem, error) {
	if !s.enabled() {
		return []query.InboxItem{}, nil
	}
	return s.repo.ListInbox(ctx, actorID, actorRole == "admin")
}

func (s NotificationService) List(ctx context.Context, actorID string, unreadOnly bool, page, pageSize int) ([]query.NotificationItem, int, error) {
	if !s.enabled() {
		return []query.NotificationItem{}, 0, nil
	}
	return s.repo.ListNotifications(ctx, actorID, unreadOnly, page, pageSize)
}

func (s NotificationService) UnreadCount(ctx context.Context, actorID string) (int, error) {
	if !s.enabled() {
		return 0, nil
	}
	return s.repo.CountUnread(ctx, actorID)
}

func (s NotificationService) MarkRead(ctx context.Context, actorID, id string) error {
	if !s.enabled() {
		return ErrNotFound
	}
	return s.repo.MarkRead(ctx, actorID, id)
}

func (s NotificationService) MarkAllRead(ctx context.Context, actorID string) (int, error) {
	if !s.enabled() {
		return 0, nil
	}
	return s.repo.MarkAllRead(ctx, actorID)
}

// ─────────────────────────── write side ──────────────────────────

// Notify stores one notification per recipient (minus the actor) and, when
// External is set, fans it out to the external channels.
func (s NotificationService) Notify(ctx context.Context, input command.NotificationInput) {
	if !s.enabled() {
		return
	}
	input.Recipients = dedupeExcept(input.Recipients, input.ActorID)
	if len(input.Recipients) == 0 {
		return
	}
	if err := s.repo.CreateNotifications(ctx, input); err != nil {
		log.Printf("[notify] store %s failed: %v", input.EventType, err)
		return
	}
	if !input.External || len(s.channels) == 0 {
		return
	}
	deliver := func(ctx context.Context) {
		contacts, err := s.repo.GetUserContacts(ctx, input.Recipients)
		if err != nil {
			log.Printf("[notify] resolve recipients for %s failed: %v", input.EventType, err)
			return
		}
		msg := notify.Message{Title: input.Title, Body: input.Body}
		if s.publicBaseURL != "" && input.Link != "" {
			msg.Link = s.publicBaseURL + input.Link
		}
		for _, c := range contacts {
			msg.Recipients = append(msg.Recipients, notify.Recipient{DisplayName: c.DisplayName, Email: c.Email})
		}
		for _, ch := range s.channels {
			if err := ch.Deliver(ctx, msg); err != nil {
				log.Printf("[notify] %s delivery of %s failed: %v", ch.Name(), input.EventType, err)
			}
		}
	}
	if !s.async {
		deliver(ctx)
		return
	}
	// Detach from the request: the HTTP response must not wait on a slow SMTP
	// server, and the request context is cancelled as soon as it returns.
	go func() {
		bg, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		deliver(bg)
	}()
}

// OnFlowAction notifies the other party of a transfer or its acceptance.
func (s NotificationService) OnFlowAction(ctx context.Context, input command.FlowActionInput) {
	if !s.enabled() {
		return
	}
	switch input.Action {
	case "transfer", "accept_transfer":
	default:
		return
	}
	doc, err := s.repo.GetDocumentBrief(ctx, input.DocumentID)
	if err != nil {
		log.Printf("[notify] document brief %s: %v", input.DocumentID, err)
		return
	}
	actor := s.displayName(ctx, input.ActorID)
	link := "/documents/" + doc.ID
	if input.Action == "transfer" {
		body := actor + " 把《" + doc.Title + "》转交给你，请在文档详情页接收。"
		if strings.TrimSpace(input.Note) != "" {
			body += "\n备注：" + input.Note
		}
		s.Notify(ctx, command.NotificationInput{
			Recipients: []string{input.ToUserID}, EventType: "document_transfer",
			Title: "待接收：《" + doc.Title + "》", Body: body, Link: link,
			RelatedType: "document", RelatedID: doc.ID, ActorID: input.ActorID, External: true,
		})
		return
	}
	sender, err := s.repo.GetLastTransferSender(ctx, doc.ID)
	if err != nil || sender == "" {
		return
	}
	s.Notify(ctx, command.NotificationInput{
		Recipients: []string{sender}, EventType: "document_transfer_accepted",
		Title: actor + " 已接收《" + doc.Title + "》", Body: "转交已完成，责任人已变更为 " + actor + "。",
		Link: link, RelatedType: "document", RelatedID: doc.ID, ActorID: input.ActorID,
	})
}

// OnHandoverCreated tells the leaving member and the receiver a handover exists.
func (s NotificationService) OnHandoverCreated(ctx context.Context, handoverID, actorID string) {
	h := s.handover(ctx, handoverID)
	if h == nil {
		return
	}
	s.Notify(ctx, command.NotificationInput{
		Recipients: []string{h.ReceiverUserID}, EventType: "handover_created",
		Title: "你被指定为 " + h.TargetName + " 的交接接收人",
		Body:  projectPrefix(h.ProjectName) + "负责人整理好清单后，请在交接页确认接收。",
		Link:  "/handovers?id=" + h.ID, RelatedType: "handover", RelatedID: h.ID, ActorID: actorID, External: true,
	})
	s.Notify(ctx, command.NotificationInput{
		Recipients: []string{h.TargetUserID}, EventType: "handover_created",
		Title: "已为你发起交接，接收人：" + h.ReceiverName,
		Body:  projectPrefix(h.ProjectName) + "请配合负责人确认需要交接的文档与数据。",
		Link:  "/handovers?id=" + h.ID, RelatedType: "handover", RelatedID: h.ID, ActorID: actorID,
	})
}

// OnHandoverAction notifies the next responsible party of a handover step.
func (s NotificationService) OnHandoverAction(ctx context.Context, input command.HandoverActionInput) {
	h := s.handover(ctx, input.HandoverID)
	if h == nil {
		return
	}
	link := "/handovers?id=" + h.ID
	base := command.NotificationInput{Link: link, RelatedType: "handover", RelatedID: h.ID, ActorID: input.ActorID}
	switch input.Action {
	case "confirm":
		n := base
		n.Recipients = append(append([]string{}, h.ManagerIDs...), h.GeneratedBy)
		n.EventType = "handover_confirmed"
		n.Title = h.ReceiverName + " 已确认接收 " + h.TargetName + " 的交接"
		n.Body = projectPrefix(h.ProjectName) + "请核对后在交接页点击「完成交接」。"
		n.External = true
		s.Notify(ctx, n)
	case "complete":
		n := base
		n.Recipients = []string{h.TargetUserID, h.ReceiverUserID}
		n.EventType = "handover_completed"
		n.Title = h.TargetName + " → " + h.ReceiverName + " 的交接已完成"
		n.Body = projectPrefix(h.ProjectName) + "清单中的文档责任人已变更为 " + h.ReceiverName + "。"
		s.Notify(ctx, n)
	case "cancel":
		n := base
		n.Recipients = []string{h.TargetUserID, h.ReceiverUserID}
		n.EventType = "handover_cancelled"
		n.Title = h.TargetName + " → " + h.ReceiverName + " 的交接已取消"
		n.Body = projectPrefix(h.ProjectName) + strings.TrimSpace("原因："+input.Reason)
		if strings.TrimSpace(input.Reason) == "" {
			n.Body = projectPrefix(h.ProjectName) + "交接单已取消。"
		}
		s.Notify(ctx, n)
	}
}

// OnAssistantResult tells the requester a summary finished (or failed).
// Plain Q&A is skipped: the asker is watching the chat already.
func (s NotificationService) OnAssistantResult(ctx context.Context, result task.Result) {
	if !s.enabled() {
		return
	}
	req, err := s.repo.GetAssistantRequestBrief(ctx, result.RequestID)
	if err != nil || req.CreatedBy == "" {
		return
	}
	var title, link string
	switch req.RequestType {
	case string(task.TaskTypeDocumentSummarize):
		doc, err := s.repo.GetDocumentBrief(ctx, req.RelatedID)
		if err != nil {
			return
		}
		title, link = "《"+doc.Title+"》的 AI 摘要", "/documents/"+doc.ID
	case string(task.TaskTypeHandoverSummarize):
		title, link = "交接单的 AI 摘要", "/handovers?id="+req.RelatedID
	default:
		return
	}
	event, body := "ai_summary_completed", "已生成，可以查看并确认 AI 建议。"
	title += "已生成"
	if result.Status != "completed" {
		event, body = "ai_summary_failed", "失败原因："+truncateRunes(result.ErrorMessage, 120)
		title = strings.TrimSuffix(title, "已生成") + "生成失败"
	}
	s.Notify(ctx, command.NotificationInput{
		Recipients: []string{req.CreatedBy}, EventType: event, Title: title,
		Body: body, Link: link, RelatedType: req.RelatedType, RelatedID: req.RelatedID,
	})
}

// OnMemberJoined notifies the new member (added by someone else) or the
// project managers (joined through an invite link).
func (s NotificationService) OnMemberJoined(ctx context.Context, projectID, userID, actorID string, viaInvite bool) {
	if !s.enabled() {
		return
	}
	managers, projectName, err := s.repo.GetProjectManagers(ctx, projectID)
	if err != nil {
		log.Printf("[notify] project managers %s: %v", projectID, err)
		return
	}
	if viaInvite {
		s.Notify(ctx, command.NotificationInput{
			Recipients: managers, EventType: "member_joined",
			Title: s.displayName(ctx, userID) + " 通过邀请链接加入了 " + projectName,
			Link:  "/members", RelatedType: "project", RelatedID: projectID, ActorID: userID,
		})
		return
	}
	s.Notify(ctx, command.NotificationInput{
		Recipients: []string{userID}, EventType: "member_added",
		Title: "你已被加入课题 " + projectName, Body: s.displayName(ctx, actorID) + " 把你加入了该课题。",
		Link: "/documents", RelatedType: "project", RelatedID: projectID, ActorID: actorID,
	})
}

func (s NotificationService) handover(ctx context.Context, id string) *query.HandoverBrief {
	if !s.enabled() || id == "" {
		return nil
	}
	h, err := s.repo.GetHandoverBrief(ctx, id)
	if err != nil {
		log.Printf("[notify] handover brief %s: %v", id, err)
		return nil
	}
	return h
}

func (s NotificationService) displayName(ctx context.Context, userID string) string {
	contacts, err := s.repo.GetUserContacts(ctx, []string{userID})
	if err != nil || len(contacts) == 0 {
		return "有成员"
	}
	return contacts[0].DisplayName
}

func projectPrefix(name string) string {
	if name == "" {
		return ""
	}
	return "课题「" + name + "」："
}

func dedupeExcept(ids []string, except string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || id == except {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func truncateRunes(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}
