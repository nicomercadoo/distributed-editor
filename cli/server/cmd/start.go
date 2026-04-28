package cmd

import (
	"fmt"
	"net/http"
	"os"

	"proyecto-final-redes/internal/server"

	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Inicia el servidor",
		Long:  "Arranca el servidor WebSocket de edicion colaborativa.",
		Run: func(cmd *cobra.Command, args []string) {
			go server.GlobalHub.Run()

			http.HandleFunc(wsPath, server.HandleConnection)

			addr := fmt.Sprintf("%s:%d", host, port)
			fmt.Printf("Servidor de documento único escuchando en %s%s...\n", addr, wsPath)
			if err := http.ListenAndServe(addr, nil); err != nil {
				fmt.Fprintf(os.Stderr, "Error iniciando servidor: %v\n", err)
				os.Exit(1)
			}
		},
	}
}
