package repl

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
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
}

func NewREPL(state *State) (*REPL, error) {
	config := &readline.Config{
		Prompt:       "> ",
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
	prompt := "> "
	for {
		line, err := r.getLine("[No File] [Local]", prompt)
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
			} else {
				prompt = "> "
			}
			continue
		}

		fmt.Println("comando desconocido. usa 'help'")
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
		go listenServer(s)
		fmt.Println("conectado")
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
		fmt.Println("desconectado")
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
		return enqueueAndSend(s, op)
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
		return enqueueAndSend(s, op)
	}

	r.handlers["show"] = func(_ *State, _ Command) error {
		fmt.Println("show")
		return nil
	}
}

// func (r *REPL) setPrompt(prompt string) {
// 	r.rl.SetPrompt(prompt)
// 	r.rl.Refresh()
// }

// func (r *REPL) setPromptForState() {
// 	if r.state.IsConnected() {
// 		r.setPrompt("OK > ")
// 		return
// 	}
// 	r.setPrompt(">>> ")
// }

func newCompleter() *readline.PrefixCompleter {
	return readline.NewPrefixCompleter(
		readline.PcItem("connect"),
		readline.PcItem("disconnect"),
		readline.PcItem("insert"),
		readline.PcItem("delete"),
		readline.PcItem("show"),
		readline.PcItem("help"),
		readline.PcItem("exit"),
	)
}

func (r *REPL) getLine(status string, prompt string) (string, error) {

	fmt.Printf("\n%s\n", status)
	r.rl.SetPrompt(prompt)

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

	// EL TRUCO TRANSIENT ✨
	// El cursor bajó una línea tras el 'Enter'.
	// Subimos 3 líneas (una por el '>', otra por el estado) y borramos.
	fmt.Print("\033[3A\033[J")

	// Reescribimos el comando en el historial para que quede limpio
	if line != "" {
		fmt.Printf("%s%s\n", prompt, line)
	}

	return line, nil
}

func printHelp() {
	fmt.Fprintln(os.Stdout, "Comandos disponibles:")
	fmt.Fprintln(os.Stdout, "    connect <uri>")
	fmt.Fprintln(os.Stdout, "    disconnect")
	fmt.Fprintln(os.Stdout, "    insert <char> <pos>")
	fmt.Fprintln(os.Stdout, "    delete <pos>")
	fmt.Fprintln(os.Stdout, "    show")
	fmt.Fprintln(os.Stdout, "    help")
	fmt.Fprintln(os.Stdout, "    exit")
}

func enqueueAndSend(s *State, op operation.Operation) error {
	s.Mutex().Lock()
	op.Revision = s.LastSyncRevision

	if len(s.SentChanges) == 0 {
		s.SentChanges = append(s.SentChanges, op)
		s.Mutex().Unlock()
		msg, err := message.NewMessage(message.MessageTypeOperation, op)
		if err != nil {
			return err
		}
		return s.conn.WriteJSON(msg)
	}

	s.PendingChanges = append(s.PendingChanges, op)
	s.Mutex().Unlock()
	return nil
}

func listenServer(s *State) {
	for {
		select {
		case <-s.ListenerDone():
			return
		default:
		}

		var msg message.Message
		if err := s.conn.ReadJSON(&msg); err != nil {
			fmt.Println("\nConexión cerrada por el servidor.")
			return
		}

		switch msg.Type {
		case message.MessageTypeOperation:
			op, err := msg.DecodeOperation()
			if err != nil {
				fmt.Print("\n[SERVER]: Operacion invalida\n")
				continue
			}

			s.Mutex().Lock()
			isValid := true
			for i := range s.SentChanges {
				localOp := s.SentChanges[i]
				s.SentChanges[i], _ = operation.Transform(localOp, op)
				op, isValid = operation.Transform(op, localOp)
				if !isValid {
					break
				}
			}

			if isValid {
				for i := range s.PendingChanges {
					localOp := s.PendingChanges[i]
					s.PendingChanges[i], _ = operation.Transform(localOp, op)
					op, isValid = operation.Transform(op, localOp)
					if !isValid {
						break
					}
				}
			}

			s.LastSyncRevision++
			s.Mutex().Unlock()

			if isValid {
				fmt.Printf("\n[SERVER]: Operación %s transformada a posición %d con contenido '%s'\n", op.Op, op.Position, op.Content)
			}
		case message.MessageTypeAck:
			ackMsg, err := msg.DecodeAck()
			if err != nil {
				fmt.Print("\n[SERVER]: Ack invalido\n")
				continue
			}

			s.Mutex().Lock()
			s.LastSyncRevision = ackMsg.Revision

			if len(s.SentChanges) > 0 {
				s.SentChanges = s.SentChanges[1:]
			}

			if len(s.PendingChanges) > 0 {
				nextOp := s.PendingChanges[0]
				s.PendingChanges = s.PendingChanges[1:]
				nextOp.Revision = s.LastSyncRevision
				s.SentChanges = append(s.SentChanges, nextOp)
				outMsg, _ := message.NewMessage(message.MessageTypeOperation, nextOp)
				s.conn.WriteJSON(outMsg)
			}
			s.Mutex().Unlock()
			fmt.Printf("\n[SERVER]: Ack revision %d\n", ackMsg.Revision)
		default:
			fmt.Print("\n[SERVER]: Tipo de mensaje desconocido\n")
		}
	}
}
