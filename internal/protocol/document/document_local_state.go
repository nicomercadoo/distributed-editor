package document

import "proyecto-final-redes/internal/protocol/operation"

type DocumentLocalState struct {
	LastSyncRevision ID                    `json:"last_sync_version"`
	PendingChanges  []operation.Operation `json:"pending_changes"`
	SentChanges     []operation.Operation `json:"sent_changes"`
	Document        DocumentRep           `json:"document"`
}

func NewDocumentLocalState() *DocumentLocalState {
	return &DocumentLocalState{
		LastSyncRevision: 0,
		PendingChanges:  make([]operation.Operation, 0),
		SentChanges:     make([]operation.Operation, 0),
		Document:        "",
	}
}
