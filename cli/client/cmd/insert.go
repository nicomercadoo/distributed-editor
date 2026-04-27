package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newInsertCmd() *cobra.Command {
	var ch string
	var pos int

	command := &cobra.Command{
		Use:   "insert",
		Short: "Envia una insercion",
		Long:  "Solicita insertar un caracter en una posicion del documento remoto.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("insert char=%s pos=%d server=%s client-id=%s\n", ch, pos, serverURL, clientID)
			return nil
		},
	}

	command.Flags().StringVar(&ch, "char", "", "Caracter a insertar")
	command.Flags().IntVar(&pos, "pos", 0, "Posicion de insercion")
	_ = command.MarkFlagRequired("char")

	return command
}
