package message

import (
	"encoding/json"
)

type ServerEvent interface {
	IsServerEvent()
}

type MessageType = string

const (
	MessageTypeOperation   = "operation"
	MessageTypeAckReceived = "ack_received"
	MessageTypeAckApplied  = "ack_applied"
)

type Message struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func NewMessage(messageType MessageType, payload any) (Message, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return Message{}, err
	}

	return Message{
		Type:    messageType,
		Payload: data,
	}, nil
}

func (m Message) DecodeOperation() (Operation, error) {
	var op Operation
	err := json.Unmarshal(m.Payload, &op)
	return op, err
}

func (m Message) DecodeAckReceived() (AckReceived, error) {
	var ackMsg AckReceived
	err := json.Unmarshal(m.Payload, &ackMsg)
	return ackMsg, err
}

func (m Message) DecodeAckApplied() (AckApplied, error) {
	var ackMsg AckApplied
	err := json.Unmarshal(m.Payload, &ackMsg)
	return ackMsg, err
}
