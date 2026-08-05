package client

import (
	"context"
	"fmt"
	"sync"

	"proyecto-final-redes/internal/ot"
	"proyecto-final-redes/internal/protocol/message"

	"github.com/google/uuid"
)

type Document = string

type SyncEngine struct {
	mu       sync.RWMutex
	document Document
	revision int
	pending  []message.Operation
	inFlight *message.Operation

	LocalOps <-chan message.Operation
	Incoming <-chan message.ServerEvent
	Outbound chan<- message.Operation

	Notifications chan<- string
}

func NewSyncEngine(
	localOps <-chan message.Operation,
	incoming <-chan message.ServerEvent,
	outbound chan<- message.Operation,
	notifications chan<- string,
) *SyncEngine {
	return &SyncEngine{
		LocalOps:      localOps,
		Incoming:      incoming,
		Outbound:      outbound,
		Notifications: notifications,
	}
}

func applyOperation(doc Document, op message.Operation) Document {
	switch op.Op {
	case message.Insert:
		if op.Position > len(doc) {
			op.Position = len(doc)
		}
		return doc[:op.Position] + op.Content + doc[op.Position:]
	case message.Delete:
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
			case message.AckApplied:
				e.handleAckApplied(event)
			case message.Operation:
				e.handleRemoteOp(event)
			}
		}
	}
}

func (e *SyncEngine) handleLocalOp(op message.Operation) {
	e.mu.Lock()
	// Aplicar optimista
	e.document = applyOperation(e.document, op)

	op.ID = uuid.New().String()
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
	case e.Notifications <- fmt.Sprintf("Operación local %s aplicada. Documento: \"%s\"", op.Op, docStr):
	default:
	}
}

func (e *SyncEngine) handleRemoteOp(op message.Operation) {
	e.mu.Lock()

	// Chequeamos si es nuestra propia operacion que vuelve durante el sync
	if e.inFlight != nil && e.inFlight.ID == op.ID {
		e.revision++
		e.advancePending()
		docStr := string(e.document)
		e.mu.Unlock()

		select {
		case e.Notifications <- fmt.Sprintf("Operación propia confirmada vía sync. Documento: \"%s\"", docStr):
		default:
		}
		return
	}

	isValid := true
	if e.inFlight != nil {
		localOp := *e.inFlight
		transformedLocalOp, _ := ot.Transform(localOp, op)
		*e.inFlight = transformedLocalOp
		op, isValid = ot.Transform(op, localOp)
	}
	if isValid {
		for i := range e.pending {
			localOp := e.pending[i]
			e.pending[i], _ = ot.Transform(localOp, op)
			op, isValid = ot.Transform(op, localOp)
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
		case e.Notifications <- fmt.Sprintf("Operación remota %s aplicada. Documento: \"%s\"", op.Op, docStr):
		default:
		}
	}
}

func (e *SyncEngine) handleAckApplied(ack message.AckApplied) {
	e.mu.Lock()
	if e.inFlight == nil || e.inFlight.Revision != ack.ClientRevision {
		e.mu.Unlock()
		return
	}
	e.revision = ack.ServerRevision
	e.advancePending()
	rev := e.revision
	e.mu.Unlock()
	select {
	case e.Notifications <- fmt.Sprintf("Ack revision %d recibido", rev):
	default:
	}
}

// Asume que e.mu está bloqueado por el invocante
func (e *SyncEngine) advancePending() {
	e.inFlight = nil
	if len(e.pending) > 0 {
		nextOp := e.pending[0]
		e.pending = e.pending[1:]
		nextOp.Revision = e.revision
		e.inFlight = &nextOp
		e.Outbound <- *e.inFlight
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
