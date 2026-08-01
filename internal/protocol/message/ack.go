package message

// AckReceived confirma recepción de operacion por parte del servidor.
type AckReceived struct {
	Revision int `json:"revision"`
}

// AckApplied confirma procesamiento con revision actual.
type AckApplied struct {
	Revision int `json:"revision"`
}

func (AckApplied) IsServerEvent() {}
