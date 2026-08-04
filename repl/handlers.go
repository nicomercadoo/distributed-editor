package repl

import (
	"errors"
	"fmt"
	"io"
	"proyecto-final-redes/internal/protocol/message"
	"strconv"
	"unicode/utf8"
)

type Handler func(Command, *REPL) error

var Handlers = map[string]Handler{
	"help": func(cmd Command, r *REPL) error {
		printHelp()
		return nil
	},

	"exit": func(cmd Command, r *REPL) error {
		return io.EOF
	},

	"connect": func(cmd Command, r *REPL) error {
		if len(cmd.Args) < 1 {
			return errors.New("Uso: connect <uri>")
		}
		if err := r.Ctrl.Connect(cmd.Args[0]); err != nil {
			return err
		}
		return nil
	},

	"disconnect": func(cmd Command, r *REPL) error {
		if err := r.Ctrl.Disconnect(); err != nil {
			return err
		}
		return nil
	},

	"status": func(cmd Command, r *REPL) error {
		connected := "No conectado"
		if r.Ctrl.IsConnected() {
			connected = "Conectado"
		}
		fmt.Printf("Estado: %s\n", connected)
		return nil
	},

	"doc": func(cmd Command, r *REPL) error {
		doc := r.Ctrl.GetDocument()
		rev := r.Ctrl.GetRevision()
		pending := r.Ctrl.GetPendingCount()
		inflight := r.Ctrl.HasInFlight()
		fmt.Printf("Documento: \"%q\"\nRevisión: %d\nOperaciones pendientes: %d\nEn vuelo: %v\n", doc, rev, pending, inflight)
		return nil
	},

	"insert": func(cmd Command, r *REPL) error {
		if len(cmd.Args) < 2 {
			return errors.New("Uso: insert <char> <pos>")
		}
		if utf8.RuneCountInString(cmd.Args[0]) != 1 {
			return errors.New("insert requiere un solo caracter.")
		}
		pos, err := strconv.Atoi(cmd.Args[1])
		if err != nil || pos < 0 {
			return errors.New("Posición invalida.")
		}
		op := message.Operation{
			Op:       message.Insert,
			Position: pos,
			Content:  cmd.Args[0],
		}
		r.LocalOps <- op
		return nil
	},

	"delete": func(cmd Command, r *REPL) error {
		if len(cmd.Args) < 1 {
			return errors.New("Uso: delete <pos>")
		}
		pos, err := strconv.Atoi(cmd.Args[0])
		if err != nil || pos < 0 {
			return errors.New("Posición invalida.")
		}
		op := message.Operation{
			Op:       message.Delete,
			Position: pos,
		}
		r.LocalOps <- op
		return nil
	},
}
