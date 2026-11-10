package query

type UserSummary struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type UserOption struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	Email       string `json:"email,omitempty"`
	Phone       string `json:"phone,omitempty"`
	Wechat      string `json:"wechat,omitempty"`
	Status      string `json:"status,omitempty"`
}

type TeamSpaceSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

type ProjectSummary struct {
	ID          string      `json:"id"`
	TeamSpaceID string      `json:"team_space_id"`
	Name        string      `json:"name"`
	Code        string      `json:"code"`
	Owner       UserSummary `json:"owner"`
	MyRole      string      `json:"my_role"`
}

type FolderTreeNode struct {
	ID       string           `json:"id"`
	Name     string           `json:"name"`
	Path     string           `json:"path"`
	Children []FolderTreeNode `json:"children"`
}

type VersionItem struct {
	ID            string `json:"id"`
	VersionNo     int    `json:"version_no"`
	FileName      string `json:"file_name"`
	SummaryStatus string `json:"summary_status"`
	CreatedAt     string `json:"created_at"`
}

type VersionDetail struct {
	ID               string `json:"id"`
	DocumentID       string `json:"document_id,omitempty"`
	VersionNo        int    `json:"version_no,omitempty"`
	CommitMessage    string `json:"commit_message,omitempty"`
	FileName         string `json:"file_name,omitempty"`
	FileSize         int64  `json:"file_size,omitempty"`
	StorageProvider  string `json:"storage_provider,omitempty"`
	StorageObjectKey string `json:"storage_object_key,omitempty"`
	MimeType         string `json:"mime_type,omitempty"`
	Download         string `json:"download,omitempty"`
	PreviewType      string `json:"preview_type,omitempty"`
	PreviewURL       string `json:"preview_url,omitempty"`
	WatermarkEnabled bool   `json:"watermark_enabled,omitempty"`
}

type FlowItem struct {
	ID         string `json:"id"`
	Action     string `json:"action"`
	FromStatus string `json:"from_status,omitempty"`
	ToStatus   string `json:"to_status"`
	CreatedAt  string `json:"created_at"`
}

type AuditEventItem struct {
	ID           string `json:"id"`
	DocumentID   string `json:"document_id,omitempty"`
	VersionID    string `json:"version_id,omitempty"`
	UserID       string `json:"user_id,omitempty"`
	ActionType   string `json:"action_type"`
	RequestID    string `json:"request_id,omitempty"`
	IPAddress    string `json:"ip_address,omitempty"`
	TerminalInfo string `json:"terminal_info,omitempty"`
	CreatedAt    string `json:"created_at"`
	// Display fields joined for the document access panel.
	UserDisplayName string `json:"user_display_name,omitempty"`
	DocumentTitle   string `json:"document_title,omitempty"`
}

type AuditSummary struct {
	ProjectID     string `json:"project_id,omitempty"`
	DownloadCount int    `json:"download_count"`
	UploadCount   int    `json:"upload_count"`
	TransferCount int    `json:"transfer_count"`
	ArchiveCount  int    `json:"archive_count"`
	// ViewCount / PreviewCount / RecentDownloadCount come from access events
	// recorded since 2026-11 (migration-free: audit_events.action_type is text).
	ViewCount           int                   `json:"view_count"`
	PreviewCount        int                   `json:"preview_count"`
	RecentDownloadCount int                   `json:"recent_download_count"`
	TopActiveUsers      []AuditUserMetric     `json:"top_active_users"`
	TopDownloads        []AuditDownloadMetric `json:"top_downloads"`
}

// AuditDownloadMetric is one row of the 30-day download ranking.
type AuditDownloadMetric struct {
	DocumentID       string `json:"document_id"`
	Title            string `json:"title"`
	DownloadCount    int    `json:"download_count"`
	UserCount        int    `json:"user_count"`
	LastDownloadedAt string `json:"last_downloaded_at"`
}

type AuditUserMetric struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name,omitempty"`
	ActionCount int    `json:"action_count"`
}

type HandoverItem struct {
	ID             string         `json:"id"`
	TargetUserID   string         `json:"target_user_id,omitempty"`
	ReceiverUserID string         `json:"receiver_user_id,omitempty"`
	ProjectID      string         `json:"project_id,omitempty"`
	Status         string         `json:"status,omitempty"`
	Remark         string         `json:"remark,omitempty"`
	Items          []HandoverLine `json:"items,omitempty"`
}

type HandoverLine struct {
	DocumentID    string `json:"document_id"`
	Title         string `json:"title,omitempty"`
	CurrentStatus string `json:"current_status,omitempty"`
	Selected      bool   `json:"selected"`
	Note          string `json:"note,omitempty"`
}

type DashboardOverview struct {
	DocumentTotal        int            `json:"document_total"`
	StatusCounts         map[string]int `json:"status_counts"`
	HandoverPendingCount int            `json:"handover_pending_count"`
	RiskDocumentCount    int            `json:"risk_document_count"`
}

type RecentFlowItem struct {
	DocumentID string `json:"document_id"`
	Title      string `json:"title"`
	Action     string `json:"action"`
	FromStatus string `json:"from_status,omitempty"`
	ToStatus   string `json:"to_status"`
	CreatedAt  string `json:"created_at"`
}

// RiskDocumentItem is one dashboard risk. Despite the name it can point at a
// handover (risk_type handover_overdue): document_id is then empty and
// related_type/related_id/link identify the handover.
type RiskDocumentItem struct {
	DocumentID  string `json:"document_id"`
	Title       string `json:"title"`
	RiskType    string `json:"risk_type"`
	RiskMessage string `json:"risk_message"`
	RelatedType string `json:"related_type"`
	RelatedID   string `json:"related_id"`
	Link        string `json:"link"`
}

type DocumentListItem struct {
	ID               string       `json:"id"`
	Title            string       `json:"title"`
	ProjectName      string       `json:"project_name,omitempty"`
	FolderPath       string       `json:"folder_path,omitempty"`
	CurrentStatus    string       `json:"current_status"`
	CurrentOwner     *UserSummary `json:"current_owner,omitempty"`
	CurrentVersionNo *int         `json:"current_version_no,omitempty"`
	UpdatedAt        *string      `json:"updated_at,omitempty"`
	ProjectID        string       `json:"project_id,omitempty"`
	IsArchived       bool         `json:"is_archived"`
	IsDeleted        bool         `json:"is_deleted"`
}

// ProjectDocumentCount is one entry of the document list's per-project counts.
type ProjectDocumentCount struct {
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`
	Count       int    `json:"count"`
}

type DocumentDetail struct {
	ProjectID        string       `json:"project_id"`
	ID               string       `json:"id"`
	Title            string       `json:"title"`
	Description      string       `json:"description,omitempty"`
	CurrentStatus    string       `json:"current_status"`
	CurrentOwner     *UserSummary `json:"current_owner,omitempty"`
	CurrentVersionID string       `json:"current_version_id,omitempty"`
	IsArchived       bool         `json:"is_archived"`
	// 2026-11-10: fields the detail page displayed but never received.
	ProjectName      string `json:"project_name,omitempty"`
	FolderID         string `json:"folder_id,omitempty"`
	FolderPath       string `json:"folder_path,omitempty"`
	CurrentVersionNo *int   `json:"current_version_no,omitempty"`
	CreatedAt        string `json:"created_at,omitempty"`
	UpdatedAt        string `json:"updated_at,omitempty"`
	// Actions lists the flow actions valid in the current status, and whether
	// the caller may run each; the UI shows denied ones disabled with Reason.
	Actions []FlowActionState `json:"actions"`
}

type FlowActionState struct {
	Action  string `json:"action"`
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

type DocumentListFilter struct {
	TeamSpaceID     string
	ProjectID       string
	FolderID        string
	OwnerID         string
	Status          string
	Keyword         string
	IncludeArchived bool
	// 2026-11-10: server-side sorting/filters previously done in the browser.
	Sort              string // updated_desc | updated_asc | title_asc | owner_asc
	UpdatedWithinDays int
	// DeletedOnly lists the recycle bin (project owners/managers and admins only).
	DeletedOnly bool
	Page        int
	PageSize    int
	// ActorID/ActorRole scope the result to projects the actor can see.
	// Non-admin actors only see documents in projects they own or are a
	// member of; admin actors see everything. Set by the handler from the
	// authenticated request context, never from client input.
	ActorID   string
	ActorRole string
}

type PaginationMeta struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

// ActorScope narrows an aggregate/list query to the projects the acting user is
// allowed to see. Aggregate endpoints (dashboard, audit log, handover list) have
// no single resource id to authorize against, so instead of a per-row permission
// check they push the actor down into the query and let the repository filter.
// IsAdmin bypasses the narrowing entirely.
type ActorScope struct {
	ActorID string
	IsAdmin bool
}

type AuditEventFilter struct {
	ProjectID  string
	DocumentID string
	UserID     string
	ActionType string
	DateFrom   string
	DateTo     string
	Page       int
	PageSize   int
	Scope      ActorScope
}

type ProjectMemberItem struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
	Username    string `json:"username"`
	// Account is the username, or a masked phone/email for self-registered users.
	Account     string `json:"account"`
	ProjectRole string `json:"project_role"`
}

type ProjectInviteItem struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	Token       string `json:"token"`
	ProjectRole string `json:"project_role"`
	Status      string `json:"status"`
	ExpiresAt   string `json:"expires_at"`
	CreatedAt   string `json:"created_at"`
}

type ProjectInvitePreview struct {
	ProjectID     string `json:"project_id"`
	ProjectName   string `json:"project_name"`
	TeamSpaceName string `json:"team_space_name"`
	ProjectRole   string `json:"project_role"`
	Status        string `json:"status"`
	ExpiresAt     string `json:"expires_at"`
	Expired       bool   `json:"expired"`
}

type AssistantSuggestionItem struct {
	ID             string   `json:"id"`
	RelatedType    string   `json:"related_type"`
	RelatedID      string   `json:"related_id"`
	SuggestionType string   `json:"suggestion_type"`
	Status         string   `json:"status"`
	Title          string   `json:"title,omitempty"`
	Content        string   `json:"content"`
	SourceScope    string   `json:"source_scope,omitempty"`
	Confidence     *float64 `json:"confidence,omitempty"`
	RequestID      string   `json:"request_id,omitempty"`
	GeneratedAt    string   `json:"generated_at"`
	// SuggestedAction is an optional whitelisted flow action (mark_in_progress,
	// finalize, archive) the user can run with "确认并执行". Never executed by AI.
	SuggestedAction string `json:"suggested_action,omitempty"`
}

type AssistantSuggestionFilter struct {
	SuggestionID   string
	RelatedType    string
	RelatedID      string
	Status         string
	SuggestionType string
	// Scope narrows the result set to the acting user's own projects, the same
	// way ActorScope does for audit events/handovers. Set by handlers; the
	// memory backend ignores it (see repository/memory precedent).
	Scope ActorScope
}

type AssistantRequestFilter struct {
	RequestType    string
	RelatedType    string
	RelatedID      string
	ConversationID string
	Status         string
	Keyword        string
	Page           int
	PageSize       int
	// Scope narrows the result set to requests the actor created, or that
	// belong to a project they are a member of. See AssistantSuggestionFilter.Scope.
	Scope ActorScope
}

type AssistantRequestItem struct {
	ID                   string           `json:"id"`
	RequestType          string           `json:"request_type"`
	RelatedType          string           `json:"related_type,omitempty"`
	RelatedID            string           `json:"related_id,omitempty"`
	ConversationID       string           `json:"conversation_id,omitempty"`
	Status               string           `json:"status"`
	Question             string           `json:"question,omitempty"`
	SourceScope          map[string]any   `json:"source_scope,omitempty"`
	MemorySources        []map[string]any `json:"memory_sources,omitempty"`
	ErrorMessage         string           `json:"error_message,omitempty"`
	Output               map[string]any   `json:"output,omitempty"`
	SkillName            string           `json:"skill_name,omitempty"`
	SkillVersion         string           `json:"skill_version,omitempty"`
	Model                string           `json:"model,omitempty"`
	UpstreamRequestID    string           `json:"upstream_request_id,omitempty"`
	Usage                map[string]any   `json:"usage,omitempty"`
	CreatedAt            string           `json:"created_at"`
	CompletedAt          string           `json:"completed_at,omitempty"`
	ProcessingDurationMs int64            `json:"processing_duration_ms,omitempty"`
}

type AssistantConversationFilter struct {
	ScopeType       string
	ScopeID         string
	CreatedBy       string
	IncludeArchived bool
}

type AssistantConversationItem struct {
	ID               string         `json:"id"`
	ScopeType        string         `json:"scope_type"`
	ScopeID          string         `json:"scope_id"`
	SourceScope      map[string]any `json:"source_scope,omitempty"`
	ScopeDisplayName string         `json:"scope_display_name,omitempty"`
	Title            string         `json:"title,omitempty"`
	CreatedBy        string         `json:"created_by,omitempty"`
	CreatedAt        string         `json:"created_at"`
	LastMessageAt    string         `json:"last_message_at,omitempty"`
	ArchivedAt       string         `json:"archived_at,omitempty"`
}

type AssistantConversationMessageItem struct {
	ID             string         `json:"id"`
	ConversationID string         `json:"conversation_id"`
	Role           string         `json:"role"`
	Content        string         `json:"content"`
	RequestID      string         `json:"request_id,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	CreatedBy      string         `json:"created_by,omitempty"`
	CreatedAt      string         `json:"created_at"`
}

// ========== Data Asset ==========

type DataFolderItem struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	ParentID  string `json:"parent_id,omitempty"`
	Depth     int    `json:"depth"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

type DataAssetListItem struct {
	ID            string `json:"id"`
	ProjectID     string `json:"project_id"`
	ProjectName   string `json:"project_name,omitempty"`
	FolderID      string `json:"folder_id,omitempty"`
	FolderName    string `json:"folder_name,omitempty"`
	DisplayName   string `json:"display_name"`
	FileName      string `json:"file_name"`
	MimeType      string `json:"mime_type,omitempty"`
	FileSize      int64  `json:"file_size"`
	CreatedByName string `json:"created_by_name,omitempty"`
	CreatedAt     string `json:"created_at"`
}

type DataAssetDetail struct {
	ID               string `json:"id"`
	TeamSpaceID      string `json:"team_space_id"`
	ProjectID        string `json:"project_id"`
	ProjectName      string `json:"project_name,omitempty"`
	FolderID         string `json:"folder_id,omitempty"`
	FolderName       string `json:"folder_name,omitempty"`
	DisplayName      string `json:"display_name"`
	FileName         string `json:"file_name"`
	Description      string `json:"description,omitempty"`
	MimeType         string `json:"mime_type,omitempty"`
	FileSize         int64  `json:"file_size"`
	StorageProvider  string `json:"storage_provider"`
	StorageObjectKey string `json:"storage_object_key,omitempty"`
	CreatedByName    string `json:"created_by_name,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type DataAssetListFilter struct {
	ProjectID string
	FolderID  string
	Keyword   string
	Page      int
	PageSize  int
	// ActorID/ActorRole scope the result to projects the actor can see.
	// Non-admin actors only see data assets in projects they own or are a
	// member of; admin actors see everything. Set by the handler from the
	// authenticated request context, never from client input.
	ActorID   string
	ActorRole string
}

type HandoverDataLine struct {
	DataAssetID string `json:"data_asset_id"`
	DisplayName string `json:"display_name,omitempty"`
	FileName    string `json:"file_name,omitempty"`
	Selected    bool   `json:"selected"`
	Note        string `json:"note,omitempty"`
}

// NotificationItem is one in-app notification (migration 013).
type NotificationItem struct {
	ID          string `json:"id"`
	EventType   string `json:"event_type"`
	Title       string `json:"title"`
	Body        string `json:"body,omitempty"`
	Link        string `json:"link,omitempty"`
	RelatedType string `json:"related_type,omitempty"`
	RelatedID   string `json:"related_id,omitempty"`
	ActorName   string `json:"actor_name,omitempty"`
	Read        bool   `json:"read"`
	CreatedAt   string `json:"created_at"`
}

// InboxItem is one "waiting for me" task, computed live from business tables.
// Kind values: accept_transfer, confirm_handover, complete_handover,
// fill_handover_items, review_suggestion, stale_document.
type InboxItem struct {
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	Detail      string `json:"detail,omitempty"`
	Link        string `json:"link"`
	RelatedType string `json:"related_type"`
	RelatedID   string `json:"related_id"`
	ProjectName string `json:"project_name,omitempty"`
	Since       string `json:"since,omitempty"`
}

// UserContact is what external channels need to address a recipient.
type UserContact struct {
	ID          string
	DisplayName string
	Email       string
}

// DocumentBrief / HandoverBrief carry just enough context to word a notification.
type DocumentBrief struct {
	ID          string
	Title       string
	ProjectID   string
	ProjectName string
	OwnerID     string
}

type HandoverBrief struct {
	ID             string
	ProjectID      string
	ProjectName    string
	TargetUserID   string
	TargetName     string
	ReceiverUserID string
	ReceiverName   string
	GeneratedBy    string
	ManagerIDs     []string
}

type AssistantRequestBrief struct {
	ID          string
	RequestType string
	RelatedType string
	RelatedID   string
	CreatedBy   string
	Status      string
}

// TextSearchFilter drives lexical search over document_text_chunks.
// DocumentID narrows to one document; ProjectID to one project; Scope keeps
// non-admin callers inside projects they belong to.
type TextSearchFilter struct {
	Query      string
	ProjectID  string
	DocumentID string
	Scope      ActorScope
	Limit      int
}

// TextSearchHit is one matching chunk of a document's current version.
type TextSearchHit struct {
	DocumentID  string `json:"document_id"`
	Title       string `json:"title"`
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name,omitempty"`
	VersionID   string `json:"version_id"`
	VersionNo   int    `json:"version_no"`
	ChunkNo     int    `json:"chunk_no"`
	Content     string `json:"content,omitempty"`
	Snippet     string `json:"snippet"`
	Score       int    `json:"score"`
	// MatchedChunks is set by document-level search: how many passages matched.
	MatchedChunks int `json:"matched_chunks,omitempty"`
	// Similarity is the cosine similarity for semantic hits (0 for keyword-only).
	Similarity float64 `json:"similarity,omitempty"`
	// Match tells which retrieval found it: keyword, semantic or both.
	Match string `json:"match,omitempty"`
}

// ChunkForEmbedding is one chunk waiting for (re)embedding.
type ChunkForEmbedding struct {
	ID      string
	Title   string
	Content string
}

// DocumentText is the readable text of a document's current version, rebuilt
// from document_text_chunks. Available is false until extraction has run.
type DocumentText struct {
	DocumentID string `json:"document_id"`
	VersionID  string `json:"version_id,omitempty"`
	VersionNo  int    `json:"version_no,omitempty"`
	Text       string `json:"text"`
	Chunks     int    `json:"-"`
	Available  bool   `json:"available"`
	Truncated  bool   `json:"truncated"`
}
