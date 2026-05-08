package client

import (
    "fmt"
    // "log"
    "os"
    "proyecto-final-redes/internal/protocol/message"
    "proyecto-final-redes/internal/protocol/operation"
    "sync"

    "github.com/gorilla/websocket"
)

func serverListener(conn *websocket.Conn, docState *DocumentLocalState, mu *sync.Mutex) {
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
