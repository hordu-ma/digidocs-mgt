package repository

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/auth"
	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/domain/task"
)

type TeamSpaceReader interface {
	ListTeamSpaces(ctx context.Context, actorID, actorRole string) ([]query.TeamSpaceSummary, error)
}

type UserReader interface {
	ListUsers(ctx context.Context) ([]query.UserOption, error)
}

type ProjectReader interface {
	ListProjects(ctx context.Context, teamSpaceID, actorID, actorRole string) ([]query.ProjectSummary, error)
	GetFolderTree(ctx context.Context, projectID string) ([]query.FolderTreeNode, error)
}

type VersionReader interface {
	ListVersions(ctx context.Context, documentID string) ([]query.VersionItem, error)
	GetVersion(ctx context.Context, versionID string) (*query.VersionDetail, error)
}

type VersionWriter interface {
	CreateVersion(ctx context.Context, input command.VersionCreateInput) (map[string]any, error)
}

type VersionWorkflow interface {
	CreateUploadedVersion(ctx context.Context, input command.VersionCreateInput) (map[string]any, error)
}

type FlowReader interface {
	ListFlows(ctx context.Context, documentID string) ([]query.FlowItem, error)
}

type HandoverReader interface {
	ListHandovers(ctx context.Context) ([]query.HandoverItem, error)
	GetHandover(ctx context.Context, handoverID string) (*query.HandoverItem, error)
}

type AuditReader interface {
	ListAuditEvents(ctx context.Context, filter query.AuditEventFilter) ([]query.AuditEventItem, int, error)
	GetAuditSummary(ctx context.Context, projectID string) (query.AuditSummary, error)
}

type DashboardReader interface {
	GetOverview(ctx context.Context, projectID string) (query.DashboardOverview, error)
	ListRecentFlows(ctx context.Context, projectID string) ([]query.RecentFlowItem, error)
	ListRiskDocuments(ctx context.Context, projectID string) ([]query.RiskDocumentItem, error)
}

type ActionWriter interface {
	CreateFlowRecord(ctx context.Context, input command.FlowActionInput) (map[string]any, error)
	CreateHandover(ctx context.Context, input command.HandoverCreateInput) (map[string]any, error)
	UpdateHandoverItems(ctx context.Context, input command.HandoverItemUpdateInput) (map[string]any, error)
	ApplyHandover(ctx context.Context, input command.HandoverActionInput) (map[string]any, error)
}

type PermissionReader interface {
	CanCreateDocument(ctx context.Context, actorID string, actorRole string, projectID string) (bool, error)
	CanUpdateDocument(ctx context.Context, actorID string, actorRole string, documentID string) (bool, error)
	CanDeleteDocument(ctx context.Context, actorID string, actorRole string, documentID string) (bool, error)
	CanUploadVersion(ctx context.Context, actorID string, actorRole string, documentID string) (bool, error)
	CanFlowDocument(ctx context.Context, actorID string, actorRole string, documentID string, action string) (bool, error)
	CanCreateHandover(ctx context.Context, actorID string, actorRole string, projectID string) (bool, error)
	CanUpdateHandoverItems(ctx context.Context, actorID string, actorRole string, handoverID string) (bool, error)
	CanApplyHandover(ctx context.Context, actorID string, actorRole string, handoverID string, action string) (bool, error)
	CanUploadDataAsset(ctx context.Context, actorID string, actorRole string, projectID string) (bool, error)
	CanManageDataAsset(ctx context.Context, actorID string, actorRole string, dataAssetID string) (bool, error)
}

type DocumentReader interface {
	ListDocuments(ctx context.Context, filter query.DocumentListFilter) ([]query.DocumentListItem, int, error)
	GetDocument(ctx context.Context, documentID string) (*query.DocumentDetail, error)
}

type DocumentWriter interface {
	CreateDocument(ctx context.Context, input command.DocumentCreateInput) (map[string]any, error)
	UpdateDocument(ctx context.Context, input command.DocumentUpdateInput) (map[string]any, error)
	DeleteDocument(ctx context.Context, input command.DocumentDeleteInput) error
	RestoreDocument(ctx context.Context, documentID string, actorID string) error
}

type UserAuthReader interface {
	FindUserByUsername(ctx context.Context, username string) (*auth.UserRecord, error)
	GetUserProfile(ctx context.Context, userID string) (*auth.UserProfile, error)
	UpdateUserProfile(ctx context.Context, userID string, input auth.ProfileUpdateInput) (*auth.UserProfile, error)
}

// ========== Data Asset ==========

type DataAssetReader interface {
	ListDataAssets(ctx context.Context, filter query.DataAssetListFilter) ([]query.DataAssetListItem, int, error)
	GetDataAsset(ctx context.Context, id string) (*query.DataAssetDetail, error)
	ListDataFolders(ctx context.Context, projectID string) ([]query.DataFolderItem, error)
	ListHandoverDataItems(ctx context.Context, handoverID string) ([]query.HandoverDataLine, error)
}

type DataAssetWriter interface {
	CreateDataAsset(ctx context.Context, input command.DataAssetCreateInput) (map[string]any, error)
	UpdateDataAsset(ctx context.Context, input command.DataAssetUpdateInput) error
	DeleteDataAsset(ctx context.Context, input command.DataAssetDeleteInput) error
	CreateDataFolder(ctx context.Context, input command.DataFolderCreateInput) (*query.DataFolderItem, error)
	DeleteDataFolder(ctx context.Context, id string) error
	UpdateHandoverDataItems(ctx context.Context, input command.HandoverDataItemUpdateInput) (map[string]any, error)
}

type AssistantRepository interface {
	CreateConversation(ctx context.Context, scopeType string, scopeID string, sourceScope map[string]any, title string, actorID string) (*query.AssistantConversationItem, error)
	GetConversation(ctx context.Context, conversationID string) (*query.AssistantConversationItem, error)
	ListConversations(ctx context.Context, filter query.AssistantConversationFilter) ([]query.AssistantConversationItem, error)
	ArchiveConversation(ctx context.Context, conversationID string, archive bool) error
	ListConversationMessages(ctx context.Context, conversationID string) ([]query.AssistantConversationMessageItem, error)
	CreateAssistantRequest(ctx context.Context, message task.Message, actorID string) error
	CompleteAssistantRequest(ctx context.Context, result task.Result) error
	ListAssistantRequests(ctx context.Context, filter query.AssistantRequestFilter) ([]query.AssistantRequestItem, int, error)
	GetAssistantRequest(ctx context.Context, requestID string) (*query.AssistantRequestItem, error)
	GetLatestDocumentExtractedText(ctx context.Context, documentID string) (string, error)
	ListSuggestions(ctx context.Context, filter query.AssistantSuggestionFilter) ([]query.AssistantSuggestionItem, error)
	ConfirmSuggestion(ctx context.Context, suggestionID string, actorID string, note string) (map[string]any, error)
	DismissSuggestion(ctx context.Context, suggestionID string, actorID string, reason string) (map[string]any, error)
}
