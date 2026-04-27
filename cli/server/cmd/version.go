package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const serverVersion = "0.1.0"

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Muestra version del servidor",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("server version %s\n", serverVersion)
			return nil
		},
	}
}
