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
	Short: "gbans agent",
	Long: `gbans remote administration agent
`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		agent, err2 := agent.NewAgent(ctx, agent.Opts{
			ServerAddress:    config.RPC.Addr,
			LogListenAddress: config.RPC.LogAddr,
			Instances: []agent.Instance{
				{
					Name:   "yyc-1",
					Secret: []byte("yyc-1"),
				},
			},
		})
		if err2 != nil {
			log.Fatalf("Could not create rpc client: %v", err2)
		}

		if errStart := agent.Start(); errStart != nil {
			log.Error("Agent exited: %v", errStart)
		}
	},
}

func init() {
	rootCmd.AddCommand(agentCmd)
}
