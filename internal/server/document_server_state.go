package server

import (
	"proyecto-final-redes/internal/protocol/document"
	"proyecto-final-redes/internal/protocol/operation"
)

type DocumentServerState struct {
	RevisionLog []operation.Operation
	Document    document.DocumentRep // Por ahora no lo ulitizamos
}

func NewDocumentServerState() *DocumentServerState {
	return &DocumentServerState{
		RevisionLog: make([]operation.Operation, 0),
		Document:    "",
	}
}

/*
ProcessOperation toma una operación entrante, la transforma contra el historial reciente
y la aplica al estado local del documento.
Retorna:
- isValid: si la operación sobrevivió a las transformaciones.
- transformedOp: la operación resultante.
- revision: el número de revisión actualizado del servidor.
*/
func (state *DocumentServerState) ProcessOperation(op operation.Operation) (bool, operation.Operation, int) {
	isValid := true
	clientRevision := op.Revision

	// Transformamos la operación contra el historial reciente
	for i := clientRevision; i < len(state.RevisionLog); i++ {
		appliedOp := state.RevisionLog[i]
		op, isValid = operation.Transform(op, appliedOp)
		// Si se anula, cortamos el bucle.
		if !isValid {
			break
		}
	}

	if !isValid {
		return false, op, len(state.RevisionLog)
	}

	// La operación superó las transformaciones, la guardamos
	state.RevisionLog = append(state.RevisionLog, op)
	return true, op, len(state.RevisionLog)
}
