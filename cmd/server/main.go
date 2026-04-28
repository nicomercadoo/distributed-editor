package main

import (
  "fmt"
  "net/http"
  "proyecto-final-redes/internal/server"
)

func main() {
  // Arrancamos el event loop del documento
  go server.GlobalHub.Run()

  http.HandleFunc("/ws", server.HandleConnection)
  
  fmt.Println("Servidor de documento único escuchando en :8080...")
  err := http.ListenAndServe(":8080", nil)
  if err != nil {
    panic(err)
  }
}