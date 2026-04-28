package server

import (
  "log"
  "net/http"
  "github.com/gorilla/websocket"
	"proyecto-final-redes/internal/protocol"
)

var upgrader = websocket.Upgrader{
  CheckOrigin: func(r *http.Request) bool { return true },
}

// Instancia global del único documento
var GlobalHub = NewHub()

func HandleConnection(w http.ResponseWriter, r *http.Request) {
  conn, err := upgrader.Upgrade(w, r, nil)
  if err != nil {
    log.Println("Error upgrade:", err)
    return
  }

  client := &Client{
    hub:  GlobalHub,
    conn: conn,
    send: make(chan protocol.Operation, 256),
  }

  client.hub.Register <- client

  go client.WritePump()
  go client.ReadPump()
}