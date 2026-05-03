package client

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"proyecto-final-redes/internal/protocol"
	"strings"

	"github.com/gorilla/websocket"
)

func StartSession(url string) {
	// 1. Conectarse al servidor
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatalf("Error conectando al servidor: %v", err)
	}
	defer conn.Close()
	fmt.Println("Conectado al servidor. Escribí 'insert' o 'delete' (o 'salir'):")

	// 2. Goroutine para LEER mensajes del servidor en segundo plano
	go func() {
		for {
			var msg protocol.Message
			err := conn.ReadJSON(&msg)
			if err != nil {
				fmt.Println("\nConexión cerrada por el servidor.")
				os.Exit(0)
			}

			switch msg.Type {
			case protocol.MessageTypeOperation:
				op, err := msg.DecodeOperation()
				if err != nil {
					fmt.Print("\n[SERVER]: Operacion invalida\n> ")
					continue
				}
				fmt.Printf("\n[SERVER]: Operación %s en posición %d con contenido '%s'\n> ", op.Op, op.Position, op.Content)

			case protocol.MessageTypeAck:
				ack, err := msg.DecodeAck()
				if err != nil {
					fmt.Print("\n[SERVER]: Ack invalido\n> ")
					continue
				}
				fmt.Printf("\n[SERVER]: Ack revision %d\n> ", ack.Revision)

			default:
				fmt.Print("\n[SERVER]: Tipo de mensaje desconocido\n> ")
			}
		}
	}()

	// 3. Bucle principal para LEER de la terminal y ENVIAR al servidor
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		texto := scanner.Text()

		if texto == "salir" {
			break
		}

		// Armamos un comando hardcodeado solo para probar la red
		op := protocol.OperationMessage{
			Op:       protocol.Insert,
			Position: 0,
			Content:  strings.TrimSpace(texto),
		}

		msg, err := protocol.NewMessage(protocol.MessageTypeOperation, op)
		if err != nil {
			fmt.Println("Error al serializar:", err)
			break
		}

		// Enviamos el struct como JSON
		err = conn.WriteJSON(msg)
		if err != nil {
			fmt.Println("Error al enviar:", err)
			break
		}
	}
}
