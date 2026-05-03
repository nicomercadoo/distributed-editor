package message

import (
	"encoding/json"

	"proyecto-final-redes/internal/protocol/ack"
	"proyecto-final-redes/internal/protocol/operation"
)

const (
	MessageTypeOperation = "operation"
	MessageTypeAck       = "ack"
)

// Message envuelve distintos tipos de mensajes en el mismo WebSocket.
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
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

func (m Message) DecodeOperation() (operation.Operation, error) {
	var op operation.Operation
	err := json.Unmarshal(m.Payload, &op)
	return op, err
}

func (m Message) DecodeAck() (ack.Ack, error) {
	var ackMsg ack.Ack
	err := json.Unmarshal(m.Payload, &ackMsg)
	return ackMsg, err
}
