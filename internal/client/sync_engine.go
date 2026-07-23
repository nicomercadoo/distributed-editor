package client

import (
	"context"
	"fmt"
	"sync"

	"proyecto-final-redes/internal/protocol/ack"
	"proyecto-final-redes/internal/protocol/message"
	"proyecto-final-redes/internal/protocol/operation"
)

type Document = string

type SyncEngine struct {
	mu       sync.RWMutex
	document Document
	revision int
	pending  []operation.Operation
	inFlight *operation.Operation

	LocalOps <-chan operation.Operation
	Incoming <-chan message.ServerEvent
	Outbound chan<- operation.Operation

	Notifications chan<- string
}

func NewSyncEngine(
	localOps <-chan operation.Operation,
	incoming <-chan message.ServerEvent,
	outbound chan<- operation.Operation,
	notifications chan<- string,
) *SyncEngine {
	return &SyncEngine{
		LocalOps:      localOps,
		Incoming:      incoming,
		Outbound:      outbound,
		Notifications: notifications,
	}
}

func applyOperation(doc Document, op operation.Operation) Document {
	switch op.Op {
	case operation.Insert:
		if op.Position > len(doc) {
			op.Position = len(doc)
		}
		return doc[:op.Position] + op.Content + doc[op.Position:]
	case operation.Delete:
		if op.Position >= len(doc) || len(doc) == 0 {
			return doc
		}
		return doc[:op.Position] + doc[op.Position+1:]
	default:
	}
	return doc
}

func (e *SyncEngine) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case op := <-e.LocalOps:
			e.handleLocalOp(op)
		case msg := <-e.Incoming:
			switch event := msg.(type) {
			// handleAckReceived ya no se procesa aquí
			case ack.AckApplied:
				e.handleAckApplied(event)
			case operation.Operation:
				e.handleRemoteOp(event)
			}
		}
	}
}

func (e *SyncEngine) handleLocalOp(op operation.Operation) {
	e.mu.Lock()
	// Aplicar optimista
	e.document = applyOperation(e.document, op)

	op.Revision = e.revision

	if e.inFlight == nil {
		inFlightOp := op
		e.inFlight = &inFlightOp
		e.Outbound <- *e.inFlight
	} else {
		e.pending = append(e.pending, op)
	}
	docStr := string(e.document)
	e.mu.Unlock()
	select {
	case e.Notifications <- fmt.Sprintf("Operación local %s aplicada. Documento: %s", op.Op, docStr):
	default:
	}
}

func (e *SyncEngine) handleRemoteOp(op operation.Operation) {
	e.mu.Lock()
	isValid := true
	if e.inFlight != nil {
		localOp := *e.inFlight
		transformedLocalOp, _ := operation.Transform(localOp, op)
		*e.inFlight = transformedLocalOp
		op, isValid = operation.Transform(op, localOp)
	}
	if isValid {
		for i := range e.pending {
			localOp := e.pending[i]
			e.pending[i], _ = operation.Transform(localOp, op)
			op, isValid = operation.Transform(op, localOp)
			if !isValid {
				break
			}
		}
	}
	if isValid {
		e.document = applyOperation(e.document, op)
	}
	e.revision++
	docStr := string(e.document)
	e.mu.Unlock()

	if isValid {
		select {
		case e.Notifications <- fmt.Sprintf("Operación remota %s aplicada. Documento: %s", op.Op, docStr):
		default:
		}
	}
}

// handleAckReceived removido porque ya no aplica a OT

func (e *SyncEngine) handleAckApplied(ack ack.AckApplied) {
	e.mu.Lock()
	e.revision = ack.Revision
	e.inFlight = nil
	if len(e.pending) > 0 {
		nextOp := e.pending[0]
		e.pending = e.pending[1:]
		nextOp.Revision = e.revision
		e.inFlight = &nextOp
		e.Outbound <- *e.inFlight
	}
	rev := e.revision
	e.mu.Unlock()
	select {
	case e.Notifications <- fmt.Sprintf("Ack revision %d recibido", rev):
	default:
	}
}

func (e *SyncEngine) GetDocument() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return string(e.document)
}

func (e *SyncEngine) GetRevision() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.revision
}

func (e *SyncEngine) GetPendingCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.pending)
}

func (e *SyncEngine) HasInFlight() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.inFlight != nil
}
