//go:build js && wasm

package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"syscall/js"
	"time"
)

type jsWSConn struct {
	ws       js.Value
	msgChan  chan []byte
	errChan  chan error
	close    chan struct{}
	isClosed bool
}

func (m *ConnectionManager) dial(uri string) (WSConn, error) {
	ws := js.Global().Get("WebSocket").New(uri)

	conn := &jsWSConn{
		ws:       ws,
		msgChan:  make(chan []byte, 100),
		errChan:  make(chan error, 1),
		close:    make(chan struct{}),
		isClosed: false,
	}

	openChan := make(chan struct{})
	errOpenChan := make(chan error)

	ws.Set("onopen", js.FuncOf(func(this js.Value, args []js.Value) any {
		close(openChan)
		return nil
	}))

	ws.Set("onerror", js.FuncOf(func(this js.Value, args []js.Value) any {
		if !conn.isClosed {
			select {
			case errOpenChan <- errors.New("WebSocket error"):
			default:
			}
		}
		return nil
	}))

	ws.Set("onmessage", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) > 0 {
			data := args[0].Get("data").String()
			select {
			case conn.msgChan <- []byte(data):
			default:
				// Drop or block? Unbuffered could block JS thread, buffered is better.
			}
		}
		return nil
	}))

	ws.Set("onclose", js.FuncOf(func(this js.Value, args []js.Value) any {
		if !conn.isClosed {
			conn.isClosed = true
			close(conn.close)
		}
		return nil
	}))

	select {
	case <-openChan:
		return conn, nil
	case err := <-errOpenChan:
		return nil, fmt.Errorf("Error al conectar via JS WebSocket: %w", err)
	}
}

func (c *jsWSConn) WriteJSON(v interface{}) error {
	if c.isClosed {
		return errors.New("WebSocket is closed")
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c.ws.Call("send", string(b))
	return nil
}

func (c *jsWSConn) ReadJSON(v interface{}) error {
	select {
	case msg := <-c.msgChan:
		return json.Unmarshal(msg, v)
	case <-c.close:
		return errors.New("WebSocket is closed")
	}
}

func (c *jsWSConn) Close() error {
	if !c.isClosed {
		c.isClosed = true
		c.ws.Call("close")
		close(c.close)
	}
	return nil
}

func (m *ConnectionManager) cleanupProxy() {
	// No-op en JS ya que no hay toxiproxy nativo
}

func (m *ConnectionManager) setLatencyNative(d time.Duration) error {
	// No se puede aplicar toxiproxy en el navegador directamente
	// a menos que se inyecte latencia artificial en este cliente JS.
	return errors.New("latencia no soportada nativamente en modo WebAssembly")
}
