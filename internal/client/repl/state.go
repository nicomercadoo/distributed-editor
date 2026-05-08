package repl

import (
    "github.com/gorilla/websocket"
)

type State struct {
	conn *websocket.Conn
}

func NewState() *State {
	return &State{
	    conn: nil,
	}
}

func (s *State) IsConnected() bool {
	return s.conn != nil
}

func (s *State) SetConnection(conn *websocket.Conn) {
	s.conn = conn
}
