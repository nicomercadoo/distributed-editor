package ack

// AckApplied confirma procesamiento con revision actual.
type AckApplied struct {
	Revision int `json:"revision"`
}

func (AckApplied) IsServerEvent() {}
