package server

import (
	"proyecto-final-redes/internal/protocol/ack"
	"proyecto-final-redes/internal/protocol/document"
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
	DocumentState *document.DocumentServerState
}

func NewHub() *Hub {
	return &Hub{
		Broadcast:     make(chan broadcastMsg),
		Register:      make(chan *Client),
		Unregister:    make(chan *Client),
		clients:       make(map[*Client]bool),
		DocumentState: document.NewDocumentServerState(),
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
			h.DocumentState.RevisionLog = append(h.DocumentState.RevisionLog, bMsg.op)
			h.DocumentState.Revision++

			ackMsg, err := message.NewMessage(message.MessageTypeAck, ack.Ack{Revision: h.DocumentState.Revision})
			if err == nil {
				select {
				case bMsg.client.send <- ackMsg:
				default:
				}
			}

			op, err := message.NewMessage(message.MessageTypeOperation, bMsg.op)
			for client := range h.clients {
				if client == bMsg.client {
					continue
				}

				select {
				case client.send <- op:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}
