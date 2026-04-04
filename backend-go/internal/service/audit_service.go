package service

import (
	"context"
	"log"
)

type AuditService struct{}

func NewAuditService() AuditService {
	return AuditService{}
}

func (s AuditService) Record(
	ctx context.Context,
	actionType string,
	userID string,
	documentID string,
	extraData map[string]any,
) error {
	_ = ctx
	log.Printf("audit action=%s user=%s document=%s extra=%v", actionType, userID, documentID, extraData)
	return nil
}
