package cmd

import (
	"fmt"
	"net/http"
	"log"

	"proyecto-final-redes/internal/server"

	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	var clear bool

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Inicia el servidor",
		Long:  "Arranca el servidor WebSocket de edicion colaborativa.",
		Run: func(cmd *cobra.Command, args []string) {
			hub := server.NewHub(clear)
			go hub.Run() // Iniciamos el hub en segundo plano
			log.Print("Hub iniciado. Esperando conexiones...\n")

			http.HandleFunc(wsPath, server.HandleConnection(hub))

			addr := fmt.Sprintf("%s:%d", host, port)
			log.Printf("Servidor de documento único escuchando en %s%s...\n", addr, wsPath)

			if err := http.ListenAndServe(addr, nil); err != nil {
				log.Fatalf("Error iniciando servidor: %v\n", err)
			}
		},
	}

	cmd.Flags().BoolVarP(&clear, "clear", "c", false, "Limpia el historial de operaciones y arranca un documento nuevo")
	return cmd
}
