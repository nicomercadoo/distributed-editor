package cmd

import (
	"proyecto-final-redes/internal/client"

	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Inicia una sesión interactiva",
		Long:  "Abre una sesión interactiva contra el servidor WebSocket configurado.",
		Run: func(cmd *cobra.Command, args []string) {
			client.StartSession(serverURL)
		},
	}
}
