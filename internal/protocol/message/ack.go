package message

// AckReceived confirma recepción de un ack con la revision procesada.
type AckReceived struct {
}

func (AckReceived) IsServerEvent() {}

// AckApplied confirma procesamiento con revision actual.
type AckApplied struct {
	Revision int `json:"revision"`
}

func (AckApplied) IsServerEvent() {}
