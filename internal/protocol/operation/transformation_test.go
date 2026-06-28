package operation

import (
	"testing"
)

func TestTransform(t *testing.T) {
	tests := []struct {
		name          string
		incoming      Operation
		applied       Operation
		expectedValid bool
		expectedPos   int
	}{
		// ==========================================
		// INSERT vs INSERT
		// ==========================================
		{
			name:          "Insert vs Insert - Applied ANTES (Desplaza a la derecha)",
			incoming:      Operation{Op: Insert, Position: 3, Content: "A"},
			applied:       Operation{Op: Insert, Position: 1, Content: "B"},
			expectedValid: true,
			expectedPos:   4,
		},
		{
			name:          "Insert vs Insert - Applied DESPUES (No afecta)",
			incoming:      Operation{Op: Insert, Position: 3, Content: "A"},
			applied:       Operation{Op: Insert, Position: 5, Content: "B"},
			expectedValid: true,
			expectedPos:   3,
		},
		{
			name:          "Insert vs Insert - EMPATE - Applied gana lexicográficamente",
			incoming:      Operation{Op: Insert, Position: 2, Content: "Z"},
			applied:       Operation{Op: Insert, Position: 2, Content: "A"}, // "A" < "Z"
			expectedValid: true,
			expectedPos:   3, // Z se desplaza
		},
		{
			name:          "Insert vs Insert - EMPATE - Incoming gana lexicográficamente",
			incoming:      Operation{Op: Insert, Position: 2, Content: "A"},
			applied:       Operation{Op: Insert, Position: 2, Content: "Z"}, // "Z" > "A"
			expectedValid: true,
			expectedPos:   2, // A se queda en su lugar
		},

		// ==========================================
		// INSERT vs DELETE
		// ==========================================
		{
			name:          "Insert vs Delete - Borraron ANTES (Desplaza a la izquierda)",
			incoming:      Operation{Op: Insert, Position: 4, Content: "X"},
			applied:       Operation{Op: Delete, Position: 2},
			expectedValid: true,
			expectedPos:   3,
		},
		{
			name:          "Insert vs Delete - Borraron DESPUES (No afecta)",
			incoming:      Operation{Op: Insert, Position: 4, Content: "X"},
			applied:       Operation{Op: Delete, Position: 5},
			expectedValid: true,
			expectedPos:   4,
		},

		// ==========================================
		// DELETE vs INSERT
		// ==========================================
		{
			name:          "Delete vs Insert - Insertaron ANTES o IGUAL (Desplaza a la derecha)",
			incoming:      Operation{Op: Delete, Position: 3},
			applied:       Operation{Op: Insert, Position: 1, Content: "X"},
			expectedValid: true,
			expectedPos:   4,
		},
		{
			name:          "Delete vs Insert - Insertaron DESPUES (No afecta)",
			incoming:      Operation{Op: Delete, Position: 3},
			applied:       Operation{Op: Insert, Position: 5, Content: "X"},
			expectedValid: true,
			expectedPos:   3,
		},

		// ==========================================
		// DELETE vs DELETE
		// ==========================================
		{
			name:          "Delete vs Delete - Borraron ANTES (Desplaza a la izquierda)",
			incoming:      Operation{Op: Delete, Position: 4},
			applied:       Operation{Op: Delete, Position: 2},
			expectedValid: true,
			expectedPos:   3,
		},
		{
			name:          "Delete vs Delete - EMPATE (Operación se anula)",
			incoming:      Operation{Op: Delete, Position: 4},
			applied:       Operation{Op: Delete, Position: 4},
			expectedValid: false,
			expectedPos:   4, // La posicion no importa porque is valid es false, pero la verificamos igual.
		},
		{
			name:          "Delete vs Delete - Borraron DESPUES (No afecta)",
			incoming:      Operation{Op: Delete, Position: 4},
			applied:       Operation{Op: Delete, Position: 6},
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
