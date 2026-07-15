package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"proyecto-final-redes/internal/client"
	"proyecto-final-redes/internal/client/syncengine"
	"proyecto-final-redes/internal/protocol/message"
	"proyecto-final-redes/internal/protocol/operation"
	"proyecto-final-redes/web_ui"

	"github.com/spf13/cobra"
)

func newStartWebCmd() *cobra.Command {
	var webPort int

	cmd := &cobra.Command{
		Use:   "start-web",
		Short: "Inicia la sesión interactiva con Web UI",
		Long:  "Abre una sesión (Local Bridge) contra el servidor WebSocket sirviendo la interfaz de usuario web localmente.",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			localOps := make(chan operation.Operation, 100)
			incoming := make(chan message.ServerEvent, 100)
			outbound := make(chan operation.Operation, 100)

			notifications := make(chan string, 100)

			syncEngine := syncengine.NewSyncEngine(localOps, incoming, outbound, notifications)
			connManager := client.NewConnectionManager(ctx, incoming, outbound, notifications, syncEngine.GetRevision)
			appClient := client.NewAppClient(syncEngine, connManager)

			server := webui.NewServer(localOps, appClient, webPort)

			go syncEngine.Run(ctx)

			// Graceful shutdown with OS signals
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-sigCh
				cancel()
			}()

			server.Run(ctx)
		},
	}

	cmd.Flags().IntVarP(&webPort, "port", "p", 3000, "Puerto local para servir la Web UI")
	return cmd
}
