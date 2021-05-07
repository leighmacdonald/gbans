package cmd

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/relay"
	"github.com/leighmacdonald/gbans/internal/service"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var netCmd = &cobra.Command{
	Use:   "block",
	Short: "Network and client blocking functionality",
	Long:  `Network and client blocking functionality`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		duration, err := config.ParseDuration(timeoutStr)
		if err != nil {
			log.Fatalf("Invalid timeout value: %v", err)
		}
		go func() {
			if err := relay.New(ctx, serverName, logPath, relayAddr, duration); err != nil {
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

var netUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update any enabled block lists",
	Long:  `Update any enabled block lists`,
	Run: func(cmd *cobra.Command, args []string) {
		service.Init(config.DB.DSN)
		if err := ip2location.Update(config.Net.CachePath, config.Net.IP2Location.Token); err != nil {
			log.Fatalf("Failed to update")
		}
		d, err := ip2location.Read(config.Net.CachePath)
		if err != nil {
			log.Fatalf("Failed to read")
		}
		if err := service.InsertBlockListData(d); err != nil {
			log.Fatalf("Failed to import")
		}
	},
}

func init() {
	netCmd.AddCommand(netUpdateCmd)
	rootCmd.AddCommand(netCmd)
}
