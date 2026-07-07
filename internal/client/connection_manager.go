package client

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"proyecto-final-redes/internal/protocol/operation"
)

type ConnectionManager struct {
	mu sync.RWMutex

	conn          *websocket.Conn
	activeConnCtx context.Context
	cancelConn    context.CancelFunc
	latency       time.Duration

	// Data plane channels
	RemoteOps chan<- operation.Operation
	Acks      chan<- int
	Outbound  <-chan operation.Operation

	Notifications chan<- string
	
	// parent context
	ctx context.Context
}

func NewConnectionManager(ctx context.Context, remoteOps chan<- operation.Operation, acks chan<- int, outbound <-chan operation.Operation, notifications chan<- string) *ConnectionManager {
	return &ConnectionManager{
		RemoteOps:     remoteOps,
		Acks:          acks,
		Outbound:      outbound,
		Notifications: notifications,
		ctx:           ctx,
	}
}

func (m *ConnectionManager) Connect(uri string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.conn != nil {
		return errors.New("Ya hay una conexión activa.")
	}

	c, _, err := websocket.DefaultDialer.Dial(uri, nil)
	if err != nil {
		return fmt.Errorf("Error al conectar: %w", err)
	}

	m.conn = c
	m.activeConnCtx, m.cancelConn = context.WithCancel(m.ctx)

	sender := NewSender(m.conn, m.Outbound, m.cancelConn, m)
	listener := NewListener(m.conn, m.RemoteOps, m.Acks, m.cancelConn, m.Notifications, m)

	go sender.Run(m.activeConnCtx)
	go listener.Run(m.activeConnCtx)

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

func (m *ConnectionManager) SetLatency(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latency = d
}

func (m *ConnectionManager) GetLatency() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.latency
}
