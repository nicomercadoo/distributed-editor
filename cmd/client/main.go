package main

import "proyecto-final-redes/internal/client"

func main() {
	client.StartSession("ws://localhost:8080/ws")
}