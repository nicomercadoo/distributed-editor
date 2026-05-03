package server

import "proyecto-final-redes/internal/protocol"

type broadcastMsg struct {
	client *Client
	op     protocol.OperationMessage
}

type Hub struct {
	clients       map[*Client]bool
	Broadcast     chan broadcastMsg
	Register      chan *Client
	Unregister    chan *Client
	DocumentState *protocol.DocumentServerState
}

func NewHub() *Hub {
	return &Hub{
		Broadcast:  make(chan broadcastMsg),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		DocumentState: protocol.NewDocumentServerState(),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.clients[client] = true
			for _, op := range h.DocumentState.RevisionLog {
				msg, err := protocol.NewMessage(protocol.MessageTypeOperation, op)
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

			ack, err := protocol.NewMessage(protocol.MessageTypeAck, protocol.AckMessage{Revision: h.DocumentState.Revision})
			if err == nil {
				select {
				case bMsg.client.send <- ack:
				default:
				}
			}

			op, err := protocol.NewMessage(protocol.MessageTypeOperation, bMsg.op)
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

// func mustOperationMessage(op protocol.OperationMessage) protocol.Message {
// 	msg, err := protocol.NewMessage(protocol.MessageTypeOperation, op)
// 	if err != nil {
// 		return protocol.Message{Type: protocol.MessageTypeOperation, Payload: []byte("null")}
// 	}
// 	return msg
// }
