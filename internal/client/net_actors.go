package client

import (
	"context"
	"errors"
	"sync"
	"time"

	"proyecto-final-redes/internal/protocol/message"
)

const retryTimeout = 10 * time.Second

type InFlightOperation struct {
	mu       sync.RWMutex
	op       message.Operation
	attempts int
	received bool
}

func NewInFlightOperation(op message.Operation) *InFlightOperation {
	return &InFlightOperation{
		op:       op,
		attempts: 3,
		received: false,
	}
}

func (i *InFlightOperation) Op() message.Operation {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.op
}

func (i *InFlightOperation) Attempts() int {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.attempts
}

func (i *InFlightOperation) DecrementAttempt() {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.attempts > 0 {
		i.attempts--
	}
}

func (i *InFlightOperation) IsReceived() bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.received
}

func (i *InFlightOperation) SetReceived(received bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.received = received
}

type Sender struct {
	Conn        WSConn
	Outbound    <-chan message.Operation
	NetworkAcks <-chan message.AckApplied
	CancelConn  context.CancelFunc
	Manager     *ConnectionManager
}

func NewSender(conn WSConn, outbound <-chan message.Operation, networkAcks <-chan message.AckApplied, cancel context.CancelFunc, manager *ConnectionManager) *Sender {
	return &Sender{
		Conn:        conn,
		Outbound:    outbound,
		NetworkAcks: networkAcks,
		CancelConn:  cancel,
		Manager:     manager,
	}
}

func (s *Sender) Run(ctx context.Context) error {
	defer s.CancelConn()

	var pendingOp *InFlightOperation
	var retryCh <-chan time.Time
	var ticker *time.Ticker

	for {
		select {
		case <-ctx.Done():
			if ticker != nil {
				ticker.Stop()
			}
			return nil

		// Enviamos una nueva operación con reintentos automáticos
		case op := <-s.Outbound:
			pendingOp = NewInFlightOperation(op)
			s.Manager.SetInFlightOp(pendingOp)

			if err := s.writeToSocket(pendingOp.Op(), ctx); err != nil {
				return err
			}

			if ticker != nil {
				ticker.Stop()
			}
			ticker = time.NewTicker(retryTimeout)
			retryCh = ticker.C

		// Al recibir el ack de la operación enviada, dejamos de reintentar
		case ack := <-s.NetworkAcks:
			if pendingOp != nil && pendingOp.Op().Revision == ack.ClientRevision {
				pendingOp.SetReceived(true)
				if ticker != nil {
					ticker.Stop()
				}
				retryCh = nil
			}

		// Si pasa el tiempo y no recibimos respuesta, reintentamos
		case <-retryCh:
			if pendingOp != nil && !pendingOp.IsReceived() {
				pendingOp.DecrementAttempt()
				if pendingOp.Attempts() > 0 {
					if err := s.writeToSocket(pendingOp.Op(), ctx); err != nil {
						return err
					}
				} else {
					if ticker != nil {
						ticker.Stop()
					}
					retryCh = nil
					select {
					case s.Manager.Notifications <- "Error: Se agotaron los reintentos de red":
					default:
					}
					return errors.New("timeout de red agotado")
				}
			}
		}
	}
}

func (s *Sender) writeToSocket(op message.Operation, ctx context.Context) error {
	msg, err := message.NewMessage(message.MessageTypeOperation, op)
	if err != nil {
		select {
		case s.Manager.Notifications <- "Error: Mensaje no pudo ser serializado":
		default:
		}
		return nil // No tiramos la conexión por un error de serialización
	}

	if err := s.Conn.WriteJSON(msg); err != nil {
		return errors.New("Connexión cerrada por el servidor")
	}

	return nil
}

type Listener struct {
	Conn          WSConn
	Incoming      chan<- message.ServerEvent
	NetworkAcks   chan<- message.AckApplied
	CancelConn    context.CancelFunc
	Notifications chan<- string
	Manager       *ConnectionManager
}

func NewListener(conn WSConn, incoming chan<- message.ServerEvent, networkAcks chan<- message.AckApplied, cancel context.CancelFunc, notifs chan<- string, manager *ConnectionManager) *Listener {
	return &Listener{
		Conn:          conn,
		Incoming:      incoming,
		NetworkAcks:   networkAcks,
		CancelConn:    cancel,
		Notifications: notifs,
		Manager:       manager,
	}
}

func (l *Listener) Run(ctx context.Context) error {
	defer l.CancelConn()

	for {
		if ctx.Err() != nil {
			return nil
		}

		var msg message.Message
		if err := l.Conn.ReadJSON(&msg); err != nil {
			return errors.New("Conexión cerrada por el servidor")
		}

		switch msg.Type {
		// Si recibimos una operación, la enviamos al motor de sincronización
		case message.MessageTypeOperation:
			op, err := msg.DecodeOperation()
			if err == nil {
				select {
				case <-ctx.Done():
					return nil
				case l.Incoming <- op:
				}
			} else {
				select {
				case l.Notifications <- "Operacion invalida recibida del servidor.":
				default:
				}
			}

		// Si recibimos un ack, lo enviamos:
		// - al sender para que deje de reintentar
		// - al motor de sincronización para que avance a la siguiente operación pendiente
		case message.MessageTypeAckApplied:
			ackMsg, err := msg.DecodeAckApplied()
			if err == nil {
				select {
				case <-ctx.Done():
					return nil
				case l.Incoming <- ackMsg:
				}
				select {
				case <-ctx.Done():
					return nil
				case l.NetworkAcks <- ackMsg:
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
