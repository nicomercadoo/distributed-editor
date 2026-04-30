package cmd

import (
	// "fmt"
	"log"
	"github.com/spf13/cobra"
)

func newWatchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "watch",
		Short: "Escucha eventos remotos",
		Long:  "Abre una conexion para escuchar y mostrar operaciones del servidor en tiempo real.",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Printf("watch server=%s client-id=%s timeout=%s\n", serverURL, clientID, timeout)
			log.Fatal("Esto es fatal")
			return nil
		},
	}
}
