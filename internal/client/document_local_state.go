package client
import "proyecto-final-redes/internal/protocol/operation"
import "proyecto-final-redes/internal/protocol/document"

type DocumentLocalState struct {
	LastSyncRevision document.ID
	PendingChanges  []operation.Operation 
	SentChanges     []operation.Operation 
	Document        document.DocumentRep  
}

func NewDocumentLocalState() *DocumentLocalState {
	return &DocumentLocalState{
		LastSyncRevision: 0,
		PendingChanges:  make([]operation.Operation, 0),
		SentChanges:     make([]operation.Operation, 0),
		Document:        "",
	}
}
