package cmd

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/relay"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// relayCmd starts the log relay service
var relayCmd = &cobra.Command{
	Use:   "relay",
	Short: "relay client",
	Long: `gbans relay client
`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		agent, err2 := relay.NewAgent(ctx, relay.Opts{
			ServerAddress:    config.RPC.Addr,
			LogListenAddress: config.RPC.LogAddr,
			Instances: []relay.Instance{
				{
					Name:   "yyc-1",
					Secret: []byte("yyc-1"),
				},
			},
		})
		if err2 != nil {
			log.Fatalf("Exited client: %v", err2)
		}
		if err3 := agent.Start(); err3 != nil {
			log.Errorf("Agent returned error: %v", err3)
		}
	},
}

func init() {
	rootCmd.AddCommand(relayCmd)
}
