package repl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/chzyer/readline"
	"proyecto-final-redes/internal/client"
	"proyecto-final-redes/internal/protocol/operation"
)

type REPL struct {
	rl            *readline.Instance
	LocalOps      chan<- operation.Operation
	Ctrl          client.AppController
	Notifications <-chan string
}

func NewREPL(
	localOps chan<- operation.Operation,
	ctrl client.AppController,
	notifications <-chan string,
) (*REPL, error) {
	config := &readline.Config{
		Prompt:       "> ",
		HistoryFile:  ".client_history",
		AutoComplete: newCompleter(),
	}

	rl, err := readline.NewEx(config)
	if err != nil {
		return nil, err
	}

	return &REPL{
		rl:            rl,
		LocalOps:      localOps,
		Ctrl:          ctrl,
		Notifications: notifications,
	}, nil
}

func (r *REPL) Run(ctx context.Context) {
	defer r.rl.Close()
	printHelp()

	go r.notificationServer(ctx)

	for {
		line, err := r.rl.Readline()
		if err != nil {
			if errors.Is(err, readline.ErrInterrupt) || errors.Is(err, io.EOF) {
				return
			}
			fmt.Println("error:", err)
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		cmd, err := Parse(line)
		if err != nil {
			continue
		}

		if err := r.handleCommand(cmd.Name, cmd.Args); err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			fmt.Println("error:", err)
			r.rl.SetPrompt("!> ")
		} else {
			r.rl.SetPrompt("> ")
		}
	}
}

func (r *REPL) handleCommand(cmd string, args []string) error {
	switch cmd {
	case "help":
		printHelp()
	case "exit":
		return io.EOF
	case "connect":
		if len(args) < 1 {
			return errors.New("uso: connect <uri>")
		}
		if err := r.Ctrl.Connect(args[0]); err != nil {
			return err
		}
	case "disconnect":
		if err := r.Ctrl.Disconnect(); err != nil {
			return err
		}
	case "latency":
		if len(args) < 1 {
			return errors.New("uso: latency <ms>")
		}
		ms, err := strconv.Atoi(args[0])
		if err != nil || ms < 0 {
			return errors.New("latencia invalida")
		}
		r.Ctrl.SetLatency(time.Duration(ms) * time.Millisecond)
		fmt.Printf("Latencia configurada a %d ms\n", ms)
	case "status":
		connected := "No conectado"
		if r.Ctrl.IsConnected() {
			connected = "Conectado"
		}
		lat := r.Ctrl.GetLatency()
		fmt.Printf("Estado: %s\nLatencia: %s\n", connected, lat)
	case "doc":
		doc := r.Ctrl.GetDocument()
		rev := r.Ctrl.GetRevision()
		pending := r.Ctrl.GetPendingCount()
		inflight := r.Ctrl.HasInFlight()
		fmt.Printf("Documento: %q\nRevisión: %d\nOperaciones pendientes: %d\nEn vuelo: %v\n", doc, rev, pending, inflight)
	case "insert":
		if len(args) < 2 {
			return errors.New("uso: insert <char> <pos>")
		}
		if utf8.RuneCountInString(args[0]) != 1 {
			return errors.New("insert requiere un solo caracter")
		}
		pos, err := strconv.Atoi(args[1])
		if err != nil || pos < 0 {
			return errors.New("posicion invalida")
		}
		op := operation.Operation{
			Op:       operation.Insert,
			Position: pos,
			Content:  args[0],
		}
		r.LocalOps <- op
	case "delete":
		if len(args) < 1 {
			return errors.New("uso: delete <pos>")
		}
		pos, err := strconv.Atoi(args[0])
		if err != nil || pos < 0 {
			return errors.New("posicion invalida")
		}
		op := operation.Operation{
			Op:       operation.Delete,
			Position: pos,
		}
		r.LocalOps <- op
	default:
		return errors.New("comando desconocido. usa 'help'")
	}
	return nil
}

func (r *REPL) notificationServer(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-r.Notifications:
			r.rl.Write([]byte("[!] " + msg + "\n"))
			r.rl.Refresh()
		}
	}
}

func printHelp() {
	fmt.Println(`Comandos disponibles:
    connect <uri>           Conecta al servidor (ej. ws://localhost:8080/ws)
    disconnect              Desconecta del servidor
    insert <char> <pos>     Envia una operacion de insercion
    delete <pos>            Envia una operacion de borrado
    latency <ms>            Establece latencia artificial
    status                  Muestra el estado de la conexion y latencia
    doc                     Muestra el documento y estado de la sincronizacion
    help                    Muestra esta ayuda
    exit                    Sale del REPL
    `)
}

func newCompleter() *readline.PrefixCompleter {
	return readline.NewPrefixCompleter(
		readline.PcItem("connect"),
		readline.PcItem("disconnect"),
		readline.PcItem("latency"),
		readline.PcItem("status"),
		readline.PcItem("doc"),
		readline.PcItem("insert"),
		readline.PcItem("delete"),
		readline.PcItem("help"),
		readline.PcItem("exit"),
	)
}
