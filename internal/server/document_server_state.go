package server

import (
	"encoding/json"
	"log"
	"os"
	"proyecto-final-redes/internal/ot"
	"proyecto-final-redes/internal/protocol/message"
)

const serverHistoryFile = "server_history.json"

type DocumentServerState struct {
	RevisionLog  []message.Operation
	processedIDs map[string]bool
}

func NewDocumentServerState(clear bool) *DocumentServerState {
	state := &DocumentServerState{
		RevisionLog:  make([]message.Operation, 0),
		processedIDs: make(map[string]bool),
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

	// Llenamos el map processedIDs con las operaciones cargadas desde el historial
	for _, op := range state.RevisionLog {
		if op.ID != "" {
			state.processedIDs[op.ID] = true
		}
	}

	return state
}

// saveState guarda el estado del servidor en un archivo JSON
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

// IsDuplicate chequea si la operación con el ID dado ya fue procesada
func (state *DocumentServerState) IsDuplicate(id string) bool {
	if id == "" {
		return false
	}
	_, exists := state.processedIDs[id]
	return exists
}

// ProcessOperation toma una operación entrante, la transforma contra el historial reciente
// y la persiste en el historial del servidor
// Retorna:
// - isValid: si la operación sobrevivió a las transformaciones
// - transformedOp: la operación resultante
// - revision: el número de revisión actualizado del servidor
func (state *DocumentServerState) ProcessOperation(op message.Operation) (bool, message.Operation, int) {
	isValid := true
	clientRevision := op.Revision

	// Transformamos la operación contra el historial reciente
	for i := clientRevision; i < len(state.RevisionLog); i++ {
		appliedOp := state.RevisionLog[i]
		op, isValid = ot.Transform(op, appliedOp)
		// Si se anula, cortamos el bucle
		if !isValid {
			break
		}
	}

	if !isValid {
		return false, op, len(state.RevisionLog)
	}

	// La operación superó las transformaciones, la guardamos
	state.RevisionLog = append(state.RevisionLog, op)
	if op.ID != "" {
		state.processedIDs[op.ID] = true
	}
	state.saveState() // Persistir estado

	return true, op, len(state.RevisionLog)
}
