package postgres_test

import (
	"context"
	"testing"

	"digidocs-mgt/backend-go/internal/domain/command"
	pgrepo "digidocs-mgt/backend-go/internal/repository/postgres"
	"digidocs-mgt/backend-go/internal/service"
)

func TestPostgresIntegrationNotificationsAndInbox(t *testing.T) {
	ctx := context.Background()
	database := openIntegrationDB(t)
	seedIntegrationBaseRows(t, ctx, database)
	const lead = integrationUserID
	const bob = "00000000-0000-0000-0000-000000000301"
	const carol = "00000000-0000-0000-0000-000000000302"
	for id, name := range map[string]string{bob: "bob", carol: "carol"} {
		if _, err := database.ExecContext(ctx, `INSERT INTO users(id,username,password_hash,display_name,role,status) VALUES($1,$2,'hash',$2,'member','active')`, id, name); err != nil {
			t.Fatal(err)
		}
	}

	permissions := service.NewPermissionService(pgrepo.NewPermissionRepository(database))
	members := pgrepo.NewMembershipRepository(database)
	notifications := service.NewNotificationService(pgrepo.NewNotificationRepository(database), "")
	membership := service.NewMembershipService(members, permissions).WithNotifications(notifications)
	if _, err := membership.AddMember(ctx, lead, "member", integrationProjectID, bob, "contributor"); err != nil {
		t.Fatal(err)
	}
	if _, err := membership.AddMember(ctx, lead, "member", integrationProjectID, carol, "contributor"); err != nil {
		t.Fatal(err)
	}

	unread := func(user string) int {
		t.Helper()
		n, err := notifications.UnreadCount(ctx, user)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	inboxKinds := func(user string) map[string]int {
		t.Helper()
		items, err := notifications.Inbox(ctx, user, "member")
		if err != nil {
			t.Fatal(err)
		}
		kinds := map[string]int{}
		for _, it := range items {
			kinds[it.Kind]++
		}
		return kinds
	}
	if unread(bob) != 1 {
		t.Fatalf("bob should be told he was added, unread=%d", unread(bob))
	}

	// Transfer: lead → bob. Bob gets a notification and an accept_transfer task.
	documents := pgrepo.NewDocumentRepository(database)
	doc, err := documents.CreateDocument(ctx, command.DocumentCreateInput{TeamSpaceID: integrationTeamSpaceID, ProjectID: integrationProjectID, Title: "开题报告", CurrentOwnerID: lead, ActorID: lead})
	if err != nil {
		t.Fatal(err)
	}
	docID := doc["id"].(string)
	actions := pgrepo.NewActionRepository(database)
	flows := service.NewFlowService(pgrepo.NewFlowRepository(database), actions, permissions).WithNotifications(notifications)
	if _, err := flows.ApplyAction(ctx, command.FlowActionInput{DocumentID: docID, Action: "mark_in_progress", ActorID: lead, ActorRole: "member"}); err != nil {
		t.Fatal(err)
	}
	if _, err := flows.ApplyAction(ctx, command.FlowActionInput{DocumentID: docID, Action: "transfer", ToUserID: bob, ActorID: lead, ActorRole: "member"}); err != nil {
		t.Fatal(err)
	}
	if unread(bob) != 2 || inboxKinds(bob)["accept_transfer"] != 1 {
		t.Fatalf("bob after transfer: unread=%d inbox=%v", unread(bob), inboxKinds(bob))
	}
	list, _, err := notifications.List(ctx, bob, true, 1, 10)
	if err != nil || list[0].EventType != "document_transfer" || list[0].ActorName != "集成测试用户" || list[0].Link != "/documents/"+docID {
		t.Fatalf("latest notification: %#v %v", list, err)
	}

	// Accept: lead is told; the task leaves bob's inbox.
	if _, err := flows.ApplyAction(ctx, command.FlowActionInput{DocumentID: docID, Action: "accept_transfer", ActorID: bob, ActorRole: "member"}); err != nil {
		t.Fatal(err)
	}
	if unread(lead) != 1 || inboxKinds(bob)["accept_transfer"] != 0 {
		t.Fatalf("after accept: lead unread=%d bob inbox=%v", unread(lead), inboxKinds(bob))
	}

	// Mark read is per user: carol cannot mark bob's notification.
	if err := notifications.MarkRead(ctx, carol, list[0].ID); err == nil {
		t.Fatal("foreign notification must not be markable")
	}
	if err := notifications.MarkRead(ctx, bob, list[0].ID); err != nil || unread(bob) != 1 {
		t.Fatalf("mark read: %v unread=%d", err, unread(bob))
	}
	if n, err := notifications.MarkAllRead(ctx, bob); err != nil || n != 1 || unread(bob) != 0 {
		t.Fatalf("mark all: n=%d err=%v unread=%d", n, err, unread(bob))
	}

	// Handover bob → carol.
	handovers := service.NewHandoverService(pgrepo.NewHandoverRepository(database), actions, permissions).WithNotifications(notifications)
	created, err := handovers.Create(ctx, command.HandoverCreateInput{TargetUserID: bob, ReceiverUserID: carol, ProjectID: integrationProjectID, ActorID: lead, ActorRole: "member"})
	if err != nil {
		t.Fatal(err)
	}
	handoverID := created["id"].(string)
	if inboxKinds(carol)["confirm_handover"] != 1 || inboxKinds(lead)["fill_handover_items"] != 1 {
		t.Fatalf("after create: carol=%v lead=%v", inboxKinds(carol), inboxKinds(lead))
	}
	if _, err := handovers.UpdateItems(ctx, command.HandoverItemUpdateInput{HandoverID: handoverID, ActorID: lead, ActorRole: "member", Items: []command.HandoverItemInput{{DocumentID: docID, Selected: true}}}); err != nil {
		t.Fatal(err)
	}
	if inboxKinds(lead)["fill_handover_items"] != 0 {
		t.Fatal("filled handover must leave the lead inbox")
	}
	leadBefore := unread(lead)
	if _, err := handovers.ApplyAction(ctx, command.HandoverActionInput{HandoverID: handoverID, Action: "confirm", ActorID: carol, ActorRole: "member"}); err != nil {
		t.Fatal(err)
	}
	if inboxKinds(carol)["confirm_handover"] != 0 || inboxKinds(lead)["complete_handover"] != 1 || unread(lead) != leadBefore+1 {
		t.Fatalf("after confirm: carol=%v lead=%v unread=%d", inboxKinds(carol), inboxKinds(lead), unread(lead))
	}
	// A plain member never sees manager tasks.
	if inboxKinds(bob)["complete_handover"] != 0 {
		t.Fatal("bob is not a manager")
	}
}
