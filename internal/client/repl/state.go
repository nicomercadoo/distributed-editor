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
	lastSyncRevision    document.ID
	PendingChanges      []operation.Operation
	SentChange          SentChange
	CommitedChanges     []operation.Operation
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
		lastSyncRevision:   0,
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

func (s *State) AddCommitedChange(op operation.Operation) {
    s.mu.Lock()
    s.CommitedChanges = append(s.CommitedChanges, op)
    s.mu.Unlock()
}

func (s *State) AddPendingChange(op operation.Operation) {
    s.mu.Lock()
    s.PendingChanges = append(s.PendingChanges, op)
    s.mu.Unlock()
}

func (s *State) GetFirstPendingChange() (operation.Operation, bool) {
    s.mu.Lock()
    defer s.mu.Unlock()
    if len(s.PendingChanges) == 0 {
        return operation.Operation{}, false
    }
    return s.PendingChanges[0], true
}

func (s *State) RemoveFirstPendingChange() {
    s.mu.Lock()
    if len(s.PendingChanges) > 0 {
        s.PendingChanges = s.PendingChanges[1:]
    }
    s.mu.Unlock()
}

func (s *State) SetSentChange(op operation.Operation) {
    s.mu.Lock()
    s.SentChange = SentChange{Op: op, Received: false}
    s.mu.Unlock()
}

func (s *State) MarkSentChangeReceived() {
    s.mu.Lock()
    s.SentChange.Received = true
    s.mu.Unlock()
}

func (s *State) ClearSentChange() {
    s.mu.Lock()
    s.SentChange = SentChange{Op: operation.Operation{}, Received: false}
    s.mu.Unlock()
}

func (s *State) SentChangeIsEmpty() bool {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.SentChange == (SentChange{})
}

func (s *State) UpdateRevision(rev document.ID) {
    s.mu.Lock()
    s.lastSyncRevision = rev
    s.mu.Unlock()
}

func (s *State) GetRevision() document.ID {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.lastSyncRevision
}
