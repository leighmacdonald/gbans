package cmd

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/relay"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/signal"
	"syscall"
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
			if err2 := relay.New(ctx, config.Relay.ServerName, config.Relay.LogPath,
				config.Relay.Host, config.Relay.Password); err2 != nil {
				log.Fatalf("Exited client: %v", err2)
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
	rootCmd.AddCommand(relayCmd)
}
