package repl

import (
	"sync"

	"github.com/gorilla/websocket"

	"proyecto-final-redes/internal/protocol/document"
	"proyecto-final-redes/internal/protocol/operation"
)

type SentChange struct {
    Op       operation.Operation
    Received bool
}

type State struct {
	conn                *websocket.Conn
	senderDone          chan struct{} // Canal para indicar que la goroutine de envío al servidor ha terminado
	listenerDone        chan struct{} // Canal para indicar que la goroutine de escucha del servidor ha terminado
	notifications       chan string
	mu                  *sync.RWMutex
	latency             int
	promptSymbol         string
	LastSyncRevision    document.ID
	PendingChanges      []operation.Operation
	SentChange          SentChange
	Document            document.DocumentRep
}

func NewState() *State {
	return &State{
		conn:               nil,
		listenerDone:       nil,
		senderDone:         nil,
		notifications:      make(chan string),
		mu:                 &sync.RWMutex{},
		promptSymbol:       "> ",
		LastSyncRevision:   0,
		PendingChanges:     make([]operation.Operation, 0),
		SentChange:         SentChange{Op: operation.Operation{}, Received: false},
		Document:           "",
	}
}

func (s *State) IsConnected() bool {
    s.mu.RLock()
    defer s.mu.RUnlock()
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

func (s *State) SetSenderDone(done chan struct{}) {
    s.senderDone = done
}

func (s *State) SenderDone() chan struct{} {
    return s.senderDone
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

func (s *State) Mutex() *sync.RWMutex {
	return s.mu
}
