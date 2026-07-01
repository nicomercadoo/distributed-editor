package cmd

import (
	"time"

	"github.com/spf13/cobra"
)

var (
	serverURL string
	clientID  string
	timeout   time.Duration
)

var rootCmd = &cobra.Command{
	Use:   "client",
	Short: "CLI del cliente colaborativo",
	Long:  "Cliente para conectarse al servidor WebSocket y enviar/recibir operaciones de edición.",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", "ws://localhost:8080/ws", "URL WebSocket del servidor")
	rootCmd.PersistentFlags().StringVar(&clientID, "client-id", "", "Identificador del cliente")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 10*time.Second, "Timeout de operaciones de red")

	rootCmd.AddCommand(newStartCmd())
	rootCmd.AddCommand(newWatchCmd())
	rootCmd.AddCommand(newVersionCmd())
}

func Execute() error {
	return rootCmd.Execute()
}
