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
	// Operaciones locales encoladas
	pending []message.Operation
	// Operación local enviada al servidor, aún no confirmada
	inFlight *message.Operation
	// Flag por si la operación inFlight se vuelve una No-op
	inFlightInvalid bool

	// Canales de entrada
	LocalOps <-chan message.Operation
	Incoming <-chan message.ServerEvent
	// Canales de salida
	Outbound      chan<- message.Operation
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

// applyOperation aplica una operación sobre un documento
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

// handleLocalOp maneja las operaciones locales que llegan por el canal localOps
func (e *SyncEngine) handleLocalOp(op message.Operation) {
	e.mu.Lock()
	// Aplicamos la operación de manera optimista
	e.document = applyOperation(e.document, op)

	op.ID = uuid.New().String()
	op.Revision = e.revision

	// Si no hay operación en vuelo, enviamos esta
	if e.inFlight == nil {
		inFlightOp := op
		e.inFlight = &inFlightOp
		e.inFlightInvalid = false
		e.Outbound <- *e.inFlight
	} else {
		// Si hay operación en vuelo, la encolamos
		e.pending = append(e.pending, op)
	}
	docStr := string(e.document)
	e.mu.Unlock()
	select {
	case e.Notifications <- fmt.Sprintf("Operación local %s aplicada. Documento: \"%s\"", op.Op, docStr):
	default:
	}
}

// handleRemoteOp maneja las operaciones remotas que llegan por el canal incoming
func (e *SyncEngine) handleRemoteOp(op message.Operation) {
	e.mu.Lock()

	// Chequeamos si es nuestra propia operación que vuelve durante una reconexión
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
	// Si hay operación en vuelo, transformamos la operación remota contra la operación en vuelo
	if e.inFlight != nil {
		// Solo transformamos si la operación en vuelo no es una No-op
		if !e.inFlightInvalid {
			localOp := *e.inFlight
			transformedLocalOp, isValidLocal := ot.Transform(localOp, op)
			if !isValidLocal {
				e.inFlightInvalid = true // Si la operación en vuelo se volvió una No-op, la marcamos como inválida
			}
			*e.inFlight = transformedLocalOp
			op, isValid = ot.Transform(op, localOp)
		}
	}

	// Transformamos la operacion remota contra todas las operaciones pendientes
	if isValid {
		validPending := make([]message.Operation, 0, len(e.pending))
		for i := range e.pending {
			localOp := e.pending[i]
			transformedLocalOp, isValidLocal := ot.Transform(localOp, op)
			// Si la operación pendiente es válida, la mantenemos
			if isValidLocal {
				validPending = append(validPending, transformedLocalOp)
			}
			op, isValid = ot.Transform(op, localOp)
			// Si la operación pendiente se volvió inválida, directamente la descartamos
			if !isValid {
				validPending = append(validPending, e.pending[i+1:]...)
				break
			}
		}
		e.pending = validPending
	}
	if isValid {
		// Aplicamos operación remota al documento local
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

// handleAckApplied maneja los acks que llegan por el canal incoming
func (e *SyncEngine) handleAckApplied(ack message.AckApplied) {
	e.mu.Lock()
	// Chequeamos que el ack corresponda a la operación en vuelo
	if e.inFlight == nil || e.inFlight.ID != ack.OperationID {
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

// advancePending avanza a la siguiente operación pendiente
// Asume que e.mu está bloqueado por el invocante
func (e *SyncEngine) advancePending() {
	e.inFlight = nil
	e.inFlightInvalid = false
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
