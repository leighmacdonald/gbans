package cmd

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"time"
)

var (
	name = ""
	host = ""
	port = 27015
	rcon = ""
)

// addServerCmd represents the addserver command
var addServerCmd = &cobra.Command{
	Use:   "addserver",
	Short: "Add a new server",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := store.New(config.DB.DSN)
		if err != nil {
			log.Fatalf("Failed to setup db connection: %v", err)
		}
		if name == "" {
			log.Fatal("Server name cannot be empty")
		}
		if host == "" {
			log.Fatal("Server address cannot be empty")
		}
		if port <= 0 || port > 65535 {
			log.Fatal("Invalid server port")
		}
		if rcon == "" {
			log.Fatal("rcon password cannot be empty")
		}
		srv := model.NewServer(name, host, port)
		srv.RCON = rcon
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		if err := db.SaveServer(ctx, &srv); err != nil {
			log.Fatalf("Could not create server: %v", err)
		}
		log.WithFields(log.Fields{"token": srv.Password, "name": name}).
			Info("Added server successfully. This password must be added to your servers gbans.cfg")
	},
}

func init() {
	rootCmd.AddCommand(addServerCmd)

	addServerCmd.Flags().StringVarP(&name, "name", "n", "", "Short server ID eg: us-1")
	addServerCmd.Flags().StringVarP(&host, "host", "H", "", "Server hostname/ip eg: us-1.myserver.com")
	addServerCmd.Flags().IntVarP(&port, "port", "p", 27015, "Server port")
	addServerCmd.Flags().StringVarP(&rcon, "rcon", "r", "", "Server rcon password")
}
