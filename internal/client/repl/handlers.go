package repl

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"
	"unicode/utf8"

	"proyecto-final-redes/internal/protocol/operation"
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
			return errors.New("uso: connect <uri>")
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

	"latency": func(cmd Command, r *REPL) error {
		if len(cmd.Args) < 1 {
			return errors.New("uso: latency <duracion>")
		}
		d, err := time.ParseDuration(cmd.Args[0])
		if err != nil || d < 0 {
			return errors.New("latencia invalida")
		}
		r.Ctrl.SetLatency(d)
		fmt.Printf("Latencia configurada a %s\n", d)
		return nil
	},

	"status": func(cmd Command, r *REPL) error {
		connected := "No conectado"
		if r.Ctrl.IsConnected() {
			connected = "Conectado"
		}
		lat := r.Ctrl.GetLatency()
		fmt.Printf("Estado: %s\nLatencia: %s\n", connected, lat)
		return nil
	},

	"doc": func(cmd Command, r *REPL) error {
		doc := r.Ctrl.GetDocument()
		rev := r.Ctrl.GetRevision()
		pending := r.Ctrl.GetPendingCount()
		inflight := r.Ctrl.HasInFlight()
		fmt.Printf("Documento: %q\nRevisión: %d\nOperaciones pendientes: %d\nEn vuelo: %v\n", doc, rev, pending, inflight)
		return nil
	},

	"insert": func(cmd Command, r *REPL) error {
		if len(cmd.Args) < 2 {
			return errors.New("uso: insert <char> <pos>")
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
		r.LocalOps <- op
		return nil
	},

	"delete": func(cmd Command, r *REPL) error {
		if len(cmd.Args) < 1 {
			return errors.New("uso: delete <pos>")
		}
		pos, err := strconv.Atoi(cmd.Args[0])
		if err != nil || pos < 0 {
			return errors.New("posicion invalida")
		}
		op := operation.Operation{
			Op:       operation.Delete,
			Position: pos,
		}
		r.LocalOps <- op
		return nil
	},
}
