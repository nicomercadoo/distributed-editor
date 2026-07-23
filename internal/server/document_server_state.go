package server

import (
	"encoding/json"
	"log"
	"os"

	"proyecto-final-redes/internal/protocol/operation"
)

const serverHistoryFile = "server_history.json"

type DocumentServerState struct {
	RevisionLog []operation.Operation
}

func NewDocumentServerState(clear bool) *DocumentServerState {
	state := &DocumentServerState{
		RevisionLog: make([]operation.Operation, 0),
	}

	if clear {
		log.Println("Limpiando el historial de operaciones...")
		state.saveState()
	} else {
		data, err := os.ReadFile(serverHistoryFile)
		if err == nil {
			err = json.Unmarshal(data, &state.RevisionLog)
			if err != nil {
				log.Printf("Error al parsear %s, iniciando historial vacío: %v\n", serverHistoryFile, err)
			} else {
				log.Printf("Historial cargado desde %s con %d operaciones.\n", serverHistoryFile, len(state.RevisionLog))
			}
		} else if !os.IsNotExist(err) {
			log.Printf("No se pudo leer %s, iniciando historial vacío: %v\n", serverHistoryFile, err)
		} else {
			log.Printf("No existe %s, iniciando documento nuevo.\n", serverHistoryFile)
		}
	}

	return state
}

func (state *DocumentServerState) saveState() {
	data, err := json.MarshalIndent(state.RevisionLog, "", "  ")
	if err != nil {
		log.Printf("Error al serializar historial: %v\n", err)
		return
	}
	err = os.WriteFile(serverHistoryFile, data, 0644)
	if err != nil {
		log.Printf("Error al escribir el historial en disco: %v\n", err)
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
	state.saveState() // Persistir estado

	return true, op, len(state.RevisionLog)
}
