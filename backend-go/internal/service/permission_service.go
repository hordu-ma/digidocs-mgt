package service

import (
	"context"
	"fmt"

	"digidocs-mgt/backend-go/internal/repository"
)

type PermissionService struct {
	reader repository.PermissionReader
}

func NewPermissionService(reader repository.PermissionReader) PermissionService {
	return PermissionService{reader: reader}
}

func (s PermissionService) EnsureCreateDocument(ctx context.Context, actorID string, actorRole string, projectID string) error {
	if s.reader == nil {
		return nil
	}
	ok, err := s.reader.CanCreateDocument(ctx, actorID, actorRole, projectID)
	return permissionResult(ok, err, "create document")
}

func (s PermissionService) EnsureUpdateDocument(ctx context.Context, actorID string, actorRole string, documentID string) error {
	if s.reader == nil {
		return nil
	}
	ok, err := s.reader.CanUpdateDocument(ctx, actorID, actorRole, documentID)
	return permissionResult(ok, err, "update document")
}

func (s PermissionService) EnsureDeleteDocument(ctx context.Context, actorID string, actorRole string, documentID string) error {
	if s.reader == nil {
		return nil
	}
	ok, err := s.reader.CanDeleteDocument(ctx, actorID, actorRole, documentID)
	return permissionResult(ok, err, "delete document")
}

func (s PermissionService) EnsureUploadVersion(ctx context.Context, actorID string, actorRole string, documentID string) error {
	if s.reader == nil {
		return nil
	}
	ok, err := s.reader.CanUploadVersion(ctx, actorID, actorRole, documentID)
	return permissionResult(ok, err, "upload version")
}

func (s PermissionService) EnsureFlowDocument(ctx context.Context, actorID string, actorRole string, documentID string, action string) error {
	if s.reader == nil {
		return nil
	}
	ok, err := s.reader.CanFlowDocument(ctx, actorID, actorRole, documentID, action)
	return permissionResult(ok, err, "flow document")
}

func (s PermissionService) EnsureCreateHandover(ctx context.Context, actorID string, actorRole string, projectID string) error {
	if s.reader == nil {
		return nil
	}
	ok, err := s.reader.CanCreateHandover(ctx, actorID, actorRole, projectID)
	return permissionResult(ok, err, "create handover")
}

func (s PermissionService) EnsureUpdateHandoverItems(ctx context.Context, actorID string, actorRole string, handoverID string) error {
	if s.reader == nil {
		return nil
	}
	ok, err := s.reader.CanUpdateHandoverItems(ctx, actorID, actorRole, handoverID)
	return permissionResult(ok, err, "update handover items")
}

func (s PermissionService) EnsureApplyHandover(ctx context.Context, actorID string, actorRole string, handoverID string, action string) error {
	if s.reader == nil {
		return nil
	}
	ok, err := s.reader.CanApplyHandover(ctx, actorID, actorRole, handoverID, action)
	return permissionResult(ok, err, "apply handover")
}

func permissionResult(ok bool, err error, action string) error {
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: cannot %s", ErrForbidden, action)
	}
	return nil
}
