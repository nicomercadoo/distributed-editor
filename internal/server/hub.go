package server

import "proyecto-final-redes/internal/protocol"

type broadcastMsg struct {
  client *Client
  op     protocol.OperationMessage
}

type Hub struct {
  clients     map[*Client]bool
  Broadcast   chan broadcastMsg
  Register    chan *Client
  Unregister  chan *Client
  DocumentLog []protocol.OperationMessage
}

func NewHub() *Hub {
  return &Hub{
    Broadcast:   make(chan broadcastMsg),
    Register:    make(chan *Client),
    Unregister:  make(chan *Client),
    clients:     make(map[*Client]bool),
    DocumentLog: make([]protocol.OperationMessage, 0),
  }
}

func (h *Hub) Run() {
  for {
    select {
    case client := <-h.Register:
      h.clients[client] = true
      for _, op := range h.DocumentLog {
        client.send <- op
      }

    case client := <-h.Unregister:
      if _, ok := h.clients[client]; ok {
        delete(h.clients, client)
        close(client.send)
      }

    case bMsg := <-h.Broadcast:
      h.DocumentLog = append(h.DocumentLog, bMsg.op)

      for client := range h.clients {
        if client == bMsg.client {
          continue
        }

        select {
        case client.send <- bMsg.op:
        default:
          close(client.send)
          delete(h.clients, client)
        }
      }
    }
  }
}
