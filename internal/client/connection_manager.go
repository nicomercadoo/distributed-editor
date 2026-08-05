package client

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"proyecto-final-redes/internal/protocol/message"
)

type ConnectionManager struct {
	mu sync.RWMutex

	conn          WSConn
	activeConnCtx context.Context
	cancelConn    context.CancelFunc

	// Canal de entrada para operaciones y acks
	Incoming chan<- message.ServerEvent
	// Canal de salida para operaciones
	Outbound chan message.Operation

	Notifications    chan<- string
	revisionProvider func() int

	inFlightOp *InFlightOperation

	ctx context.Context
}

func NewConnectionManager(ctx context.Context, incoming chan<- message.ServerEvent, outbound chan message.Operation, notifications chan<- string, getRevision func() int) *ConnectionManager {
	return &ConnectionManager{
		Incoming:         incoming,
		Outbound:         outbound,
		Notifications:    notifications,
		revisionProvider: getRevision,
		ctx:              ctx,
	}
}

// Connect intenta conectar al servidor en la URI especificada
func (m *ConnectionManager) Connect(uri string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.conn != nil {
		return errors.New("Ya hay una conexión activa.")
	}

	// Obtenemos la revisión actual y agregamos el parámetro a la URI
	rev := 0
	if m.revisionProvider != nil {
		rev = m.revisionProvider()
	}
	uriWithRev := fmt.Sprintf("%s?revision=%d", uri, rev)

	c, err := m.dial(uriWithRev)
	if err != nil {
		return err
	}

	m.conn = c
	m.activeConnCtx, m.cancelConn = context.WithCancel(m.ctx)

	networkAcks := make(chan message.AckApplied, 10)

	// Enviamos la operación que quedó en vuelo si existe
	// Esto solo ocurre si nos reconectamos
	if m.inFlightOp != nil && !m.inFlightOp.IsReceived() {
		m.Outbound <- m.inFlightOp.Op()
	}

	// Lanzamos los actores de red
	sender := NewSender(m.conn, m.Outbound, networkAcks, m.cancelConn, m)
	listener := NewListener(m.conn, m.Incoming, networkAcks, m.cancelConn, m.Notifications, m)

	go func() {
		if err := sender.Run(m.activeConnCtx); err != nil {
			select {
			case m.Notifications <- "Error: " + err.Error():
			default:
			}
			m.Disconnect()
		}
	}()
	go func() {
		if err := listener.Run(m.activeConnCtx); err != nil {
			select {
			case m.Notifications <- "Error: " + err.Error():
			default:
			}
			m.Disconnect()
		}
	}()

	select {
	case m.Notifications <- "Conectado a " + uri:
	default:
	}

	return nil
}

func (m *ConnectionManager) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.conn == nil {
		return nil
	}

	if m.cancelConn != nil {
		m.cancelConn()
		m.cancelConn = nil
	}

	err := m.conn.Close()
	m.conn = nil

	select {
	case m.Notifications <- "Desconectado":
	default:
	}

	return err
}

func (m *ConnectionManager) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.conn != nil
}

func (m *ConnectionManager) GetInFlightAttempts() int {
	m.mu.RLock()
	op := m.inFlightOp
	m.mu.RUnlock()

	if op == nil {
		return 0
	}
	return op.Attempts()
}

func (m *ConnectionManager) IsInFlightReceived() bool {
	m.mu.RLock()
	op := m.inFlightOp
	m.mu.RUnlock()

	if op == nil {
		return false
	}
	return op.IsReceived()
}

func (m *ConnectionManager) SetInFlightOp(op *InFlightOperation) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inFlightOp = op
}
