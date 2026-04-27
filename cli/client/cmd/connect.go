package cmd

import (
	"fmt"
	"proyecto-final-redes/internal/config"

	"github.com/spf13/cobra"
)

func newConnectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "connect",
		Short: "Conecta al servidor",
		Long:  "Establece una conexión WebSocket con el servidor de edición colaborativa.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("connect server=%s client-id=%s timeout=%s\n", serverURL, clientID, timeout)
			return nil
		},
	}
}
