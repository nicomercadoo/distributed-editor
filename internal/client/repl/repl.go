package repl

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/chzyer/readline"

	"proyecto-final-redes/internal/protocol/message"
	"proyecto-final-redes/internal/protocol/operation"
)


type REPL struct {
	rl       *readline.Instance
	state    *State
	handlers map[string]Handler
	mu sync.RWMutex
}

func NewREPL(state *State) (*REPL, error) {
	config := &readline.Config{
		Prompt:       state.promptSymbol,
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
		mu: sync.RWMutex{},
	}

	repl.registerHandlers()
	return repl, nil
}

// func Run() error {
// 	repl, err := NewREPL(NewState())
// 	if err != nil {
// 		return err
// 	}
// 	defer repl.Close()
// 	return repl.Run()
// }

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

		if handler, ok := r.handlers[cmd.Name]; ok { // If handler exists for the command
			if err := handler(r.state, cmd); err != nil { // Execute the handler and check for errors
				if errors.Is(err, io.EOF) {
					return nil
				}
				fmt.Println("error:", err)
				r.state.promptSymbol = "!> "
			} else {
				r.state.promptSymbol = "> "
			}
			continue
		}

		fmt.Println("comando desconocido. usa 'help'")
	}
}

func (r *REPL) NotificationServer() {
    for msg := range r.state.notifications {
        r.rl.Write([]byte("[!] " + msg + "\n"))
        r.rl.Refresh()
    }
}

func (r *REPL) registerHandlers() {
    r.handlers = Handlers
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

	r.rl.SetPrompt(promptConnection + promptLatency + r.state.promptSymbol)

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

// Lo de abajo hay que moverlo a otro lado, son independientes del repl

func enqueueAndSend(s *State, op operation.Operation) error {
	s.Mutex().Lock()
	op.Revision = s.lastSyncRevision

	if s.IsConnected() && s.SentChange == (SentChange{}) {
		s.SentChange = SentChange{Op: op, Received: false}
		s.Mutex().Unlock()
		msg, err := message.NewMessage(message.MessageTypeOperation, op)
		if err != nil {
			return err
		}
		if latency := s.Latency(); latency > 0 {
            time.Sleep(time.Duration(latency) * time.Millisecond)
        }
		return s.conn.WriteJSON(msg)
	}

	s.PendingChanges = append(s.PendingChanges, op)
	s.Mutex().Unlock()
	return nil
}

// Listener refactor proposal
func listener(state *State, handlers map[message.MessageType] func(*State)) {
    for {
        select {
        case <-state.ListenerDone():
            return
        default:
        }

        var msg message.Message
        if err := state.conn.ReadJSON(&msg); err != nil {
            state.notifications <- "Conexión cerrada por el servidor."
            return
        }

        if handler, ok := handlers[msg.Type]; ok {
            handler(state)
        } else {
            state.notifications <- "Tipo de mensaje desconocido recibido del servidor."
        }
    }
}

var ServerMessageHandlers = map[message.MessageType]func(*State){
    message.MessageTypeOperation: func(state *State) {
        // Aquí va la lógica para manejar mensajes de tipo Operation
    },
    message.MessageTypeAck: func(state *State) {
        // Aquí va la lógica para manejar mensajes de tipo Ack
    },
}

func listenServer(state *State) {
   	for {
  		select {
  		case <-state.ListenerDone():
 			return
  		default:
  		}

  		var msg message.Message
  		if err := state.conn.ReadJSON(&msg); err != nil {
            state.notifications <- "Conexión cerrada por el servidor."
 			// fmt.Println("\nConexión cerrada por el servidor.")
 			return
  		}

  		switch msg.Type {
  		case message.MessageTypeOperation:
 			op, err := msg.DecodeOperation()
 			if err != nil {
                state.notifications <- "Operacion invalida recibida del servidor."
				// fmt.Print("\n[SERVER]: Operacion invalida\n")
				continue
 			}

 			state.Mutex().Lock()
 			isValid := true
 			// for i := range state.SentChanges {
            if sentOp := state.SentChange; sentOp != (SentChange{}) {
				localOp := state.SentChange.Op
				state.SentChange.Op, _ = operation.Transform(localOp, op)
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

 			state.lastSyncRevision++
 			state.Mutex().Unlock()

 			if isValid {
                state.notifications <- fmt.Sprintf("Operación %s transformada a posición %d con contenido '%s'", op.Op, op.Position, op.Content)
				// fmt.Printf("\n[SERVER]: Operación %s transformada a posición %d con contenido '%s'\n", op.Op, op.Position, op.Content)
 			}
  		case message.MessageTypeAck:
 			ackMsg, err := msg.DecodeAck()
 			if err != nil {
                state.notifications <- "Ack invalido recibido del servidor."
				// fmt.Print("\n[SERVER]: Ack invalido\n")
				continue
 			}

 			state.Mutex().Lock()
 			state.lastSyncRevision = ackMsg.Revision

 			// if len(state.SentChanges) > 0 {
 			if state.SentChange != (SentChange{}) {
                state.CommitedChanges = append(state.CommitedChanges, state.SentChange.Op)
				state.SentChange = SentChange{}
 			}

 			if len(state.PendingChanges) > 0 {
				nextOp := state.PendingChanges[0]
				state.PendingChanges = state.PendingChanges[1:]
				nextOp.Revision = state.lastSyncRevision
				// state.SentChanges = append(state.SentChanges, nextOp)
				state.SentChange = SentChange{Op: nextOp, Received: false}
				outMsg, _ := message.NewMessage(message.MessageTypeOperation, nextOp)
				state.conn.WriteJSON(outMsg)
 			}
 			state.Mutex().Unlock()
            state.notifications <- fmt.Sprintf("Ack revision %d recibido del servidor.", ackMsg.Revision)
 			// fmt.Printf("\n[SERVER]: Ack revision %d\n", ackMsg.Revision)
  		default:
            state.notifications <- "Tipo de mensaje desconocido recibido del servidor."
 			fmt.Print("\n[SERVER]: Tipo de mensaje desconocido\n")
  		}
   	}
}

func sender(state *State) {
    for {
        select {
        case <-state.SenderDone():
            return
        default:
        }



        // Generated by Copilot
        state.Mutex().Lock()
        if state.SentChange != (SentChange{}) && !state.SentChange.Received {
            msg, err := message.NewMessage(message.MessageTypeOperation, state.SentChange.Op)
            if err == nil {
                if latency := state.Latency(); latency > 0 {
                    time.Sleep(time.Duration(latency) * time.Millisecond)
                }
                state.conn.WriteJSON(msg)
            }
        }
        state.Mutex().Unlock()

        time.Sleep(100 * time.Millisecond) // Sleep to prevent busy waiting
    }
}
