package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Inicia el servidor",
		Long:  "Arranca el servidor WebSocket de edicion colaborativa.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("start host=%s port=%d ws-path=%s log-level=%s\n", host, port, wsPath, logLevel)
			return nil
		},
	}
}
