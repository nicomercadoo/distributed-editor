package webui

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"proyecto-final-redes/internal/client"
	"proyecto-final-redes/internal/protocol/message"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Server struct {
	localOps chan<- message.Operation
	ctrl     client.AppController
	port     int
}

func NewServer(localOps chan<- message.Operation, ctrl client.AppController, port int) *Server {
	return &Server{
		localOps: localOps,
		ctrl:     ctrl,
		port:     port,
	}
}

func (s *Server) Run(ctx context.Context) {
	mux := http.NewServeMux()

	// Serve static files (index.html)
	mux.Handle("/", http.FileServer(http.Dir("./web_ui")))

	// Serve WebSocket endpoint
	mux.HandleFunc("/ws", s.handleWS)

	addr := fmt.Sprintf(":%d", s.port)
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()

	log.Printf("Web UI listening on http://localhost%s\n", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %s\n", err)
	}
}

type StateMsg struct {
	Document     string `json:"document"`
	Revision     int    `json:"revision"`
	PendingCount int    `json:"pendingCount"`
	HasInFlight  bool   `json:"hasInFlight"`
	IsConnected  bool   `json:"isConnected"`
}

type ClientMsg struct {
	Type string `json:"type"` // "insert", "delete", "cmd_connect", "cmd_disconnect"
	Pos  int    `json:"pos,omitempty"`
	Char string `json:"char,omitempty"`
	URI  string `json:"uri,omitempty"`
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer c.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		s.pushLoop(ctx, c)
	}()

	go func() {
		defer wg.Done()
		defer cancel()
		s.readLoop(c)
	}()

	wg.Wait()
}

func (s *Server) pushLoop(ctx context.Context, c *websocket.Conn) {
	ticker := time.NewTicker(16 * time.Millisecond) // ~60fps
	defer ticker.Stop()

	var lastState StateMsg
	// force initial send
	lastState.Revision = -1

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			currentState := StateMsg{
				Document:     s.ctrl.GetDocument(),
				Revision:     s.ctrl.GetRevision(),
				PendingCount: s.ctrl.GetPendingCount(),
				HasInFlight:  s.ctrl.HasInFlight(),
				IsConnected:  s.ctrl.IsConnected(),
			}

			if currentState != lastState {
				err := c.WriteJSON(currentState)
				if err != nil {
					log.Println("write json:", err)
					return
				}
				lastState = currentState
			}
		}
	}
}

func (s *Server) readLoop(c *websocket.Conn) {
	for {
		var msg ClientMsg
		err := c.ReadJSON(&msg)
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Println("read json:", err)
			}
			break
		}

		switch msg.Type {
		case "insert":
			s.localOps <- message.Operation{
				Op:       message.Insert,
				Position: msg.Pos,
				Content:  msg.Char,
				Revision: s.ctrl.GetRevision(),
			}
		case "delete":
			s.localOps <- message.Operation{
				Op:       message.Delete,
				Position: msg.Pos,
				Revision: s.ctrl.GetRevision(),
			}
		case "cmd_connect":
			if err := s.ctrl.Connect(msg.URI); err != nil {
				log.Println("connect error:", err)
			}
		case "cmd_disconnect":
			if err := s.ctrl.Disconnect(); err != nil {
				log.Println("disconnect error:", err)
			}
		}
	}
}
