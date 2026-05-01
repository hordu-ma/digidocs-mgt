package postgres

import (
	"regexp"
	"testing"
	"time"

	"digidocs-mgt/backend-go/internal/domain/command"
)

func TestFlowTransitionHelpers(t *testing.T) {
	valid := []struct {
		status string
		action string
	}{
		{status: "draft", action: "mark_in_progress"},
		{status: "handed_over", action: "mark_in_progress"},
		{status: "in_progress", action: "transfer"},
		{status: "pending_handover", action: "accept_transfer"},
		{status: "in_progress", action: "finalize"},
		{status: "handed_over", action: "archive"},
		{status: "archived", action: "unarchive"},
	}
	for _, tc := range valid {
		if !isValidFlowTransition(tc.status, tc.action) {
			t.Fatalf("isValidFlowTransition(%q, %q) = false, want true", tc.status, tc.action)
		}
	}
	invalid := []struct {
		status string
		action string
	}{
		{status: "draft", action: "transfer"},
		{status: "finalized", action: "accept_transfer"},
		{status: "archived", action: "archive"},
		{status: "draft", action: "unknown"},
	}
	for _, tc := range invalid {
		if isValidFlowTransition(tc.status, tc.action) {
			t.Fatalf("isValidFlowTransition(%q, %q) = true, want false", tc.status, tc.action)
		}
	}
}

func TestHandoverTransitionHelpers(t *testing.T) {
	if !isValidHandoverTransition("generated", "confirm") {
		t.Fatal("generated confirm should be valid")
	}
	if !isValidHandoverTransition("pending_confirm", "complete") {
		t.Fatal("pending_confirm complete should be valid")
	}
	if !isValidHandoverTransition("generated", "cancel") || !isValidHandoverTransition("pending_confirm", "cancel") {
		t.Fatal("cancel should be valid for generated and pending_confirm")
	}
	if isValidHandoverTransition("completed", "cancel") || isValidHandoverTransition("generated", "complete") {
		t.Fatal("invalid handover transition returned true")
	}
}

func TestActionMappingHelpers(t *testing.T) {
	auditCases := map[string]string{
		"transfer":        "transfer",
		"accept_transfer": "receive_transfer",
		"finalize":        "finalize",
		"archive":         "archive",
		"unarchive":       "restore",
		"unknown":         "admin_update",
	}
	for action, want := range auditCases {
		if got := mappedAuditAction(action); got != want {
			t.Fatalf("mappedAuditAction(%q) = %q, want %q", action, got, want)
		}
	}

	handoverAuditCases := map[string]string{
		"confirm":  "handover_confirm",
		"complete": "handover_complete",
		"cancel":   "admin_update",
	}
	for action, want := range handoverAuditCases {
		if got := mappedHandoverAuditAction(action); got != want {
			t.Fatalf("mappedHandoverAuditAction(%q) = %q, want %q", action, got, want)
		}
	}
}

func TestStatusMappingHelpers(t *testing.T) {
	statusCases := map[string]string{
		"archive":          "archived",
		"finalize":         "finalized",
		"transfer":         "pending_handover",
		"accept_transfer":  "in_progress",
		"mark_in_progress": "in_progress",
		"unarchive":        "finalized",
		"unknown":          "in_progress",
	}
	for action, want := range statusCases {
		if got := flowActionToStatus(action); got != want {
			t.Fatalf("flowActionToStatus(%q) = %q, want %q", action, got, want)
		}
	}

	handoverCases := map[string][2]string{
		"confirm":  {"pending_confirm", "confirmed_at"},
		"complete": {"completed", "completed_at"},
		"cancel":   {"cancelled", "cancelled_at"},
		"unknown":  {"generated", "generated_at"},
	}
	for action, want := range handoverCases {
		gotStatus, gotColumn := handoverActionToUpdate(action)
		if gotStatus != want[0] || gotColumn != want[1] {
			t.Fatalf("handoverActionToUpdate(%q) = (%q, %q), want (%q, %q)", action, gotStatus, gotColumn, want[0], want[1])
		}
	}
}

func TestOwnerAndActorHelpers(t *testing.T) {
	transfer := command.FlowActionInput{Action: "transfer", ToUserID: "u-2"}
	if got := nextOwnerID(transfer, "u-1"); got != "u-2" {
		t.Fatalf("nextOwnerID transfer = %q, want u-2", got)
	}
	if got := nextOwnerID(command.FlowActionInput{Action: "transfer"}, "u-1"); got != "u-1" {
		t.Fatalf("nextOwnerID fallback = %q, want u-1", got)
	}
	if got := nullableOwnerID(transfer, "u-2"); got != "u-2" {
		t.Fatalf("nullableOwnerID transfer = %q, want u-2", got)
	}
	if got := nullableOwnerID(command.FlowActionInput{Action: "archive"}, "u-2"); got != "" {
		t.Fatalf("nullableOwnerID archive = %q, want empty", got)
	}
	if got := actorOrSystem(""); got != systemUserID() {
		t.Fatalf("actorOrSystem empty = %q, want system user", got)
	}
	if got := actorOrSystem("u-1"); got != "u-1" {
		t.Fatalf("actorOrSystem explicit = %q, want u-1", got)
	}
}

func TestNewIDAndNowUTC(t *testing.T) {
	id := newID()
	if !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(id) {
		t.Fatalf("newID() = %q, want uuid v4 shape", id)
	}
	if nowUTC().Location() != time.UTC {
		t.Fatalf("nowUTC location = %v, want UTC", nowUTC().Location())
	}
}
