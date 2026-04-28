package client

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"github.com/gorilla/websocket"
	"proyecto-final-redes/internal/protocol"
)

func StartSession(url string) {
	// 1. Conectarse al servidor
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		panic(fmt.Sprintf("Error conectando al servidor: %v", err))
	}
	defer conn.Close()
	fmt.Println("Conectado al servidor. Escribí 'insert' o 'delete' (o 'salir'):")

	// 2. Goroutine para LEER mensajes del servidor en segundo plano
	go func() {
		for {
			var op protocol.Operation
			err := conn.ReadJSON(&op)
			if err != nil {
				fmt.Println("\nConexión cerrada por el servidor.")
				os.Exit(0)
			}
			fmt.Printf("\n[SERVIDOR DICE]: Operación %s en posición %d\n> ", op.Type, op.Pos)
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
		op := protocol.Operation{
			Type: "insert",
			Char: strings.TrimSpace(texto),
			Pos:  0,
		}

		// Enviamos el struct como JSON
		err = conn.WriteJSON(op)
		if err != nil {
			fmt.Println("Error al enviar:", err)
			break
		}
	}
}