package cmd

import (
	// "proyecto-final-redes/internal/client"
	"proyecto-final-redes/internal/client/repl"

	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Inicia una sesión interactiva",
		Long:  "Abre una sesión interactiva contra el servidor WebSocket configurado.",
		Run: func(cmd *cobra.Command, args []string) {
			// client.StartSession(serverURL)
			state := repl.NewState()
			repl, err := repl.NewREPL(state)
			if err != nil {
                panic(err)
            }
            repl.Run()
		},
	}
}
