package cmd

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/agent"
	"github.com/leighmacdonald/gbans/internal/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// agentCmd starts the log relay service
var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "gbans remote agent",
	Long:  `gbans remote agent`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		agent, err2 := agent.NewAgent(ctx, agent.Opts{
			ServerAddress:    config.RPC.Addr,
			LogListenAddress: config.RPC.LogAddr,
			Instances: []agent.Instance{
				{
					Name:   "abc-1",
					Secret: []byte("abc-1"),
				},
			},
		})
		if err2 != nil {
			log.Fatalf("Could not create rpc client: %v", err2)
		}
		if errStart := agent.Start(); errStart != nil {
			log.Errorf("Agent exited: %v", errStart)
		}
		<-ctx.Done()
	},
}

func init() {
	rootCmd.AddCommand(agentCmd)
}
