package command

type DocumentCreateInput struct {
	TeamSpaceID    string
	ProjectID      string
	FolderID       string
	Title          string
	Description    string
	CurrentOwnerID string
	ActorID        string
	ActorRole      string
}

type DocumentUpdateInput struct {
	DocumentID  string
	Title       string
	Description string
	FolderID    string
	ActorID     string
	ActorRole   string
}

type DocumentDeleteInput struct {
	DocumentID string
	Reason     string
	ActorID    string
	ActorRole  string
}

type FlowActionInput struct {
	DocumentID string
	Action     string
	Note       string
	ToUserID   string
	ActorID    string
	ActorRole  string
}

type VersionCreateInput struct {
	DocumentID       string
	FileName         string
	FileSize         int64
	CommitMessage    string
	StorageObjectKey string
	StorageProvider  string
	ActorID          string
	ActorRole        string
}

type HandoverCreateInput struct {
	TargetUserID   string
	ReceiverUserID string
	ProjectID      string
	Remark         string
	ActorID        string
	ActorRole      string
}

type HandoverItemInput struct {
	DocumentID string
	Selected   bool
	Note       string
}

type HandoverItemUpdateInput struct {
	HandoverID string
	Items      []HandoverItemInput
	ActorID    string
	ActorRole  string
}

type HandoverActionInput struct {
	HandoverID string
	Action     string
	Note       string
	Reason     string
	ActorID    string
	ActorRole  string
}

// ========== Data Asset ==========

type DataFolderCreateInput struct {
	ProjectID string
	ParentID  string
	Name      string
	ActorID   string
	ActorRole string
}

type DataAssetCreateInput struct {
	TeamSpaceID          string
	ProjectID            string
	FolderID             string
	DisplayName          string
	FileName             string
	Description          string
	FileSize             int64
	MimeType             string
	StorageProvider      string
	StorageBucketOrShare string
	StorageObjectKey     string
	ActorID              string
	ActorRole            string
}

type DataAssetUpdateInput struct {
	DataAssetID string
	DisplayName string
	Description string
	FolderID    string
	ActorID     string
	ActorRole   string
}

type DataAssetDeleteInput struct {
	DataAssetID string
	ActorID     string
	ActorRole   string
}

type HandoverDataItemInput struct {
	DataAssetID string
	Selected    bool
	Note        string
}

type HandoverDataItemUpdateInput struct {
	HandoverID string
	Items      []HandoverDataItemInput
	ActorID    string
	ActorRole  string
}
