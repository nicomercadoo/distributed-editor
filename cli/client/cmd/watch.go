package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newWatchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "watch",
		Short: "Escucha eventos remotos",
		Long:  "Abre una conexion para escuchar y mostrar operaciones del servidor en tiempo real.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("watch server=%s client-id=%s timeout=%s\n", serverURL, clientID, timeout)
			return nil
		},
	}
}
