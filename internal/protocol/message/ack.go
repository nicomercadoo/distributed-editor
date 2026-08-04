package message

// AckApplied confirma procesamiento con revision actual.
type AckApplied struct {
	ClientRevision int `json:"client_revision"`
	ServerRevision int `json:"server_revision"`
}

func (AckApplied) IsServerEvent() {}
