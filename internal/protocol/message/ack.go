package message

// AckApplied confirma procesamiento con revision actual.
type AckApplied struct {
	OperationID    string `json:"operation_id"`
	ServerRevision int    `json:"server_revision"`
}

func (AckApplied) IsServerEvent() {}
