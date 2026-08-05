package server

import (
	"log"

	"proyecto-final-redes/internal/protocol/message"
)

type dispatchMsg struct {
	client *Client
	op     message.Operation
}

type registerMsg struct {
	client   *Client
	revision int
}

type Hub struct {
	clients       map[*Client]bool
	Dispatch      chan dispatchMsg
	Register      chan registerMsg
	Unregister    chan *Client
	DocumentState *DocumentServerState
}

func NewHub(clear bool) *Hub {
	return &Hub{
		Dispatch:      make(chan dispatchMsg),
		Register:      make(chan registerMsg),
		Unregister:    make(chan *Client),
		clients:       make(map[*Client]bool),
		DocumentState: NewDocumentServerState(clear),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case regMsg := <-h.Register:
			h.handleRegister(regMsg)
		case client := <-h.Unregister:
			h.handleUnregister(client)
		case dispatchMsg := <-h.Dispatch:
			h.handleDispatch(dispatchMsg)
		}
	}
}

// handleRegister registra un nuevo cliente y le envía las operaciones faltantes para sincronizar su estado
func (h *Hub) handleRegister(regMsg registerMsg) {
	h.clients[regMsg.client] = true
	log.Printf("Cliente registrado %s. Clientes activos: %d", regMsg.client.conn.RemoteAddr(), len(h.clients))
	if regMsg.revision >= 0 && regMsg.revision <= len(h.DocumentState.RevisionLog) {
		opsMissing := len(h.DocumentState.RevisionLog) - regMsg.revision
		if opsMissing > 0 {
			log.Printf("Sincronizando %d operaciones faltantes al cliente %s", opsMissing, regMsg.client.conn.RemoteAddr())
		}
	SyncLoop:
		for i := regMsg.revision; i < len(h.DocumentState.RevisionLog); i++ {
			op := h.DocumentState.RevisionLog[i]
			msg, err := message.NewMessage(message.MessageTypeOperation, op)
			if err != nil {
				continue
			}
			select {
			case regMsg.client.send <- msg:
			default:
				log.Printf("Cliente %s desconectado por saturación en SyncLoop", regMsg.client.conn.RemoteAddr())
				close(regMsg.client.send)
				delete(h.clients, regMsg.client)
				break SyncLoop
			}
		}
	}
}

// handleUnregister elimina al cliente del hub y cierra su canal de envío
func (h *Hub) handleUnregister(client *Client) {
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
		log.Printf("Cliente %s desconectado. Clientes activos: %d", client.conn.RemoteAddr(), len(h.clients))
	}
}

// handleDispatch procesa una operación entrante: la transforma, la aplica al estado del documento y la retransmite a los demás clientes
func (h *Hub) handleDispatch(dispatchMsg dispatchMsg) {
	// Si la operación ya fue procesada, le enviamos el ack al cliente y salimos
	if h.DocumentState.IsDuplicate(dispatchMsg.op.ID) {
		log.Printf("Operación duplicada detectada -> Tipo: %s, Char: %q, Pos: %d, ID: %s (rev cliente: %d). Host: %s", dispatchMsg.op.Op, dispatchMsg.op.Content, dispatchMsg.op.Position, dispatchMsg.op.ID, dispatchMsg.op.Revision, dispatchMsg.client.conn.RemoteAddr())
		ackMsg, err := message.NewMessage(message.MessageTypeAckApplied, message.AckApplied{
			OperationID:    dispatchMsg.op.ID,
			ServerRevision: len(h.DocumentState.RevisionLog),
		})
		if err == nil {
			select {
			case dispatchMsg.client.send <- ackMsg:
			default:
			}
		}
		return
	}

	isValid, transformedOp, newRevision := h.DocumentState.ProcessOperation(dispatchMsg.op)
	log.Printf("Operación procesada -> Tipo: %s, Char: %q, Pos: %d, Rev cliente: %d, ID: %s, Host: %s. Válida: %v, Nueva Revisión: %d", dispatchMsg.op.Op, dispatchMsg.op.Content, dispatchMsg.op.Position, dispatchMsg.op.Revision, dispatchMsg.op.ID, dispatchMsg.client.conn.RemoteAddr(), isValid, newRevision)

	// CONFIRMAR AL AUTOR (ACK)
	// Le avisamos al cliente su nuevo número de revisión (incluso si la op fue rechazada)
	ackMsg, err := message.NewMessage(message.MessageTypeAckApplied, message.AckApplied{
		OperationID:    dispatchMsg.op.ID,
		ServerRevision: newRevision,
	})
	if err == nil {
		select {
		case dispatchMsg.client.send <- ackMsg:
		default:
		}
	}

	// Si la operación fue invalidada, cortamos acá y no hacemos broadcast
	if !isValid {
		return
	}

	// BROADCAST A LOS DEMÁS
	// Enviamos la operación TRANSFORMADA a todos menos al autor
	outMsg, err := message.NewMessage(message.MessageTypeOperation, transformedOp)
	if err == nil {
		for client := range h.clients {
			if client == dispatchMsg.client {
				continue // No le enviamos la operación al autor
			}

			select {
			case client.send <- outMsg:
			default:
				log.Printf("Cliente desconectado por saturación en broadcast")
				close(client.send)
				delete(h.clients, client)
			}
		}
	}
}
