//go:build !js || !wasm

package client

import (
	"github.com/gorilla/websocket"
)

func (m *ConnectionManager) dial(uri string) (WSConn, error) {
	c, _, err := websocket.DefaultDialer.Dial(uri, nil)
	if err != nil {
		return nil, err
	}
	return c, nil
}
