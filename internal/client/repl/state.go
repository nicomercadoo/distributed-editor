package repl

import (
	"sync"

	"github.com/gorilla/websocket"

	"proyecto-final-redes/internal/protocol/document"
	"proyecto-final-redes/internal/protocol/operation"
)

type State struct {
	conn             *websocket.Conn
	listenerDone     chan struct{}
	mu               *sync.Mutex
	latency          int
	LastSyncRevision document.ID
	PendingChanges   []operation.Operation
	SentChanges      []operation.Operation
	Document         document.DocumentRep
}

func NewState() *State {
	return &State{
		conn:             nil,
		listenerDone:     nil,
		mu:               &sync.Mutex{},
		LastSyncRevision: 0,
		PendingChanges:   make([]operation.Operation, 0),
		SentChanges:      make([]operation.Operation, 0),
		Document:         "",
	}
}

func (s *State) IsConnected() bool {
	return s.conn != nil
}

func (s *State) SetConnection(conn *websocket.Conn) {
	s.conn = conn
}

func (s *State) SetListenerDone(done chan struct{}) {
	s.listenerDone = done
}

func (s *State) ListenerDone() chan struct{} {
	return s.listenerDone
}

func (s *State) SetLatency(ms int) {
    s.mu.Lock()
    s.latency = ms
    s.mu.Unlock()
}

func (s *State) Latency() int {
    s.mu.Lock()
    defer s.mu.Unlock()
    return s.latency
}

func (s *State) Mutex() *sync.Mutex {
	return s.mu
}
