package protocol

// Operation representa los comandos insert(c,p) y delete(p)
type Operation struct {
	Type string `json:"type"` // "insert" o "delete"
	Char string `json:"c,omitempty"`
	Pos  int    `json:"p"`
}