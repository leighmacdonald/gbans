package cmd

import (
	"context"
	"github.com/leighmacdonald/gbans/relay"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	serverName string
	relayAddr  string
	logPath    string
)

// relayCmd starts the log relay service
var relayCmd = &cobra.Command{
	Use:   "relay",
	Short: "relay client",
	Long: `gbans relay client
`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		go func() {
			if err := relay.NewClient(ctx, serverName, logPath, relayAddr); err != nil {
				log.Fatalf("Exited client: %v", err)
			}
		}()
		exitChan := make(chan os.Signal, 10)
		signal.Notify(exitChan, os.Interrupt, syscall.SIGTERM)
		select {
		case <-exitChan:
			return
		case <-ctx.Done():
			return
		}
	},
}

func init() {
	relayCmd.PersistentFlags().StringVarP(&serverName, "name", "n", "", "Server ID used for identification")
	relayCmd.PersistentFlags().StringVarP(&relayAddr, "host", "H", "localhost", "Server host to send logs to")
	relayCmd.PersistentFlags().StringVarP(&logPath, "logdir", "l", "", "Path to tf2 logs directory")
	rootCmd.AddCommand(relayCmd)
}
