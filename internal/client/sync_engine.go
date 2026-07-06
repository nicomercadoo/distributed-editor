package client

import (
	"context"
	"fmt"
	"sync"

	"proyecto-final-redes/internal/protocol/document"
	"proyecto-final-redes/internal/protocol/operation"
)

type SyncEngine struct {
	mu       sync.RWMutex
	document document.DocumentRep
	revision document.ID
	pending  []operation.Operation
	inFlight *operation.Operation

	LocalOps  <-chan operation.Operation
	RemoteOps <-chan operation.Operation
	Acks      <-chan int
	Outbound  chan<- operation.Operation

	Notifications chan<- string
}

func NewSyncEngine(
	localOps <-chan operation.Operation,
	remoteOps <-chan operation.Operation,
	acks <-chan int,
	outbound chan<- operation.Operation,
	notifications chan<- string,
) *SyncEngine {
	return &SyncEngine{
		LocalOps:      localOps,
		RemoteOps:     remoteOps,
		Acks:          acks,
		Outbound:      outbound,
		Notifications: notifications,
	}
}

func applyOperation(doc document.DocumentRep, op operation.Operation) document.DocumentRep {
	if op.Op == operation.Insert {
		if op.Position > len(doc) {
			op.Position = len(doc)
		}
		return doc[:op.Position] + op.Content + doc[op.Position:]
	} else if op.Op == operation.Delete {
		if op.Position >= len(doc) || len(doc) == 0 {
			return doc
		}
		return doc[:op.Position] + doc[op.Position+1:]
	}
	return doc
}

func (e *SyncEngine) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case op := <-e.LocalOps:
			e.mu.Lock()
			// Aplicar optimista
			e.document = applyOperation(e.document, op)
			op.Revision = e.revision

			if e.inFlight == nil {
				inFlightOp := op
				e.inFlight = &inFlightOp
				e.Outbound <- inFlightOp
			} else {
				e.pending = append(e.pending, op)
			}
			docStr := string(e.document)
			e.mu.Unlock()
			select {
			case e.Notifications <- fmt.Sprintf("Operación local %s aplicada. Documento: %s", op.Op, docStr):
			default:
			}

		case op := <-e.RemoteOps:
			e.mu.Lock()
			isValid := true
			if e.inFlight != nil {
				localOp := *e.inFlight
				*e.inFlight, _ = operation.Transform(localOp, op)
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

		case ackRev := <-e.Acks:
			e.mu.Lock()
			e.revision = ackRev
			e.inFlight = nil
			if len(e.pending) > 0 {
				nextOp := e.pending[0]
				e.pending = e.pending[1:]
				nextOp.Revision = e.revision
				e.inFlight = &nextOp
				e.Outbound <- nextOp
			}
			rev := e.revision
			e.mu.Unlock()
			select {
			case e.Notifications <- fmt.Sprintf("Ack revision %d recibido", rev):
			default:
			}
		}
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
