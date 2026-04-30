package server

import (
  "log"
  "github.com/gorilla/websocket"
  "proyecto-final-redes/internal/protocol"
)

type Client struct {
  hub  *Hub
  conn *websocket.Conn
  send chan protocol.OperationMessage
}

func (c *Client) ReadPump() {
  defer func() {
    c.hub.Unregister <- c
    c.conn.Close()
  }()
  for {
    var op protocol.OperationMessage
    err := c.conn.ReadJSON(&op)
    if err != nil {
      if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
        log.Printf("Error de lectura: %v", err)
      }
      break
    }
    c.hub.Broadcast <- op
  }
}

func (c *Client) WritePump() {
  defer func() {
    c.conn.Close()
  }()
  for {
    op, ok := <-c.send
    if !ok {
      c.conn.WriteMessage(websocket.CloseMessage, []byte{})
      return
    }

    err := c.conn.WriteJSON(op)
    if err != nil {
      return
    }
  }
}
