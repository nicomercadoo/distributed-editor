package repl

import (
    "errors"
    "fmt"
    "io"
    "os"
    "net/url"
    "strconv"
    "unicode/utf8"

    "github.com/chzyer/readline"
    "github.com/gorilla/websocket"
    "proyecto-final-redes/internal/protocol/operation"

)

type Command struct {
	Name string
	Args []string
	Raw  string
}

type Handler func(*State, Command) error

var Handlers = map[string]Handler{
	"help": func(_ *State, _ Command) error {
		printHelp()
		return nil
	},

	"exit": func(_ *State, _ Command) error {
		return io.EOF
	},

	"connect": func(s *State, cmd Command) error {
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
		s.promptSymbol = "\033[32m[Conectado]\033[0m"

		go listenServer(s)

		return nil
	},

	"disconnect": func(s *State, _ Command) error {
		if !s.IsConnected() {
			return errors.New("no hay conexion activa")
		}
		if s.ListenerDone() != nil {
			close(s.ListenerDone())
			s.SetListenerDone(nil)
		}
		_ = s.conn.Close()
		s.SetConnection(nil)
		return nil
	},

	"insert": func(s *State, cmd Command) error {
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
	},

	"delete": func(s *State, cmd Command) error {
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
	},

	"latency": func(s *State, cmd Command) error {
		if len(cmd.Args) < 1 {
			return errors.New("uso: latency <ms>")
		}
		ms, err := strconv.Atoi(cmd.Args[0])
		if err != nil || ms < 0 {
			return errors.New("latencia invalida")
		}
		s.SetLatency(ms)
		return nil
	},

	"show": func(_ *State, _ Command) error {
		fmt.Println("show")
		return nil
	},
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
