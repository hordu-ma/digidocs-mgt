package service

import (
	"context"
	"fmt"
	"io"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/repository"
	"digidocs-mgt/backend-go/internal/storage"
)

type DocumentService struct {
	reader   repository.DocumentReader
	writer   repository.DocumentWriter
	storage  storage.Provider
	workflow repository.VersionWorkflow
}

func NewDocumentService(
	reader repository.DocumentReader,
	writer repository.DocumentWriter,
	storage storage.Provider,
	workflow repository.VersionWorkflow,
) DocumentService {
	return DocumentService{
		reader:   reader,
		writer:   writer,
		storage:  storage,
		workflow: workflow,
	}
}

func (s DocumentService) ListDocuments(ctx context.Context, filter query.DocumentListFilter) ([]query.DocumentListItem, int, error) {
	return s.reader.ListDocuments(ctx, filter)
}

func (s DocumentService) GetDocument(ctx context.Context, documentID string) (*query.DocumentDetail, error) {
	return s.reader.GetDocument(ctx, documentID)
}

func (s DocumentService) UpdateDocument(ctx context.Context, input command.DocumentUpdateInput) (map[string]any, error) {
	if input.DocumentID == "" {
		return nil, fmt.Errorf("%w: document_id is required", ErrValidation)
	}
	if input.Title == "" && input.Description == "" && input.FolderID == "" {
		return nil, fmt.Errorf("%w: at least one field (title, description, folder_id) must be provided", ErrValidation)
	}
	return s.writer.UpdateDocument(ctx, input)
}

func (s DocumentService) DeleteDocument(ctx context.Context, input command.DocumentDeleteInput) error {
	if input.DocumentID == "" {
		return fmt.Errorf("%w: document_id is required", ErrValidation)
	}
	return s.writer.DeleteDocument(ctx, input)
}

func (s DocumentService) RestoreDocument(ctx context.Context, documentID string, actorID string) error {
	if documentID == "" {
		return fmt.Errorf("%w: document_id is required", ErrValidation)
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

	docData, err := s.writer.CreateDocument(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("document creation failed: %w", err)
	}

	documentID, _ := docData["id"].(string)

	objectKey := fmt.Sprintf("documents/%s/%s", documentID, fileName)
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
