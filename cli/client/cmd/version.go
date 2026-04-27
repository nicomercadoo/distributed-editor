package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const clientVersion = "0.1.0"

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Muestra version del cliente",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("client version %s\n", clientVersion)
			return nil
		},
	}
}
