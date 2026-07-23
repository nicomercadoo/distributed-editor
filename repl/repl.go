package repl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"proyecto-final-redes/internal/client"
	"proyecto-final-redes/internal/protocol/message"

	"github.com/chzyer/readline"
)

type REPL struct {
	rl            *readline.Instance
	onErrorState  bool
	LocalOps      chan<- message.Operation
	Ctrl          client.AppController
	Notifications <-chan string
}

func NewREPL(
	localOps chan<- message.Operation,
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
		onErrorState:  false,
		LocalOps:      localOps,
		Ctrl:          ctrl,
		Notifications: notifications,
	}, nil
}

func (r *REPL) Run(ctx context.Context) {
	defer r.rl.Close()
	printHelp()

	go r.notificationServer(ctx)

	r.onErrorState = false

	for {
		fmt.Println("") // Add new line after each command for better readability

		r.setPrompt()
		line, err := r.rl.Readline()
		if err != nil {
			if errors.Is(err, readline.ErrInterrupt) || errors.Is(err, io.EOF) {
				return
			}
			fmt.Println("\033[31mError:\033[0m", err)
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

		if err := r.handleCommand(cmd); err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			fmt.Println("\033[31mError:\033[0m", err)
			r.onErrorState = true
		} else {
			r.onErrorState = false
		}
	}
}

func (r *REPL) setPrompt() {
	promptConnectionStatus := "[Local]"
	if r.Ctrl.IsConnected() {
		promptConnectionStatus = "\033[32m[Conectado]\033[0m"
	}

	promptLatency := ""
	latency := r.Ctrl.GetLatency()
	if latency > 0 {
		promptLatency = fmt.Sprintf("\033[33m[%s]\033[0m", latency)
	}

	promptSymbol := "> "
	if r.onErrorState {
		promptSymbol = "!> "
	}

	r.rl.SetPrompt(promptConnectionStatus + promptLatency + promptSymbol)
}

func (r *REPL) handleCommand(cmd Command) error {
	handler, ok := Handlers[cmd.Name]
	if !ok {
		return errors.New("Comando desconocido. Usa 'help'.")
	}
	return handler(cmd, r)
}

func (r *REPL) notificationServer(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-r.Notifications:
			// \r       : Mueve el cursor al inicio de la línea física.
			// \033[2K  : Limpia toda la línea (borra el prompt residual).
			// [!] msg  : Tu mensaje.
			// \n       : Salto de línea.
			// r.rl.Stdout() imprimirá esto y luego redibujará el prompt limpio debajo.
			r.setPrompt()
			fmt.Fprintf(r.rl.Stdout(), "\r\033[2K[!] %s\n", msg)
		}
	}
}

func printHelp() {
	fmt.Println(`Comandos disponibles:
    connect <uri>               Conecta al servidor (ej. ws://localhost:8080/ws)
    disconnect                  Desconecta del servidor
    insert <char> <pos>         Envia una operacion de insercion
    delete <pos>                Envia una operacion de borrado
    latency <duracion><unit>    Establece latencia artificial (unidades: ns, us, ms, s, m, h)
    status                      Muestra el estado de la conexion y latencia
    doc                         Muestra el documento y estado de la sincronizacion
    help                        Muestra esta ayuda
    exit                        Sale del REPL`)
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
