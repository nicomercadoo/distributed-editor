package server

import (
	"log"
	"proyecto-final-redes/internal/protocol/ack"
	"proyecto-final-redes/internal/protocol/message"
	"proyecto-final-redes/internal/protocol/operation"
)

type dispatchMsg struct {
	client *Client
	op     operation.Operation
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

func (h *Hub) handleRegister(regMsg registerMsg) {
	h.clients[regMsg.client] = true
	log.Printf("Cliente registrado. Clientes activos: %d", len(h.clients))
	// Enviamos operaciones faltantes
	if regMsg.revision >= 0 && regMsg.revision <= len(h.DocumentState.RevisionLog) {
		opsMissing := len(h.DocumentState.RevisionLog) - regMsg.revision
		if opsMissing > 0 {
			log.Printf("Sincronizando %d operaciones faltantes al cliente", opsMissing)
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
				log.Printf("Cliente desconectado por saturación en SyncLoop")
				close(regMsg.client.send)
				delete(h.clients, regMsg.client)
				break SyncLoop
			}
		}
	}
}

func (h *Hub) handleUnregister(client *Client) {
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
		log.Printf("Cliente desconectado. Clientes activos: %d", len(h.clients))
	}
}

func (h *Hub) handleDispatch(dispatchMsg dispatchMsg) {
	isValid, transformedOp, newRevision := h.DocumentState.ProcessOperation(dispatchMsg.op)
	log.Printf("Operación procesada (rev cliente: %d). Válida: %v, Nueva Revisión: %d", dispatchMsg.op.Revision, isValid, newRevision)

	// CONFIRMAR AL AUTOR (ACK)
	// Le avisamos al cliente su nuevo número de revisión (incluso si la op fue rechazada).
	ackMsg, err := message.NewMessage(message.MessageTypeAckApplied, ack.AckApplied{Revision: newRevision})
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
	// Enviamos la operación TRANSFORMADA a todos menos al autor.
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
