package client

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
	"proyecto-final-redes/internal/protocol/message"
	"proyecto-final-redes/internal/protocol/operation"
)

type Sender struct {
	Conn       *websocket.Conn
	Outbound   <-chan operation.Operation
	CancelConn context.CancelFunc
	Manager    *ConnectionManager
}

func NewSender(conn *websocket.Conn, outbound <-chan operation.Operation, cancel context.CancelFunc, manager *ConnectionManager) *Sender {
	return &Sender{
		Conn:       conn,
		Outbound:   outbound,
		CancelConn: cancel,
		Manager:    manager,
	}
}

func (s *Sender) Run(ctx context.Context) {
	defer s.CancelConn()
	for {
		select {
		case <-ctx.Done():
			return
		case op := <-s.Outbound:
			msg, err := message.NewMessage(message.MessageTypeOperation, op)
			if err != nil {
				return
			}
			
			lat := s.Manager.GetLatency()
			if lat > 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(lat):
				}
			}
			
			if err := s.Conn.WriteJSON(msg); err != nil {
				return
			}
		}
	}
}

type Listener struct {
	Conn          *websocket.Conn
	RemoteOps     chan<- operation.Operation
	Acks          chan<- int
	CancelConn    context.CancelFunc
	Notifications chan<- string
	Manager       *ConnectionManager
}

func NewListener(conn *websocket.Conn, remoteOps chan<- operation.Operation, acks chan<- int, cancel context.CancelFunc, notifs chan<- string, manager *ConnectionManager) *Listener {
	return &Listener{
		Conn:          conn,
		RemoteOps:     remoteOps,
		Acks:          acks,
		CancelConn:    cancel,
		Notifications: notifs,
		Manager:       manager,
	}
}

func (l *Listener) Run(ctx context.Context) {
	defer l.CancelConn()

	var expectedSeqNum int
	for {
		if ctx.Err() != nil {
			return
		}

		var msg message.Message
		if err := l.Conn.ReadJSON(&msg); err != nil {
			return
		}

		if msg.SeqNum != 0 && msg.SeqNum != expectedSeqNum {
			select {
			case l.Notifications <- "Error: Secuencia de paquete invalida.":
			default:
			}
			return
		}
		expectedSeqNum = msg.SeqNum + 1

		lat := l.Manager.GetLatency()
		if lat > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(lat):
			}
		}

		switch msg.Type {
		case message.MessageTypeOperation:
			op, err := msg.DecodeOperation()
			if err == nil {
				select {
				case <-ctx.Done():
					return
				case l.RemoteOps <- op:
				}
			} else {
				select {
				case l.Notifications <- "Operacion invalida recibida del servidor.":
				default:
				}
			}
		case message.MessageTypeAck:
			ackMsg, err := msg.DecodeAck()
			if err == nil {
				select {
				case <-ctx.Done():
					return
				case l.Acks <- ackMsg.Revision:
				}
			} else {
				select {
				case l.Notifications <- "Ack invalido recibido del servidor.":
				default:
				}
			}
		default:
			select {
			case l.Notifications <- "Tipo de mensaje desconocido recibido del servidor.":
			default:
			}
		}
	}
}
