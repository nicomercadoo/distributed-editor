package server

import (
	"proyecto-final-redes/internal/protocol/ack"
	"proyecto-final-redes/internal/protocol/message"
	"proyecto-final-redes/internal/protocol/operation"
)

type broadcastMsg struct {
	client *Client
	op     operation.Operation
}

type Hub struct {
	clients       map[*Client]bool
	Broadcast     chan broadcastMsg
	Register      chan *Client
	Unregister    chan *Client
	DocumentState *DocumentServerState
}

func NewHub() *Hub {
	return &Hub{
		Broadcast:     make(chan broadcastMsg),
		Register:      make(chan *Client),
		Unregister:    make(chan *Client),
		clients:       make(map[*Client]bool),
		DocumentState: NewDocumentServerState(),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.clients[client] = true
			for _, op := range h.DocumentState.RevisionLog {
				msg, err := message.NewMessage(message.MessageTypeOperation, op)
				if err != nil {
					continue
				}
				client.send <- msg
			}

		case client := <-h.Unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}

		case bMsg := <-h.Broadcast:
			op := bMsg.op
			clientRevision := op.Revision

			// 1. ALGORITMO DE CONTROL DE TRANSFORMACIÓN
			// Comparamos la revisión del cliente con la longitud actual del log del servidor.
			// Si el cliente está desactualizado, transformamos su operación contra la historia reciente.
			isValid := true
			for i := clientRevision; i < len(h.DocumentState.RevisionLog); i++ {
				appliedOp := h.DocumentState.RevisionLog[i]

				// Llamamos a la Función de Transformación que creaste en el paquete protocol
				op, isValid = operation.Transform(op, appliedOp)

				// Si la operación se anula (ej. intentar borrar un carácter ya borrado), cortamos el bucle.
				if !isValid {
					break
				}
			}

			// Si la operación quedó invalidada (No-Op), solo le enviamos el Ack al cliente
			// para que actualice su revisión, pero no la guardamos ni hacemos broadcast.
			if !isValid {
				ackMsg, err := message.NewMessage(message.MessageTypeAck, ack.Ack{Revision: len(h.DocumentState.RevisionLog)})
				if err == nil {
					select {
					case bMsg.client.send <- ackMsg:
					default:
					}
				}
				continue // Pasamos a escuchar el siguiente mensaje
			}

			// 2. APLICAR Y GUARDAR
			// La operación superó las transformaciones. La guardamos como la verdad absoluta.
			h.DocumentState.RevisionLog = append(h.DocumentState.RevisionLog, op)
			newRevision := len(h.DocumentState.RevisionLog)

			// 3. CONFIRMAR AL AUTOR (ACK)
			// Le avisamos al cliente que su operación fue aceptada y le damos el nuevo número de revisión.
			ackMsg, err := message.NewMessage(message.MessageTypeAck, ack.Ack{Revision: newRevision})
			if err == nil {
				select {
				case bMsg.client.send <- ackMsg:
				default:
				}
			}

			// 4. BROADCAST A LOS DEMÁS
			// IMPORTANTE: Enviamos la operación TRANSFORMADA, no la original.
			outMsg, err := message.NewMessage(message.MessageTypeOperation, op)
			if err == nil {
				for client := range h.clients {
					if client == bMsg.client {
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
