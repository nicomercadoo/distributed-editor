package document

import "proyecto-final-redes/internal/protocol/operation"

type DocumentServerState struct {
	Revision       ID                    `json:"revision"`
	PendingChanges []operation.Operation `json:"pending_changes"`
	RevisionLog    []operation.Operation `json:"revision_log"`
	Document       DocumentRep           `json:"document"`
}

func NewDocumentServerState() *DocumentServerState {
	return &DocumentServerState{
		Revision:       0,
		PendingChanges: make([]operation.Operation, 0),
		RevisionLog:    make([]operation.Operation, 0),
		Document:       "",
	}
}
