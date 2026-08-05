package message

// Operation representa los comandos insert(c,p) y delete(p).
type Operation struct {
	ID       string        `json:"id"`
	Op       OperationType `json:"op"`
	Position int           `json:"pos"`
	Content  string        `json:"content,omitempty"` // Solo para insert
	Revision int           `json:"revision"`
}

func (Operation) IsServerEvent() {}
