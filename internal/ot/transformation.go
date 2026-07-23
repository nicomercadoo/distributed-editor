package ot

import "proyecto-final-redes/internal/protocol/message"

// Transform ajusta 'incomingOp' teniendo en cuenta que 'appliedOp' ya se ejecutó en el documento.
// El booleano retorna 'false' si la operación entrante se vuelve redundante (No-Op) y debe descartarse.
func Transform(incomingOp, appliedOp message.Operation) (message.Operation, bool) {
	switch incomingOp.Op {
	case message.Insert:
		switch appliedOp.Op {
		case message.Insert:
			// Ambos insertan. Si la operacion aplicada fue ANTES que la mia, mi posicion se desplaza a la derecha.
			if appliedOp.Position < incomingOp.Position {
				incomingOp.Position++
			} else if appliedOp.Position == incomingOp.Position {
				// EMPATE: Dos usuarios insertaron en el mismo lugar al mismo tiempo.
				// Para que los documentos converjan, la decisión debe ser determinista.
				// Como no tenemos ID de cliente en el struct message.Operation, podemos desempatar lexicográficamente:
				if appliedOp.Content < incomingOp.Content {
					incomingOp.Position++
				}
			}
		case message.Delete:
			// Si borraron un caracter que estaba ANTES de mi insercion, mi posicion se desplaza a la izquierda.
			if appliedOp.Position < incomingOp.Position {
				incomingOp.Position--
			}
		}

	case message.Delete:
		switch appliedOp.Op {
		case message.Insert:
			// Si insertaron algo ANTES de lo que yo quiero borrar, mi posicion de borrado se desplaza a la derecha.
			if appliedOp.Position <= incomingOp.Position {
				incomingOp.Position++
			}
		case message.Delete:
			if appliedOp.Position < incomingOp.Position {
				// Borraron algo ANTES de lo que yo quiero borrar
				incomingOp.Position--
			} else if appliedOp.Position == incomingOp.Position {
				// Alguien más ya borró el caracter que yo quería borrar. Mi operación se anula (No-Op).
				return incomingOp, false
			}
		}
	}

	return incomingOp, true
}
