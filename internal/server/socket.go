package server

import (
	"log"
	"net/http"
	"proyecto-final-redes/internal/protocol/message"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func HandleConnection(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Error upgrade:", err)
			return
		}

		client := &Client{
			hub:  hub,
			conn: conn,
			send: make(chan message.Message, 256),
		}

		client.hub.Register <- client

		go client.WritePump()
		go client.ReadPump()
	}
}
