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

type DocumentService struct {
	reader      repository.DocumentReader
	writer      repository.DocumentWriter
	storage     storage.Provider
	workflow    repository.VersionWorkflow
	permissions PermissionService
}

func NewDocumentService(
	reader repository.DocumentReader,
	writer repository.DocumentWriter,
	storage storage.Provider,
	workflow repository.VersionWorkflow,
	permissions ...PermissionService,
) DocumentService {
	permissionService := PermissionService{}
	if len(permissions) > 0 {
		permissionService = permissions[0]
	}
	return DocumentService{
		reader:      reader,
		writer:      writer,
		storage:     storage,
		workflow:    workflow,
		permissions: permissionService,
	}
}

func (s DocumentService) ListDocuments(ctx context.Context, filter query.DocumentListFilter) ([]query.DocumentListItem, int, error) {
	return s.reader.ListDocuments(ctx, filter)
}

func (s DocumentService) CountDocumentsByProject(ctx context.Context, filter query.DocumentListFilter) ([]query.ProjectDocumentCount, error) {
	return s.reader.CountDocumentsByProject(ctx, filter)
}

func (s DocumentService) GetDocument(ctx context.Context, documentID string, actorID string, actorRole string) (*query.DocumentDetail, error) {
	if err := s.permissions.EnsureViewDocument(ctx, actorID, actorRole, documentID); err != nil {
		return nil, err
	}
	item, err := s.reader.GetDocument(ctx, documentID)
	if err != nil {
		return nil, err
	}
	item.Actions = s.flowActionStates(ctx, item, actorID, actorRole)
	return item, nil
}

// flowActionStates evaluates every action valid in the document's status
// against the caller's permission, using the same checks the flow endpoint
// applies, so an enabled button never ends in a 403.
func (s DocumentService) flowActionStates(ctx context.Context, item *query.DocumentDetail, actorID, actorRole string) []query.FlowActionState {
	states := make([]query.FlowActionState, 0, 3)
	for _, action := range flowActionOrder {
		if !ValidFlowTransition(item.CurrentStatus, action) {
			continue
		}
		state := query.FlowActionState{Action: action, Allowed: true}
		if err := s.permissions.EnsureFlowDocument(ctx, actorID, actorRole, item.ID, action); err != nil {
			state.Allowed = false
			state.Reason = flowDeniedReason(action)
		}
		states = append(states, state)
	}
	return states
}

func (s DocumentService) UpdateDocument(ctx context.Context, input command.DocumentUpdateInput) (map[string]any, error) {
	if input.DocumentID == "" {
		return nil, fmt.Errorf("%w: document_id is required", ErrValidation)
	}
	if input.Title == "" && input.Description == "" && input.FolderID == "" {
		return nil, fmt.Errorf("%w: at least one field (title, description, folder_id) must be provided", ErrValidation)
	}
	if err := s.permissions.EnsureUpdateDocument(ctx, input.ActorID, input.ActorRole, input.DocumentID); err != nil {
		return nil, err
	}
	return s.writer.UpdateDocument(ctx, input)
}

func (s DocumentService) DeleteDocument(ctx context.Context, input command.DocumentDeleteInput) error {
	if input.DocumentID == "" {
		return fmt.Errorf("%w: document_id is required", ErrValidation)
	}
	if err := s.permissions.EnsureDeleteDocument(ctx, input.ActorID, input.ActorRole, input.DocumentID); err != nil {
		return err
	}
	return s.writer.DeleteDocument(ctx, input)
}

func (s DocumentService) RestoreDocument(ctx context.Context, documentID string, actorID string, actorRoles ...string) error {
	if documentID == "" {
		return fmt.Errorf("%w: document_id is required", ErrValidation)
	}
	actorRole := ""
	if len(actorRoles) > 0 {
		actorRole = actorRoles[0]
	}
	if err := s.permissions.EnsureDeleteDocument(ctx, actorID, actorRole, documentID); err != nil {
		return err
	}
	return s.writer.RestoreDocument(ctx, documentID, actorID)
}

func (s DocumentService) CreateWithFirstVersion(
	ctx context.Context,
	input command.DocumentCreateInput,
	fileName string,
	fileSize int64,
	commitMessage string,
	file io.Reader,
) (map[string]any, error) {
	if input.TeamSpaceID == "" {
		return nil, fmt.Errorf("%w: team_space_id is required", ErrValidation)
	}
	if input.ProjectID == "" {
		return nil, fmt.Errorf("%w: project_id is required", ErrValidation)
	}
	if input.Title == "" {
		return nil, fmt.Errorf("%w: title is required", ErrValidation)
	}
	if input.CurrentOwnerID == "" {
		return nil, fmt.Errorf("%w: current_owner_id is required", ErrValidation)
	}
	if input.ActorID == "" {
		return nil, fmt.Errorf("%w: actor_id is required", ErrValidation)
	}
	if fileName == "" {
		return nil, fmt.Errorf("%w: file is required", ErrValidation)
	}
	if err := s.permissions.EnsureCreateDocument(ctx, input.ActorID, input.ActorRole, input.ProjectID); err != nil {
		return nil, err
	}

	docData, err := s.writer.CreateDocument(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("document creation failed: %w", err)
	}

	documentID, _ := docData["id"].(string)

	objectKey := fmt.Sprintf("documents/%s/%s", documentID, shared.SanitizeStoredFileName(fileName))
	uploadResult, err := s.storage.PutObject(ctx, storage.PutObjectInput{
		ObjectKey:   objectKey,
		Reader:      file,
		CreatePaths: true,
	})
	if err != nil {
		return nil, fmt.Errorf("file upload failed: %w", err)
	}

	versionData, err := s.workflow.CreateUploadedVersion(ctx, command.VersionCreateInput{
		DocumentID:       documentID,
		FileName:         fileName,
		FileSize:         fileSize,
		CommitMessage:    commitMessage,
		StorageObjectKey: uploadResult.ObjectKey,
		StorageProvider:  uploadResult.Provider,
		ActorID:          input.ActorID,
	})
	if err != nil {
		return nil, fmt.Errorf("first version creation failed: %w", err)
	}

	docData["current_owner"] = map[string]string{
		"id": input.CurrentOwnerID,
	}
	docData["current_version"] = map[string]any{
		"id":         versionData["id"],
		"version_no": versionData["version_no"],
	}
	return docData, nil
}
