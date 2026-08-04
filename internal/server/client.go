package server

import (
	"log"

	"proyecto-final-redes/internal/protocol/message"

	"github.com/gorilla/websocket"
)

type Client struct {
	hub              *Hub
	conn             *websocket.Conn
	send             chan message.Message
	lastRevisionRead int
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister <- c
		c.conn.Close()
	}()
	for {
		var msg message.Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error de lectura: %v", err)
			}
			break
		}

		if msg.Type != message.MessageTypeOperation {
			continue
		}

		op, err := msg.DecodeOperation()
		if err != nil {
			log.Printf("Error decodificando operacion: %v", err)
			continue
		}

		if op.Revision <= c.lastRevisionRead {
			log.Printf("Operación duplicada de %s: revisión %d <= %d", c.conn.RemoteAddr().String(), op.Revision, c.lastRevisionRead)
			c.hub.Dispatch <- dispatchMsg{client: c, op: op, IsDuplicate: true}
			continue
		}
		c.lastRevisionRead = op.Revision

		c.hub.Dispatch <- dispatchMsg{client: c, op: op, IsDuplicate: false}
	}
}

func (c *Client) WritePump() {
	defer func() {
		c.conn.Close()
	}()
	for {
		msg, ok := <-c.send
		if !ok {
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		err := c.conn.WriteJSON(msg)
		if err != nil {
			log.Printf("Error de escritura al cliente: %v", err)
			return
		}
	}
}
