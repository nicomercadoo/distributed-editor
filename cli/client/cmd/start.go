package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"proyecto-final-redes/repl"
	"proyecto-final-redes/internal/client"
	"proyecto-final-redes/internal/protocol/operation"

	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Inicia una sesión interactiva",
		Long:  "Abre una sesión interactiva contra el servidor WebSocket configurado.",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			localOps := make(chan operation.Operation, 100)
			remoteOps := make(chan operation.Operation, 100)
			acks := make(chan int, 100)
			outbound := make(chan operation.Operation, 100)

			notifications := make(chan string, 100)

			syncEngine := client.NewSyncEngine(localOps, remoteOps, acks, outbound, notifications)
			connManager := client.NewConnectionManager(ctx, remoteOps, acks, outbound, notifications)
			appClient := client.NewAppClient(syncEngine, connManager)

			r, err := repl.NewREPL(
				localOps,
				appClient,
				notifications,
			)
			if err != nil {
				panic(err)
			}

			go syncEngine.Run(ctx)

			// Graceful shutdown with OS signals
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-sigCh
				cancel()
			}()

			r.Run(ctx)
		},
	}
}
