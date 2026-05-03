package server

import "proyecto-final-redes/internal/protocol/operation"
import "proyecto-final-redes/internal/protocol/document"

type Change struct {
    Revision  document.ID
    Operation operation.Operation
    Client    *Client
}

type DocumentServerState struct {
	PendingChanges []Change
	RevisionLog    []operation.Operation
	Document       document.DocumentRep
}

func NewDocumentServerState() *DocumentServerState {
	return &DocumentServerState{
		PendingChanges: make([]Change, 0),
		RevisionLog:    make([]operation.Operation, 0),
		Document:       "",
	}
}
