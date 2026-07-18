package server

import (
	"log"
	"proyecto-final-redes/internal/protocol/ack"
	"proyecto-final-redes/internal/protocol/message"

	"github.com/gorilla/websocket"
)

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan message.Message
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

		ackRecv, err := message.NewMessage(message.MessageTypeAckReceived, ack.AckReceived{})
		if err == nil {
			// Bloqueamos hasta que haya espacio en el canal de envío.
			// Esto garantiza la entrega del AckReceived. Si el canal está lleno,
			// significa que el cliente está recibiendo muy lento (backpressure)
			// y está bien que ReadPump se frene hasta que se libere espacio.
			c.send <- ackRecv
		}

		c.hub.Dispatch <- dispatchMsg{client: c, op: op}
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
