package main

import (
	"os"

	"proyecto-final-redes/cli/client/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
