package cmd

import "github.com/spf13/cobra"

var (
	host     string
	port     int
	wsPath   string
	logLevel string
)

var rootCmd = &cobra.Command{
	Use:   "server",
	Short: "CLI del servidor colaborativo",
	Long:  "Servidor de edicion colaborativa sobre WebSocket.",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&host, "host", "localhost", "Host de escucha")
	rootCmd.PersistentFlags().IntVar(&port, "port", 8080, "Puerto de escucha")
	rootCmd.PersistentFlags().StringVar(&wsPath, "ws-path", "/ws", "Path del endpoint WebSocket")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Nivel de logging")

	rootCmd.AddCommand(newStartCmd())
	rootCmd.AddCommand(newVersionCmd())
}

func Execute() error {
	return rootCmd.Execute()
}
