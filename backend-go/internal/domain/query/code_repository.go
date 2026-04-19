package query

type CodeRepositoryItem struct {
	ID               string `json:"id"`
	TeamSpaceID      string `json:"team_space_id"`
	ProjectID        string `json:"project_id"`
	ProjectName      string `json:"project_name,omitempty"`
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	Description      string `json:"description,omitempty"`
	DefaultBranch    string `json:"default_branch"`
	TargetFolderPath string `json:"target_folder_path"`
	RepoStoragePath  string `json:"repo_storage_path,omitempty"`
	LastCommitSHA    string `json:"last_commit_sha,omitempty"`
	LastPushedAt     string `json:"last_pushed_at,omitempty"`
	Status           string `json:"status"`
	CreatedByName    string `json:"created_by_name,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type CodeRepositoryDetail struct {
	CodeRepositoryItem
	RemoteURL string `json:"remote_url,omitempty"`
	PushToken string `json:"push_token,omitempty"`
}

type CodeRepositoryListFilter struct {
	ProjectID string
	Keyword   string
	Page      int
	PageSize  int
}

type CodePushEventItem struct {
	ID            string `json:"id"`
	RepositoryID  string `json:"repository_id"`
	Branch        string `json:"branch"`
	BeforeSHA     string `json:"before_sha,omitempty"`
	AfterSHA      string `json:"after_sha,omitempty"`
	CommitMessage string `json:"commit_message,omitempty"`
	PusherName    string `json:"pusher_name,omitempty"`
	SyncStatus    string `json:"sync_status"`
	ErrorMessage  string `json:"error_message,omitempty"`
	CreatedAt     string `json:"created_at"`
	CompletedAt   string `json:"completed_at,omitempty"`
}
