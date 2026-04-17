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
}

type AuditSummary struct {
	ProjectID      string            `json:"project_id,omitempty"`
	DownloadCount  int               `json:"download_count"`
	UploadCount    int               `json:"upload_count"`
	TransferCount  int               `json:"transfer_count"`
	ArchiveCount   int               `json:"archive_count"`
	TopActiveUsers []AuditUserMetric `json:"top_active_users"`
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
	DocumentID string `json:"document_id"`
	Selected   bool   `json:"selected"`
	Note       string `json:"note,omitempty"`
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

type RiskDocumentItem struct {
	DocumentID  string `json:"document_id"`
	Title       string `json:"title"`
	RiskType    string `json:"risk_type"`
	RiskMessage string `json:"risk_message"`
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
}

type DocumentDetail struct {
	ID               string       `json:"id"`
	Title            string       `json:"title"`
	Description      string       `json:"description,omitempty"`
	CurrentStatus    string       `json:"current_status"`
	CurrentOwner     *UserSummary `json:"current_owner,omitempty"`
	CurrentVersionID string       `json:"current_version_id,omitempty"`
	IsArchived       bool         `json:"is_archived"`
}

type DocumentListFilter struct {
	TeamSpaceID     string
	ProjectID       string
	FolderID        string
	OwnerID         string
	Status          string
	Keyword         string
	IncludeArchived bool
	Page            int
	PageSize        int
}

type PaginationMeta struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
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
}

type AssistantSuggestionFilter struct {
	RelatedType    string
	RelatedID      string
	Status         string
	SuggestionType string
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
	ID            string         `json:"id"`
	ScopeType     string         `json:"scope_type"`
	ScopeID       string         `json:"scope_id"`
	SourceScope   map[string]any `json:"source_scope,omitempty"`
	Title         string         `json:"title,omitempty"`
	CreatedBy     string         `json:"created_by,omitempty"`
	CreatedAt     string         `json:"created_at"`
	LastMessageAt string         `json:"last_message_at,omitempty"`
	ArchivedAt    string         `json:"archived_at,omitempty"`
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
