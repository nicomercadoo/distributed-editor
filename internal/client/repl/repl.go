package repl

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/chzyer/readline"
	"github.com/gorilla/websocket"

	"proyecto-final-redes/internal/protocol/message"
	"proyecto-final-redes/internal/protocol/operation"
)

type Handler func(*State, Command) error

type REPL struct {
	rl       *readline.Instance
	state    *State
	handlers map[string]Handler
	prompt_symbol   string
	prompt_connection string
	notifications chan string
	mu sync.RWMutex
}

func NewREPL(state *State) (*REPL, error) {
    prompt := "> "
	config := &readline.Config{
		Prompt:       prompt,
		HistoryFile:  ".client_history",
		AutoComplete: newCompleter(),
	}

	rl, err := readline.NewEx(config)
	if err != nil {
		return nil, err
	}

	repl := &REPL{
		rl:       rl,
		state:    state,
		handlers: map[string]Handler{},
		prompt_symbol: prompt,
		prompt_connection: "[Local]",
		notifications: make(chan string),
		mu: sync.RWMutex{},
	}

	repl.registerHandlers()
	return repl, nil
}

func Run() error {
	repl, err := NewREPL(NewState())
	if err != nil {
		return err
	}
	return repl.Run()
}

func (r *REPL) Close() {
	if r.rl != nil {
		_ = r.rl.Close()
	}
}

func (r *REPL) Run() error {
	defer r.Close()
	printHelp()

	go r.NotificationServer()

	for {
		line, err := r.getLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			fmt.Println("error:", err)
			continue
		}
		cmd, err := Parse(line)
		if err != nil {
			continue
		}

		if handler, ok := r.handlers[cmd.Name]; ok {
			if err := handler(r.state, cmd); err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				fmt.Println("error:", err)
				r.prompt_symbol = "!> "
			} else {
				r.prompt_symbol = "> "
			}
			continue
		}

		fmt.Println("comando desconocido. usa 'help'")
	}
}

func (r *REPL) NotificationServer() {
    for msg := range r.notifications {
        r.rl.Write([]byte("[!] " + msg + "\n"))
        r.rl.Refresh()
    }
}

func (r *REPL) registerHandlers() {
	r.handlers["help"] = func(_ *State, _ Command) error {
		printHelp()
		return nil
	}

	r.handlers["exit"] = func(_ *State, _ Command) error {
		return io.EOF
	}

	r.handlers["connect"] = func(s *State, cmd Command) error {
		if len(cmd.Args) < 1 {
			return errors.New("uso: connect <uri>")
		}
		if s.IsConnected() {
			return errors.New("ya hay una conexion activa")
		}
		if _, err := url.ParseRequestURI(cmd.Args[0]); err != nil {
			return errors.New("uri invalida")
		}
		conn, _, err := websocket.DefaultDialer.Dial(cmd.Args[0], nil)
		if err != nil {
			return err
		}
		s.SetConnection(conn)
		done := make(chan struct{})
		s.SetListenerDone(done)
		r.listenServer()
		r.prompt_connection = "\033[32m[Conectado]\033[0m"
		return nil
	}

	r.handlers["disconnect"] = func(s *State, _ Command) error {
		if !s.IsConnected() {
			return errors.New("no hay conexion activa")
		}
		if s.ListenerDone() != nil {
			close(s.ListenerDone())
			s.SetListenerDone(nil)
		}
		_ = s.conn.Close()
		s.SetConnection(nil)
		r.prompt_connection = "[Local]"
		return nil
	}

	r.handlers["insert"] = func(s *State, cmd Command) error {
		if len(cmd.Args) < 2 {
			return errors.New("uso: insert <char> <pos>")
		}
		if !s.IsConnected() {
			return errors.New("no hay conexion activa")
		}
		if utf8.RuneCountInString(cmd.Args[0]) != 1 {
			return errors.New("insert requiere un solo caracter")
		}
		pos, err := strconv.Atoi(cmd.Args[1])
		if err != nil || pos < 0 {
			return errors.New("posicion invalida")
		}
		op := operation.Operation{
			Op:       operation.Insert,
			Position: pos,
			Content:  cmd.Args[0],
		}
		return r.enqueueAndSend(op)
	}

	r.handlers["delete"] = func(s *State, cmd Command) error {
		if len(cmd.Args) < 1 {
			return errors.New("uso: delete <pos>")
		}
		if !s.IsConnected() {
			return errors.New("no hay conexion activa")
		}
		pos, err := strconv.Atoi(cmd.Args[0])
		if err != nil || pos < 0 {
			return errors.New("posicion invalida")
		}
		op := operation.Operation{
			Op:       operation.Delete,
			Position: pos,
		}
		return r.enqueueAndSend(op)
	}

	r.handlers["latency"] = func(s *State, cmd Command) error {
        if len(cmd.Args) < 1 {
            return errors.New("uso: latency <ms>")
        }
        ms, err := strconv.Atoi(cmd.Args[0])
        if err != nil || ms < 0 {
            return errors.New("latencia invalida")
        }
        s.SetLatency(ms)
        return nil
    }

	r.handlers["show"] = func(_ *State, _ Command) error {
		fmt.Println("show")
		return nil
	}
}


func newCompleter() *readline.PrefixCompleter {
	return readline.NewPrefixCompleter(
		readline.PcItem("connect"),
		readline.PcItem("disconnect"),
		readline.PcItem("latency"),
		readline.PcItem("insert"),
		readline.PcItem("delete"),
		readline.PcItem("show"),
		readline.PcItem("help"),
		readline.PcItem("exit"),
	)
}

func (r *REPL) getLine() (string, error) {

	fmt.Println("") // Add new line after each command for better readability

	promptConnection := "[Local]"
	if r.state.IsConnected() {
	    promptConnection = "\033[32m[Conectado]\033[0m"
	}

	promptLatency := ""
    if r.state.Latency() > 0 {
        promptLatency = fmt.Sprintf("\033[33m[%dms]\033[0m", r.state.Latency())
    }

	r.rl.SetPrompt(promptConnection + promptLatency + r.prompt_symbol)

	line, err := r.rl.Readline()
	if err != nil {
		if errors.Is(err, readline.ErrInterrupt) {
			return "", io.EOF
		}
		if errors.Is(err, io.EOF) {
			return "", io.EOF
		}
		return "", err
	}

	line = strings.TrimSpace(line)

	return line, nil
}


func printHelp() {
    fmt.Fprintln(os.Stdout, `Comandos disponibles:
    connect <uri>           Connect to the server at the specified
                            URI (e.g. ws://localhost:8080/ws)
    disconnect              Disconnect from the server
    insert <char> <pos>     Send an insert operation
    delete <pos>            Send a delete operation
    latency <ms>            Set artificial latency in milliseconds
                            for outgoing messages. Use 0 to disable.
    show
    help                    Show this help message
    exit                    Quit the REPL

Keys:
    Ctrl+C:                 Quit the REPL
    TAB:                    Auto-complete commands
    UP/DOWN:                Navigate command history`)
}

func (r *REPL) enqueueAndSend(op operation.Operation) error {
    s := r.state
	s.Mutex().Lock()
	op.Revision = s.LastSyncRevision

	if len(s.SentChanges) == 0 {
		s.SentChanges = append(s.SentChanges, op)
		s.Mutex().Unlock()
		msg, err := message.NewMessage(message.MessageTypeOperation, op)
		if err != nil {
			return err
		}
		if latency := r.state.Latency(); latency > 0 {
            time.Sleep(time.Duration(latency) * time.Millisecond)
        }
		return s.conn.WriteJSON(msg)
	}

	s.PendingChanges = append(s.PendingChanges, op)
	s.Mutex().Unlock()
	return nil
}

func (r *REPL) listenServer() {
    go func(){
        state := r.state
    	for {
    		select {
    		case <-state.ListenerDone():
    			return
    		default:
    		}

    		var msg message.Message
    		if err := state.conn.ReadJSON(&msg); err != nil {
                r.notifications <- "Conexión cerrada por el servidor."
    			// fmt.Println("\nConexión cerrada por el servidor.")
    			return
    		}

    		switch msg.Type {
    		case message.MessageTypeOperation:
    			op, err := msg.DecodeOperation()
    			if err != nil {
                    r.notifications <- "Operacion invalida recibida del servidor."
    				// fmt.Print("\n[SERVER]: Operacion invalida\n")
    				continue
    			}

    			state.Mutex().Lock()
    			isValid := true
    			for i := range state.SentChanges {
    				localOp := state.SentChanges[i]
    				state.SentChanges[i], _ = operation.Transform(localOp, op)
    				op, isValid = operation.Transform(op, localOp)
    				if !isValid {
    					break
    				}
    			}

    			if isValid {
    				for i := range state.PendingChanges {
    					localOp := state.PendingChanges[i]
    					state.PendingChanges[i], _ = operation.Transform(localOp, op)
    					op, isValid = operation.Transform(op, localOp)
    					if !isValid {
    						break
    					}
    				}
    			}

    			state.LastSyncRevision++
    			state.Mutex().Unlock()

    			if isValid {
                    r.notifications <- fmt.Sprintf("Operación %s transformada a posición %d con contenido '%s'", op.Op, op.Position, op.Content)
    				// fmt.Printf("\n[SERVER]: Operación %s transformada a posición %d con contenido '%s'\n", op.Op, op.Position, op.Content)
    			}
    		case message.MessageTypeAck:
    			ackMsg, err := msg.DecodeAck()
    			if err != nil {
                    r.notifications <- "Ack invalido recibido del servidor."
    				// fmt.Print("\n[SERVER]: Ack invalido\n")
    				continue
    			}

    			state.Mutex().Lock()
    			state.LastSyncRevision = ackMsg.Revision

    			if len(state.SentChanges) > 0 {
    				state.SentChanges = state.SentChanges[1:]
    			}

    			if len(state.PendingChanges) > 0 {
    				nextOp := state.PendingChanges[0]
    				state.PendingChanges = state.PendingChanges[1:]
    				nextOp.Revision = state.LastSyncRevision
    				state.SentChanges = append(state.SentChanges, nextOp)
    				outMsg, _ := message.NewMessage(message.MessageTypeOperation, nextOp)
    				state.conn.WriteJSON(outMsg)
    			}
    			state.Mutex().Unlock()
                r.notifications <- fmt.Sprintf("Ack revision %d recibido del servidor.", ackMsg.Revision)
    			// fmt.Printf("\n[SERVER]: Ack revision %d\n", ackMsg.Revision)
    		default:
                r.notifications <- "Tipo de mensaje desconocido recibido del servidor."
    			fmt.Print("\n[SERVER]: Tipo de mensaje desconocido\n")
    		}
    	}
    }()
}
