package query

type UserSummary struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
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
	ID               string `json:"id"`
	Title            string `json:"title"`
	Description      string `json:"description,omitempty"`
	CurrentStatus    string `json:"current_status"`
	CurrentOwnerID   string `json:"current_owner_id"`
	CurrentVersionID string `json:"current_version_id,omitempty"`
	IsArchived       bool   `json:"is_archived"`
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
