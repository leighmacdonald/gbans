package cmd

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/proxy"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"net"
)

// proxyCmd starts the L7 proxy
var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "proxy client",
	Long:  `gbans game proxy proxy`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		fa, _ := net.ResolveUDPAddr("udp", "192.168.0.101:10000")
		ba, _ := net.ResolveUDPAddr("udp", "192.168.0.220:27015")
		p, err := proxy.New(ctx, fa, ba, proxy.Opts{
			Limit: 75,
		})
		if err != nil {
			log.Errorf("Proxy error: %v", err)
			return
		}
		p.Start()
	},
}

func init() {
	rootCmd.AddCommand(proxyCmd)
}
