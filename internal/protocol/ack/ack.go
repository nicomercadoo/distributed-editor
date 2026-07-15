package ack

// AckApplied confirma procesamiento con revision actual.
type AckApplied struct {
	Revision int `json:"revision"`
}

func (AckApplied) IsServerEvent() {}

// AckReceived confirma recepción de un ack con la revision procesada.
type AckReceived struct {
	Received bool `json:"received"`
}

func (AckReceived) IsServerEvent() {}
