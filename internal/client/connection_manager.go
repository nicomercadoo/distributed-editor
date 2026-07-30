package client

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"proyecto-final-redes/internal/protocol/message"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
	"github.com/gorilla/websocket"
)

type ConnectionManager struct {
	mu sync.RWMutex

	conn          *websocket.Conn
	activeConnCtx context.Context
	cancelConn    context.CancelFunc
	latency       time.Duration
	toxiProxy     *toxiproxy.Proxy

	// Data plane channels
	Incoming chan<- message.ServerEvent
	Outbound <-chan message.Operation

	Notifications    chan<- string
	revisionProvider func() int

	inFlightOp *InFlightOperation

	// parent context
	ctx context.Context
}

func NewConnectionManager(ctx context.Context, incoming chan<- message.ServerEvent, outbound <-chan message.Operation, notifications chan<- string, getRevision func() int) *ConnectionManager {
	return &ConnectionManager{
		Incoming:         incoming,
		Outbound:         outbound,
		Notifications:    notifications,
		revisionProvider: getRevision,
		ctx:              ctx,
	}
}

func (m *ConnectionManager) Connect(uri string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.conn != nil {
		return errors.New("Ya hay una conexión activa.")
	}

	rev := 0
	if m.revisionProvider != nil {
		rev = m.revisionProvider()
	}
	uriWithRev := fmt.Sprintf("%s?revision=%d", uri, rev)

	// Setup Toxiproxy if available
	proxyURI := uriWithRev
	u, err := url.Parse(uriWithRev)
	if err == nil {
		toxiClient := toxiproxy.NewClient("localhost:8474")
		proxyName := fmt.Sprintf("redes_proxy_%d", time.Now().UnixNano())

		p, errProxy := toxiClient.CreateProxy(proxyName, "localhost:0", u.Host)
		if errProxy == nil {
			m.toxiProxy = p
			u.Host = p.Listen
			proxyURI = u.String()
		} else {
			select {
			case m.Notifications <- "Aviso: Toxiproxy no está corriendo. Conectando directo a " + u.Host:
			default:
			}
		}
	}

	c, _, err := websocket.DefaultDialer.Dial(proxyURI, nil)
	if err != nil {
		return fmt.Errorf("Error al conectar: %w", err)
	}

	m.conn = c
	m.activeConnCtx, m.cancelConn = context.WithCancel(m.ctx)

	networkAcks := make(chan message.AckReceived, 10)

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

	if m.toxiProxy != nil {
		m.toxiProxy.Delete()
		m.toxiProxy = nil
	}

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

func (m *ConnectionManager) SetLatency(d time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.toxiProxy == nil && d > 0 {
		return errors.New("toxiproxy no está activo; no se puede aplicar latencia")
	}

	m.latency = d

	if m.toxiProxy != nil {
		latencyMs := d.Milliseconds()
		
		if d > 0 {
			_, err := m.toxiProxy.AddToxic("latency_toxic", "latency", "downstream", 1.0, toxiproxy.Attributes{
				"latency": latencyMs,
			})
			if err != nil {
				_, err = m.toxiProxy.UpdateToxic("latency_toxic", 1.0, toxiproxy.Attributes{
					"latency": latencyMs,
				})
				if err != nil {
					return fmt.Errorf("no se pudo configurar la latencia en toxiproxy: %w", err)
				}
			}
		} else {
			err := m.toxiProxy.RemoveToxic("latency_toxic")
			if err != nil && err.Error() != "not found" {
				// We can ignore error if it was not found
			}
		}
	}

	return nil
}

func (m *ConnectionManager) GetLatency() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.latency
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
