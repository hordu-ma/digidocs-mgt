package command

type FlowActionInput struct {
	DocumentID string
	Action     string
	Note       string
	ToUserID   string
	ActorID    string
}

type VersionCreateInput struct {
	DocumentID       string
	FileName         string
	FileSize         int64
	CommitMessage    string
	StorageObjectKey string
	StorageProvider  string
	ActorID          string
}

type HandoverCreateInput struct {
	TargetUserID   string
	ReceiverUserID string
	ProjectID      string
	Remark         string
	ActorID        string
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
}

type HandoverActionInput struct {
	HandoverID string
	Action     string
	Note       string
	Reason     string
	ActorID    string
}
