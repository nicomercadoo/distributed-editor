package operation

// Operation representa los comandos insert(c,p) y delete(p).
type Operation struct {
	Op       OperationType `json:"op"`
	Position int           `json:"pos"`
	Content  string        `json:"content,omitempty"` // solo para insert
}
