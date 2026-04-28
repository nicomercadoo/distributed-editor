package server

import "proyecto-final-redes/internal/protocol"

type Hub struct {
  clients    map[*Client]bool
  Broadcast  chan protocol.Operation
  Register   chan *Client
  Unregister chan *Client
  
  // El documento en memoria: una secuencia de operaciones (log)
  DocumentLog []protocol.Operation
}

func NewHub() *Hub {
  return &Hub{
    Broadcast:   make(chan protocol.Operation),
    Register:    make(chan *Client),
    Unregister:  make(chan *Client),
    clients:     make(map[*Client]bool),
    DocumentLog: make([]protocol.Operation, 0),
  }
}

func (h *Hub) Run() {
  for {
    select {
    case client := <-h.Register:
      h.clients[client] = true
      
      // Al conectarse, el cliente obtiene la última versión
      for _, op := range h.DocumentLog {
        client.send <- op
      }

    case client := <-h.Unregister:
      if _, ok := h.clients[client]; ok {
        delete(h.clients, client)
        close(client.send)
      }

    case op := <-h.Broadcast:
      // Guardamos la operación en el log general
      h.DocumentLog = append(h.DocumentLog, op)
      
      // Reenviamos a todos los conectados
      for client := range h.clients {
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