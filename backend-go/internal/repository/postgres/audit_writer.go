package postgres

import (
	"context"
	"encoding/json"
	"net"
	"strings"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/shared"
)

// RecordAuditEvent inserts one access event. With DedupWindowSeconds > 0 the
// insert is skipped when the same user already has the same action on the same
// document/version inside the window.
func (r AuditRepository) RecordAuditEvent(ctx context.Context, event command.AuditEventInput) error {
	extra := event.Extra
	if extra == nil {
		extra = map[string]any{}
	}
	extraData, err := json.Marshal(extra)
	if err != nil {
		return err
	}
	userID := event.UserID
	if userID == "" {
		userID = systemUserID()
	}
	ip := strings.TrimSpace(event.IPAddress)
	if net.ParseIP(ip) == nil {
		ip = ""
	}
	terminal := event.TerminalInfo
	if len(terminal) > 255 {
		terminal = terminal[:255]
	}

	_, err = r.db.ExecContext(
		ctx,
		`
		INSERT INTO audit_events (
			id, document_id, version_id, user_id, action_type,
			request_id, ip_address, terminal_info, extra_data, created_at
		)
		SELECT
			$1::uuid,
			NULLIF($2::text, '')::uuid,
			NULLIF($3::text, '')::uuid,
			$4::uuid,
			$5::varchar,
			NULLIF($6::text, ''),
			NULLIF($7::text, '')::inet,
			NULLIF($8::text, ''),
			$9::jsonb,
			$10::timestamptz
		WHERE $11::int <= 0 OR NOT EXISTS (
			SELECT 1 FROM audit_events ae
			WHERE ae.user_id = $4::uuid
			  AND ae.action_type = $5::varchar
			  AND ae.document_id IS NOT DISTINCT FROM NULLIF($2::text, '')::uuid
			  AND ae.version_id IS NOT DISTINCT FROM NULLIF($3::text, '')::uuid
			  AND ae.created_at > $10::timestamptz - make_interval(secs => $11::int)
		)
		`,
		newID(),
		event.DocumentID,
		event.VersionID,
		userID,
		event.ActionType,
		shared.RequestIDFromContext(ctx),
		ip,
		terminal,
		string(extraData),
		nowUTC(),
		event.DedupWindowSeconds,
	)
	return err
}
