package server

import (
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

func NewHub() *Hub {
	return &Hub{
		Dispatch:      make(chan dispatchMsg),
		Register:      make(chan registerMsg),
		Unregister:    make(chan *Client),
		clients:       make(map[*Client]bool),
		DocumentState: NewDocumentServerState(),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case regMsg := <-h.Register:
			h.clients[regMsg.client] = true
			// Enviamos operaciones faltantes
			if regMsg.revision >= 0 && regMsg.revision <= len(h.DocumentState.RevisionLog) {
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
						close(regMsg.client.send)
						delete(h.clients, regMsg.client)
						break SyncLoop
					}
				}
			}

		case client := <-h.Unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}

		case dispatchMsg := <-h.Dispatch:
			isValid, transformedOp, newRevision := h.DocumentState.ProcessOperation(dispatchMsg.op)

			// CONFIRMAR AL AUTOR (ACK)
			// Le avisamos al cliente su nuevo número de revisión (incluso si la op fue rechazada).
			ackMsg, err := message.NewMessage(message.MessageTypeAck, ack.Ack{Revision: newRevision})
			if err == nil {
				select {
				case dispatchMsg.client.send <- ackMsg:
				default:
				}
			}

			// Si la operación fue invalidada, cortamos acá y no hacemos broadcast
			if !isValid {
				continue
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
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
		}
	}
}
