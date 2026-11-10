package service

import (
	"context"
	"log"
	"strings"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/repository"
)

// Audit action types for access events. Business-write actions (transfer,
// archive, handover_*) are written by the repositories inside their own
// transactions and are not listed here.
const (
	AuditActionView     = "view"
	AuditActionPreview  = "preview"
	AuditActionDownload = "download"
)

// viewDedupWindowSeconds keeps one "view" row per user/document per window:
// the detail page fetches the document several times while it renders.
const viewDedupWindowSeconds = 600

type AuditService struct {
	writer repository.AuditWriter
}

func NewAuditService(writer ...repository.AuditWriter) AuditService {
	if len(writer) > 0 {
		return AuditService{writer: writer[0]}
	}
	return AuditService{}
}

// Record writes an access event. Callers that hand out file content must treat
// an error as fatal (fail closed): "who downloaded what" is a product promise,
// and silently serving a file whose download was not recorded breaks it.
func (s AuditService) Record(ctx context.Context, event command.AuditEventInput) error {
	if strings.TrimSpace(event.ActionType) == "" {
		return ErrValidation
	}
	if event.ActionType == AuditActionView && event.DedupWindowSeconds == 0 {
		event.DedupWindowSeconds = viewDedupWindowSeconds
	}
	if s.writer == nil {
		log.Printf("audit action=%s user=%s document=%s version=%s", event.ActionType, event.UserID, event.DocumentID, event.VersionID)
		return nil
	}
	return s.writer.RecordAuditEvent(ctx, event)
}

// RecordBestEffort is for events whose loss must not block the user, such as
// opening a detail page. Failures are logged, not returned.
func (s AuditService) RecordBestEffort(ctx context.Context, event command.AuditEventInput) {
	if err := s.Record(ctx, event); err != nil {
		log.Printf("[audit] record %s failed: %v", event.ActionType, err)
	}
}
