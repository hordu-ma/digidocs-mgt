package command

type FlowActionInput struct {
	DocumentID string
	Action     string
	Note       string
	ToUserID   string
}

type VersionCreateInput struct {
	DocumentID       string
	FileName         string
	FileSize         int64
	CommitMessage    string
	StorageObjectKey string
	StorageProvider  string
}

type HandoverCreateInput struct {
	TargetUserID   string
	ReceiverUserID string
	ProjectID      string
	Remark         string
}

type HandoverItemInput struct {
	DocumentID string
	Selected   bool
	Note       string
}

type HandoverItemUpdateInput struct {
	HandoverID string
	Items      []HandoverItemInput
}

type HandoverActionInput struct {
	HandoverID string
	Action     string
	Note       string
	Reason     string
}
