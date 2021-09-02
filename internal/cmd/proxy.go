package cmd

import (
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

		fa, _ := net.ResolveUDPAddr("udp", "192.168.0.101:10000")
		ba, _ := net.ResolveUDPAddr("udp", "192.168.0.101:27015")
		p, err := proxy.New(fa, ba)
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
