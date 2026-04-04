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

type HandoverActionInput struct {
	HandoverID string
	Action     string
	Note       string
	Reason     string
}
