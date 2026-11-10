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
	ListHandovers(ctx context.Context, scope query.ActorScope) ([]query.HandoverItem, error)
	GetHandover(ctx context.Context, handoverID string) (*query.HandoverItem, error)
}

type AuditReader interface {
	ListAuditEvents(ctx context.Context, filter query.AuditEventFilter) ([]query.AuditEventItem, int, error)
	GetAuditSummary(ctx context.Context, projectID string, scope query.ActorScope) (query.AuditSummary, error)
}

// AuditWriter persists access events (view / preview / download) that are not
// a side effect of a transactional business write. Business actions such as
// flow transitions keep writing their audit rows inside their own transaction.
type AuditWriter interface {
	RecordAuditEvent(ctx context.Context, event command.AuditEventInput) error
}

// NotificationRepository stores in-app notifications and resolves the context
// needed to word them. ListInbox computes "waiting for me" tasks live from the
// business tables; nothing about the inbox is stored.
type NotificationRepository interface {
	CreateNotifications(ctx context.Context, input command.NotificationInput) error
	ListNotifications(ctx context.Context, userID string, unreadOnly bool, page int, pageSize int) ([]query.NotificationItem, int, error)
	CountUnread(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID string, notificationID string) error
	MarkAllRead(ctx context.Context, userID string) (int, error)
	ListInbox(ctx context.Context, userID string, isAdmin bool) ([]query.InboxItem, error)

	GetUserContacts(ctx context.Context, userIDs []string) ([]query.UserContact, error)
	GetDocumentBrief(ctx context.Context, documentID string) (*query.DocumentBrief, error)
	GetHandoverBrief(ctx context.Context, handoverID string) (*query.HandoverBrief, error)
	GetLastTransferSender(ctx context.Context, documentID string) (string, error)
	GetAssistantRequestBrief(ctx context.Context, requestID string) (*query.AssistantRequestBrief, error)
	GetProjectManagers(ctx context.Context, projectID string) ([]string, string, error)
}

type DashboardReader interface {
	GetOverview(ctx context.Context, projectID string, scope query.ActorScope) (query.DashboardOverview, error)
	ListRecentFlows(ctx context.Context, projectID string, scope query.ActorScope) ([]query.RecentFlowItem, error)
	ListRiskDocuments(ctx context.Context, projectID string, scope query.ActorScope) ([]query.RiskDocumentItem, error)
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
	CanViewDocument(ctx context.Context, actorID string, actorRole string, documentID string) (bool, error)
	CanViewVersion(ctx context.Context, actorID string, actorRole string, versionID string) (bool, error)
	CanViewHandover(ctx context.Context, actorID string, actorRole string, handoverID string) (bool, error)
	CanViewProject(ctx context.Context, actorID string, actorRole string, projectID string) (bool, error)
	CanViewDataAsset(ctx context.Context, actorID string, actorRole string, dataAssetID string) (bool, error)
	CanViewCodeRepository(ctx context.Context, actorID string, actorRole string, repositoryID string) (bool, error)
	CanCreateCodeRepository(ctx context.Context, actorID string, actorRole string, projectID string) (bool, error)
	CanManageCodeRepository(ctx context.Context, actorID string, actorRole string, repositoryID string) (bool, error)
	CanPushCodeRepository(ctx context.Context, actorID string, actorRole string, repositoryID string) (bool, error)
	CanViewProjectMembers(ctx context.Context, actorID string, actorRole string, projectID string) (bool, error)
	CanManageProjectMembers(ctx context.Context, actorID string, actorRole string, projectID string) (bool, error)
	CanViewAssistantRequest(ctx context.Context, actorID string, actorRole string, requestID string) (bool, error)
	CanViewAssistantSuggestion(ctx context.Context, actorID string, actorRole string, suggestionID string) (bool, error)
	CanViewConversation(ctx context.Context, actorID string, actorRole string, conversationID string) (bool, error)
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
	FindUserByIdentifier(ctx context.Context, identifier string) (*auth.UserRecord, error)
	RegisterUser(ctx context.Context, input auth.RegisterUserInput) (*auth.UserRecord, error)
	GetUserProfile(ctx context.Context, userID string) (*auth.UserProfile, error)
	UpdateUserProfile(ctx context.Context, userID string, input auth.ProfileUpdateInput) (*auth.UserProfile, error)
	GetUserCredentials(ctx context.Context, userID string) (*auth.UserRecord, error)
	UpdateUserPassword(ctx context.Context, userID string, passwordHash string) error
}

// ========== Data Asset ==========

type DataAssetReader interface {
	GetDataFolder(ctx context.Context, id string) (*query.DataFolderItem, error)
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

type CodeRepositoryReader interface {
	ListCodeRepositories(ctx context.Context, filter query.CodeRepositoryListFilter) ([]query.CodeRepositoryItem, int, error)
	GetCodeRepository(ctx context.Context, id string) (*query.CodeRepositoryDetail, error)
	GetCodeRepositoryBySlug(ctx context.Context, slug string) (*query.CodeRepositoryDetail, error)
	ListCodePushEvents(ctx context.Context, repositoryID string) ([]query.CodePushEventItem, error)
}

type CodeRepositoryWriter interface {
	CreateCodeRepository(ctx context.Context, input command.CodeRepositoryCreateInput) (*query.CodeRepositoryDetail, error)
	UpdateCodeRepository(ctx context.Context, input command.CodeRepositoryUpdateInput) (*query.CodeRepositoryDetail, error)
	CreateCodePushEvent(ctx context.Context, input command.CodePushEventCreateInput) (*query.CodePushEventItem, error)
	UpdateCodeRepositoryAfterPush(ctx context.Context, repositoryID string, commitSHA string, status string) error
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
	GetDocumentVersionExtractedText(ctx context.Context, documentID, versionID string) (string, error)
	ListSuggestions(ctx context.Context, filter query.AssistantSuggestionFilter) ([]query.AssistantSuggestionItem, error)
	ConfirmSuggestion(ctx context.Context, suggestionID string, actorID string, note string) (map[string]any, error)
	GetSuggestion(ctx context.Context, suggestionID string) (*query.AssistantSuggestionItem, error)
	UpdateRequestProgress(ctx context.Context, requestID string, partialAnswer string) error
	GetRequestCreator(ctx context.Context, requestID string) (string, error)
	DismissSuggestion(ctx context.Context, suggestionID string, actorID string, reason string) (map[string]any, error)
}
