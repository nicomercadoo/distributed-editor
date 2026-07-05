package client

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"proyecto-final-redes/internal/protocol/operation"
)

type ConnectionManager struct {
	ConnectCmd    <-chan string
	DisconnectCmd <-chan struct{}

	RemoteOps chan<- operation.Operation
	Acks      chan<- int
	Outbound  <-chan operation.Operation

	Notifications chan<- string
	LatencyCmd    <-chan int
}

func (m *ConnectionManager) Run(ctx context.Context) {
	var activeConnCtx context.Context
	var cancelConn context.CancelFunc
	var conn *websocket.Conn
	var latencyMs int

	for {
		select {
		case <-ctx.Done():
			if cancelConn != nil {
				cancelConn()
			}
			if conn != nil {
				conn.Close()
			}
			return

		case lat := <-m.LatencyCmd:
			latencyMs = lat
			// Latency update when not connected is kept in latencyMs state.
			// When connected, we could forward it to Sender via channel or context,
			// but we pass it during Sender initialization or via shared atomic.
			// To keep it 100% Actor based, we'll recreate or just rely on a channel we pass to Sender.
			// Actually, it's easier to pass a Latency channel to Sender and forward it here.
			// However, ConnectionManager receives latencyCmd. We can buffer it or just pass it to Sender if active.

		case uri := <-m.ConnectCmd:
			if conn != nil {
				select {
				case m.Notifications <- "Ya hay una conexión activa":
				default:
				}
				continue
			}
			c, _, err := websocket.DefaultDialer.Dial(uri, nil)
			if err != nil {
				select {
				case m.Notifications <- fmt.Sprintf("Error al conectar: %v", err):
				default:
				}
				continue
			}
			conn = c
			activeConnCtx, cancelConn = context.WithCancel(ctx)

			// Canal de latencia dedicado para este Sender
			senderLatCmd := make(chan int, 1)
			senderLatCmd <- latencyMs

			sender := &Sender{
				Conn:       conn,
				Outbound:   m.Outbound,
				LatencyCmd: senderLatCmd,
				CancelConn: cancelConn,
			}
			listener := &Listener{
				Conn:       conn,
				RemoteOps:  m.RemoteOps,
				Acks:       m.Acks,
				CancelConn: cancelConn,
				Notifications: m.Notifications,
			}

			go sender.Run(activeConnCtx)
			go listener.Run(activeConnCtx)

			select {
			case m.Notifications <- "Conectado a " + uri:
			default:
			}

		case <-m.DisconnectCmd:
			if cancelConn != nil {
				cancelConn()
				cancelConn = nil
			}
			if conn != nil {
				conn.Close()
				conn = nil
			}
			select {
			case m.Notifications <- "Desconectado":
			default:
			}
		}
	}
}
