package service

import (
	"context"
	"fmt"
	"io"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/repository"
	"digidocs-mgt/backend-go/internal/shared"
	"digidocs-mgt/backend-go/internal/storage"
)

type VersionService struct {
	storage     storage.Provider
	workflow    repository.VersionWorkflow
	reader      repository.VersionReader
	permissions PermissionService
}

func NewVersionService(
	storage storage.Provider,
	workflow repository.VersionWorkflow,
	reader repository.VersionReader,
	permissions ...PermissionService,
) VersionService {
	permissionService := PermissionService{}
	if len(permissions) > 0 {
		permissionService = permissions[0]
	}
	return VersionService{storage: storage, workflow: workflow, reader: reader, permissions: permissionService}
}

func (s VersionService) UploadAndCreateVersion(
	ctx context.Context,
	documentID string,
	fileName string,
	fileSize int64,
	commitMessage string,
	reader io.Reader,
	actorID string,
	actorRoles ...string,
) (map[string]any, error) {
	if documentID == "" {
		return nil, fmt.Errorf("%w: document_id is required", ErrValidation)
	}
	if fileName == "" {
		return nil, fmt.Errorf("%w: file_name is required", ErrValidation)
	}
	if actorID == "" {
		return nil, fmt.Errorf("%w: actor_id is required", ErrValidation)
	}
	actorRole := ""
	if len(actorRoles) > 0 {
		actorRole = actorRoles[0]
	}
	if err := s.permissions.EnsureUploadVersion(ctx, actorID, actorRole, documentID); err != nil {
		return nil, err
	}

	objectKey := fmt.Sprintf("documents/%s/%s", documentID, shared.SanitizeStoredFileName(fileName))
	result, err := s.storage.PutObject(ctx, storage.PutObjectInput{
		ObjectKey:   objectKey,
		Reader:      reader,
		CreatePaths: true,
	})
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}

	versionData, err := s.workflow.CreateUploadedVersion(ctx, command.VersionCreateInput{
		DocumentID:       documentID,
		FileName:         fileName,
		FileSize:         fileSize,
		CommitMessage:    commitMessage,
		StorageObjectKey: result.ObjectKey,
		StorageProvider:  result.Provider,
		ActorID:          actorID,
	})
	if err != nil {
		return nil, fmt.Errorf("version creation failed: %w", err)
	}

	versionData["storage"] = result
	versionData["status"] = "uploaded"
	return versionData, nil
}

// List returns the version history of a document. The caller must be allowed to
// read the document itself: version ids are surfaced by the audit log, so an
// unguarded version listing would hand out cross-project file handles.
func (s VersionService) List(ctx context.Context, documentID string, actorID string, actorRole string) ([]query.VersionItem, error) {
	if documentID == "" {
		return nil, fmt.Errorf("%w: document_id is required", ErrValidation)
	}
	if err := s.permissions.EnsureViewDocument(ctx, actorID, actorRole, documentID); err != nil {
		return nil, err
	}
	return s.reader.ListVersions(ctx, documentID)
}

func (s VersionService) Get(ctx context.Context, versionID string, actorID string, actorRole string) (*query.VersionDetail, error) {
	if versionID == "" {
		return nil, fmt.Errorf("%w: version_id is required", ErrValidation)
	}
	if err := s.permissions.EnsureViewVersion(ctx, actorID, actorRole, versionID); err != nil {
		return nil, err
	}
	return s.reader.GetVersion(ctx, versionID)
}

// GetFile returns the file content for a version from storage.
//
// The worker-facing internal endpoint has no acting user and passes
// ("", "admin"), matching DocumentService.GetDocument's existing convention.
func (s VersionService) GetFile(ctx context.Context, versionID string, actorID string, actorRole string) (*query.VersionDetail, *storage.GetObjectOutput, error) {
	if versionID == "" {
		return nil, nil, fmt.Errorf("%w: version_id is required", ErrValidation)
	}
	if err := s.permissions.EnsureViewVersion(ctx, actorID, actorRole, versionID); err != nil {
		return nil, nil, err
	}

	ver, err := s.reader.GetVersion(ctx, versionID)
	if err != nil {
		return nil, nil, err
	}

	obj, err := s.storage.GetObject(ctx, ver.StorageObjectKey)
	if err != nil {
		return ver, nil, fmt.Errorf("storage get failed: %w", err)
	}

	return ver, obj, nil
}
