package ack

// Ack confirma procesamiento con revision actual.
type Ack struct {
	Revision int `json:"revision"`
}
