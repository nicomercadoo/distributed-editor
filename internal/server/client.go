package server

import (
	"log"
	"proyecto-final-redes/internal/protocol"

	"github.com/gorilla/websocket"
)

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan protocol.Message
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister <- c
		c.conn.Close()
	}()
	for {
		var msg protocol.Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error de lectura: %v", err)
			}
			break
		}

		if msg.Type != protocol.MessageTypeOperation {
			continue
		}

		op, err := msg.DecodeOperation()
		if err != nil {
			log.Printf("Error decodificando operacion: %v", err)
			continue
		}

		c.hub.Broadcast <- broadcastMsg{client: c, op: op}
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
			return
		}
	}
}
