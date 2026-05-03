package protocol

import "encoding/json"

const (
	MessageTypeOperation = "operation"
	MessageTypeAck       = "ack"
)

// Message envuelve distintos tipos de mensajes en el mismo WebSocket.
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// OperationMessage representa los comandos insert(c,p) y delete(p).
type OperationMessage struct {
	Op       Operation `json:"op"`
	Position int       `json:"pos"`
	Content  string    `json:"content,omitempty"` // solo para insert
}

// AckMessage confirma procesamiento con revision actual.
type AckMessage struct {
	Revision int `json:"revision"`
}

func NewMessage(messageType string, payload any) (Message, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return Message{}, err
	}

	return Message{
		Type:    messageType,
		Payload: data,
	}, nil
}

func (m Message) DecodeOperation() (OperationMessage, error) {
	var op OperationMessage
	err := json.Unmarshal(m.Payload, &op)
	return op, err
}

func (m Message) DecodeAck() (AckMessage, error) {
	var ack AckMessage
	err := json.Unmarshal(m.Payload, &ack)
	return ack, err
}
