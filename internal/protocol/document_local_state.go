package protocol

type DocumentLocalState struct {
	LastSyncVersion ID                 `json:"last_sync_version"`
	PendingChanges  []OperationMessage `json:"pending_changes"`
	SentChanges     []OperationMessage `json:"sent_changes"`
	Document        DocumentRep        `json:"document"`
}

func NewDocumentLocalState() *DocumentLocalState {
    return &DocumentLocalState{
        LastSyncVersion: 0,
        PendingChanges:  make([]OperationMessage, 0),
        SentChanges:     make([]OperationMessage, 0),
        Document:        "",
    }
}
