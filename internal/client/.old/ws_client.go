package client

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"proyecto-final-redes/internal/protocol/message"
	"proyecto-final-redes/internal/protocol/operation"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

func StartSession(url string) {
	// 1. Conectarse al servidor
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatalf("Error conectando al servidor: %v", err)
	}
	defer conn.Close()
	fmt.Println("Conectado al servidor. Uso 'I <pos> <char>' o 'D <pos>' (o 'salir'):")

	docState := NewDocumentLocalState()
	var mu sync.Mutex // Mutex para proteger el acceso a docState

	// 2. Goroutine para LEER mensajes del servidor en segundo plano
	serverListener := func() {
		for {
			var msg message.Message
			err := conn.ReadJSON(&msg)
			if err != nil {
				fmt.Println("\nConexión cerrada por el servidor.")
				os.Exit(0)
			}

			switch msg.Type {
			case message.MessageTypeOperation:
				op, err := msg.DecodeOperation()
				if err != nil {
					fmt.Print("\n[SERVER]: Operacion invalida\n> ")
					continue
				}

				mu.Lock()
				// A. Transformar la operación entrante contra lo que ya mandamos al servidor (SentChanges)
				isValid := true
				for i := range docState.SentChanges {
					localOp := docState.SentChanges[i]
					// Actualizamos la operación local
					docState.SentChanges[i], _ = operation.Transform(localOp, op)
					// Actualizamos la operación entrante
					op, isValid = operation.Transform(op, localOp)
					if !isValid {
						break
					}
				}

				// B. Transformar la operación entrante contra lo que el usuario tipeó pero no se mandó (PendingChanges)
				if isValid {
					for i := range docState.PendingChanges {
						localOp := docState.PendingChanges[i]
						// Actualizamos la operación local
						docState.PendingChanges[i], _ = operation.Transform(localOp, op)
						// Actualizamos la operación entrante
						op, isValid = operation.Transform(op, localOp)
						if !isValid {
							break
						}
					}
				}

				// Incrementamos la revisión local porque el servidor nos mandó una operación oficial nueva
				docState.LastSyncRevision++
				mu.Unlock()

				if isValid {
					// Aca iría una posible lógica para escribir el documento.
					fmt.Printf("\n[SERVER]: Operación %s transformada a posición %d con contenido '%s'\n> ", op.Op, op.Position, op.Content)
				}

			case message.MessageTypeAck:
				ackMsg, err := msg.DecodeAck()
				if err != nil {
					fmt.Print("\n[SERVER]: Ack invalido\n> ")
					continue
				}

				mu.Lock()
				// Actualizamos nuestra revisión a la que nos confirmó el servidor
				docState.LastSyncRevision = ackMsg.Revision

				// La primera operación de SentChanges ya fue confirmada, la sacamos de la cola
				if len(docState.SentChanges) > 0 {
					docState.SentChanges = docState.SentChanges[1:]
				}

				// Si tenemos cambios pendientes, movemos el primero a Sent y lo enviamos
				if len(docState.PendingChanges) > 0 {
					nextOp := docState.PendingChanges[0]
					docState.PendingChanges = docState.PendingChanges[1:]
					// Ajustamos la revisión a la última sincronizada antes de mandar
					nextOp.Revision = docState.LastSyncRevision
					docState.SentChanges = append(docState.SentChanges, nextOp)

					outMsg, _ := message.NewMessage(message.MessageTypeOperation, nextOp)
					conn.WriteJSON(outMsg) // WriteJSON está protegido por el Mutex
				}
				mu.Unlock()
				fmt.Printf("\n[SERVER]: Ack revision %d\n> ", ackMsg.Revision)

			default:
				fmt.Print("\n[SERVER]: Tipo de mensaje desconocido\n> ")
			}
		}
	}

	go serverListener()

	// 3. Bucle principal para LEER de la terminal y ENVIAR al servidor
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		text := scanner.Text()

		if text == "salir" {
			break
		}

		parts := strings.Split(strings.TrimSpace(text), " ")
		if len(parts) == 0 || parts[0] == "" {
			continue
		}

		var op operation.Operation
		command := strings.ToUpper(parts[0])

		mu.Lock()
		actualRevision := docState.LastSyncRevision
		mu.Unlock()

		if command == "I" && len(parts) >= 3 {
			// Parsear Posición
			var pos int
			_, err := fmt.Sscanf(parts[1], "%d", &pos)
			if err != nil {
				fmt.Println("\nFormato incorrecto. La posición debe ser un número entero.")
				continue
			}

			// Asegurarnos de tomar estrictamente un solo carácter (rune)
			runes := []rune(parts[2])
			if len(runes) == 0 {
				continue
			}
			charString := string(runes[0])

			op = operation.Operation{
				Op:       operation.Insert,
				Position: pos,
				Content:  charString,
				Revision: actualRevision,
			}

		} else if command == "D" && len(parts) >= 2 {
			// Parsear Posición
			var pos int
			_, err := fmt.Sscanf(parts[1], "%d", &pos)
			if err != nil {
				fmt.Println("\nFormato incorrecto. La posición debe ser un número entero.")
				continue
			}

			op = operation.Operation{
				Op:       operation.Delete,
				Position: pos,
				Revision: actualRevision,
			}

		} else {
			fmt.Println("\nFormato incorrecto. Uso: 'I <pos> <char>' o 'D <pos>'")
			continue
		}

		mu.Lock()
		// Control de flujo: Solo mandamos al servidor si no estamos esperando un ACK
		if len(docState.SentChanges) == 0 {
			docState.SentChanges = append(docState.SentChanges, op)
			msg, err := message.NewMessage(message.MessageTypeOperation, op)
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
		} else {
			// Si la red está ocupada, lo encolamos
			docState.PendingChanges = append(docState.PendingChanges, op)
		}
		mu.Unlock()
	}
}
