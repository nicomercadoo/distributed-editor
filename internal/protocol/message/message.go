package message

import (
	"encoding/json"

	"proyecto-final-redes/internal/protocol/ack"
	"proyecto-final-redes/internal/protocol/operation"
)

type ServerEvent interface {
	IsServerEvent()
}

type MessageType = string

const (
	MessageTypeOperation    = "operation"
	MessageTypeAckReceived  = "ack_received"
	MessageTypeAckApplied   = "ack_applied"
)

type Message struct {
	Type     string          `json:"type"`
	SeqNum   int             `json:"seqNum,omitempty"`
	Revision int             `json:"revision,omitempty"`
	Payload  json.RawMessage `json:"payload"`
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

func (m Message) DecodeOperation() (operation.Operation, error) {
	var op operation.Operation
	err := json.Unmarshal(m.Payload, &op)
	return op, err
}

func (m Message) DecodeAckReceived() (ack.AckReceived, error) {
	var ackMsg ack.AckReceived
	err := json.Unmarshal(m.Payload, &ackMsg)
	return ackMsg, err
}

func (m Message) DecodeAckApplied() (ack.AckApplied, error) {
	var ackMsg ack.AckApplied
	err := json.Unmarshal(m.Payload, &ackMsg)
	return ackMsg, err
}
