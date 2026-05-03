package protocol

type DocumentServerState struct {
    Revision       ID                 `json:"revision"`
    PendingChanges []OperationMessage `json:"pending_changes"`
    RevisionLog    []OperationMessage `json:"revision_log"`
    Document       DocumentRep        `json:"document"`
}

func NewDocumentServerState() *DocumentServerState {
    return &DocumentServerState{
        Revision:       0,
        PendingChanges: make([]OperationMessage, 0),
        RevisionLog:    make([]OperationMessage, 0),
        Document:       "",
    }
}
