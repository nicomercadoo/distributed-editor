package repl

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chzyer/readline"
)

type Handler func(*State, Command) error
type propmtStatus struct {
    file string
    connected bool
}

type REPL struct {
	rl              *readline.Instance
	state           *State
	handlers        map[string]Handler
	propmtStatus    propmtStatus
	Notity          chan string
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
				prompt = "! > "
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
		fmt.Println("connect", cmd.Args[0])
		s.SetConnection(nil);
		return nil
	}

	r.handlers["disconnect"] = func(s *State, _ Command) error {
		fmt.Println("disconnect")
		s.SetConnection(nil);
		return nil
	}

	r.handlers["send"] = func(_ *State, cmd Command) error {
		if len(cmd.Args) < 1 {
			return errors.New("uso: send <op>")
		}
		fmt.Println("send", strings.Join(cmd.Args, " "))
		return nil
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
		readline.PcItem("send",
			readline.PcItem("insert"),
			readline.PcItem("delete"),
		),
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
			return "", nil
		}
		if errors.Is(err, io.EOF) {
			return "", nil
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
	fmt.Fprintln(os.Stdout, "    send <op>")
	fmt.Fprintln(os.Stdout, "    show")
	fmt.Fprintln(os.Stdout, "    help")
	fmt.Fprintln(os.Stdout, "    exit")
}
