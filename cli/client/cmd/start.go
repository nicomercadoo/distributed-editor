package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"proyecto-final-redes/internal/client"
	"proyecto-final-redes/internal/client/repl"
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

			connectCmd := make(chan string, 1)
			disconnectCmd := make(chan struct{}, 1)
			latencyCmd := make(chan int, 1)
			notifications := make(chan string, 100)

			syncEngine := &client.SyncEngine{
				LocalOps:      localOps,
				RemoteOps:     remoteOps,
				Acks:          acks,
				Outbound:      outbound,
				Notifications: notifications,
			}

			connManager := &client.ConnectionManager{
				ConnectCmd:    connectCmd,
				DisconnectCmd: disconnectCmd,
				LatencyCmd:    latencyCmd,
				RemoteOps:     remoteOps,
				Acks:          acks,
				Outbound:      outbound,
				Notifications: notifications,
			}

			r, err := repl.NewREPL(
				localOps,
				connectCmd,
				disconnectCmd,
				latencyCmd,
				notifications,
			)
			if err != nil {
				panic(err)
			}

			go syncEngine.Run(ctx)
			go connManager.Run(ctx)

			// Graceful shutdown with OS signals
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-sigCh
				cancel()
				r.Run(ctx) // well, r.Run is blocking, so we cancel and the program will exit
			}()

			r.Run(ctx)
		},
	}
}
