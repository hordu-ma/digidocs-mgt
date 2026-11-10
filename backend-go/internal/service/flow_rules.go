package service

// flowTransitions is the single definition of which document status each flow
// action may start from. The postgres action repository enforces it inside the
// write transaction; the document detail uses it to tell the UI which buttons
// to show. Before 2026-11-10 the frontend kept its own copy that had drifted:
// it offered nothing for handed_over documents, which the backend allows to be
// resumed, finalized or archived, so those documents were stuck in the UI.
var flowTransitions = map[string][]string{
	"mark_in_progress": {"draft", "handed_over", "in_progress"},
	"transfer":         {"in_progress"},
	"accept_transfer":  {"pending_handover"},
	"finalize":         {"in_progress", "handed_over"},
	"archive":          {"finalized", "handed_over"},
	"unarchive":        {"archived"},
}

// flowActionOrder is the display order of the action buttons.
var flowActionOrder = []string{"accept_transfer", "mark_in_progress", "transfer", "finalize", "archive", "unarchive"}

// ValidFlowTransition reports whether action may run on a document in status.
func ValidFlowTransition(status string, action string) bool {
	for _, from := range flowTransitions[action] {
		if from == status {
			return true
		}
	}
	return false
}

// managerOnlyFlowActions need a project owner/manager (or admin); every other
// action is also open to the document's current owner. Mirrors
// PermissionRepository.CanFlowDocument.
var managerOnlyFlowActions = map[string]bool{"finalize": true, "archive": true, "unarchive": true}

func flowDeniedReason(action string) string {
	if managerOnlyFlowActions[action] {
		return "仅课题负责人或协管人可以执行"
	}
	return "仅当前责任人或课题负责人、协管人可以执行"
}
