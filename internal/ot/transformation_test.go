package ot

import (
	"proyecto-final-redes/internal/protocol/message"
	"testing"
)

func TestTransform(t *testing.T) {
	tests := []struct {
		name          string
		incoming      message.Operation
		applied       message.Operation
		expectedValid bool
		expectedPos   int
	}{
		// ==========================================
		// INSERT vs INSERT
		// ==========================================
		{
			name:          "Insert vs message.Insert - Applied ANTES (Desplaza a la derecha)",
			incoming:      message.Operation{Op: message.Insert, Position: 3, Content: "A"},
			applied:       message.Operation{Op: message.Insert, Position: 1, Content: "B"},
			expectedValid: true,
			expectedPos:   4,
		},
		{
			name:          "Insert vs message.Insert - Applied DESPUES (No afecta)",
			incoming:      message.Operation{Op: message.Insert, Position: 3, Content: "A"},
			applied:       message.Operation{Op: message.Insert, Position: 5, Content: "B"},
			expectedValid: true,
			expectedPos:   3,
		},
		{
			name:          "Insert vs message.Insert - EMPATE - Applied gana lexicográficamente",
			incoming:      message.Operation{Op: message.Insert, Position: 2, Content: "Z"},
			applied:       message.Operation{Op: message.Insert, Position: 2, Content: "A"}, // "A" < "Z"
			expectedValid: true,
			expectedPos:   3, // Z se desplaza
		},
		{
			name:          "Insert vs message.Insert - EMPATE - Incoming gana lexicográficamente",
			incoming:      message.Operation{Op: message.Insert, Position: 2, Content: "A"},
			applied:       message.Operation{Op: message.Insert, Position: 2, Content: "Z"}, // "Z" > "A"
			expectedValid: true,
			expectedPos:   2, // A se queda en su lugar
		},

		// ==========================================
		// INSERT vs DELETE
		// ==========================================
		{
			name:          "Insert vs message.Delete - Borraron ANTES (Desplaza a la izquierda)",
			incoming:      message.Operation{Op: message.Insert, Position: 4, Content: "X"},
			applied:       message.Operation{Op: message.Delete, Position: 2},
			expectedValid: true,
			expectedPos:   3,
		},
		{
			name:          "Insert vs message.Delete - Borraron DESPUES (No afecta)",
			incoming:      message.Operation{Op: message.Insert, Position: 4, Content: "X"},
			applied:       message.Operation{Op: message.Delete, Position: 5},
			expectedValid: true,
			expectedPos:   4,
		},

		// ==========================================
		// DELETE vs INSERT
		// ==========================================
		{
			name:          "Delete vs message.Insert - message.Insertaron ANTES o IGUAL (Desplaza a la derecha)",
			incoming:      message.Operation{Op: message.Delete, Position: 3},
			applied:       message.Operation{Op: message.Insert, Position: 1, Content: "X"},
			expectedValid: true,
			expectedPos:   4,
		},
		{
			name:          "Delete vs message.Insert - message.Insertaron DESPUES (No afecta)",
			incoming:      message.Operation{Op: message.Delete, Position: 3},
			applied:       message.Operation{Op: message.Insert, Position: 5, Content: "X"},
			expectedValid: true,
			expectedPos:   3,
		},

		// ==========================================
		// DELETE vs DELETE
		// ==========================================
		{
			name:          "Delete vs message.Delete - Borraron ANTES (Desplaza a la izquierda)",
			incoming:      message.Operation{Op: message.Delete, Position: 4},
			applied:       message.Operation{Op: message.Delete, Position: 2},
			expectedValid: true,
			expectedPos:   3,
		},
		{
			name:          "Delete vs message.Delete - EMPATE (Operación se anula)",
			incoming:      message.Operation{Op: message.Delete, Position: 4},
			applied:       message.Operation{Op: message.Delete, Position: 4},
			expectedValid: false,
			expectedPos:   4, // La posicion no importa porque is valid es false, pero la verificamos igual.
		},
		{
			name:          "Delete vs message.Delete - Borraron DESPUES (No afecta)",
			incoming:      message.Operation{Op: message.Delete, Position: 4},
			applied:       message.Operation{Op: message.Delete, Position: 6},
			expectedValid: true,
			expectedPos:   4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultOp, isValid := Transform(tt.incoming, tt.applied)

			if isValid != tt.expectedValid {
				t.Fatalf("Esperaba valid=%v, obtuve %v", tt.expectedValid, isValid)
			}

			if isValid && resultOp.Position != tt.expectedPos {
				t.Errorf("Esperaba posicion %d, obtuve %d", tt.expectedPos, resultOp.Position)
			}
		})
	}
}
