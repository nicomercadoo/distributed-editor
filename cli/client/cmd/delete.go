package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
	var pos int

	command := &cobra.Command{
		Use:   "delete",
		Short: "Envia una eliminacion",
		Long:  "Solicita eliminar un caracter en una posicion del documento remoto.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("delete pos=%d server=%s client-id=%s\n", pos, serverURL, clientID)
			return nil
		},
	}

	command.Flags().IntVar(&pos, "pos", 0, "Posicion a eliminar")

	return command
}
