package cmd

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"time"
)

var (
	asn      = ""
	steamId  = ""
	cidr     = ""
	reason   = ""
	duration = 0
)

// serverCmd represents the addserver command
var banCmd = &cobra.Command{
	Use:   "ban",
	Short: "ban functions",
	Long:  `Functionality for ban, or modifying bans`,
}

var banSteamCmd = &cobra.Command{
	Use:   "steam",
	Short: "create a steam ban",
	Long:  `Create a new steam ban in the database`,
	Run: func(cmd *cobra.Command, args []string) {
		db, err := store.New(config.DB.DSN)
		if err != nil {
			log.Fatalf("Failed to setup db connection: %v", err)
		}
		if nameLong == "" {
			log.Fatal("Server nameLong cannot be empty")
		}
		if host == "" {
			log.Fatal("Server address cannot be empty")
		}
		if port == 0 {
			port = 27015
		}
		if port <= 0 || port > 65535 {
			log.Fatal("Invalid server port")
		}
		if rcon == "" {
			log.Fatal("rcon password cannot be empty")
		}
		srv := model.NewServer(nameLong, host, port)
		srv.RCON = rcon
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		if err := db.SaveServer(ctx, &srv); err != nil {
			log.Fatalf("Could not create server: %v", err)
		}
		log.WithFields(log.Fields{"token": srv.Password, "nameLong": nameLong}).
			Info("Added server successfully. This password must be added to your servers gbans.cfg")
	},
}

var banCIDRCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete an existing server",
	Long: `Deletes an existing server in the database. This will also delete all associated data such 
	as log data, stats, demos. It is non-reversible`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
		defer cancel()
		db, err := store.New(config.DB.DSN)
		if err != nil {
			log.Fatalf("Failed to setup db connection: %v", err)
		}
		var server model.Server
		if err := db.GetServerByName(ctx, serverId, &server); err != nil {
			if errors.Is(err, store.ErrNoResult) {
				log.WithFields(log.Fields{"server_id": serverId}).Fatalf("Server not found: %s", serverId)
			}
			log.WithFields(log.Fields{"server_id": serverId}).Fatalf("Failed to setup db connection: %v", err)
		}
		log.WithFields(log.Fields{"server_id": serverId}).Infof("Server deleted successfully")
	},
}

var banASNCmd = &cobra.Command{
	Use:   "update",
	Short: "update an existing server config",
	Long:  `Update an existing server in the database`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
		defer cancel()
		db, err := store.New(config.DB.DSN)
		if err != nil {
			log.Fatalf("Failed to setup db connection: %v", err)
		}
		var server model.Server
		if err := db.GetServerByName(ctx, serverId, &server); err != nil {
			if errors.Is(err, store.ErrNoResult) {
				log.WithFields(log.Fields{"server_id": serverId}).Fatalf("Server not found: %s", serverId)
			}
			log.WithFields(log.Fields{"server_id": serverId}).Fatalf("Failed to fetch server to update: %v", err)
		}
		if serverIdNew != "" {
			server.ServerName = serverIdNew
		}
		if nameLong != "" {
			server.ServerNameLong = nameLong
		}
		if host != "" {
			server.Address = host
		}
		if port != 0 {
			server.Port = port
		}
		if rcon != "" {
			server.RCON = rcon
		}
		if errSave := db.SaveServer(ctx, &server); errSave != nil {
			log.WithFields(log.Fields{"server_id": serverId}).Fatalf("Failed to save server: %v", errSave)
		}
		log.WithFields(log.Fields{"server_id": serverId}).Infof("Server updated successfully")
	},
}

func init() {
	banSteamCmd.Flags().StringVarP(&serverId, "id", "i", "", "Short server ID eg: us-1")
	banSteamCmd.Flags().StringVarP(&nameLong, "nameLong", "n", "", "Server nameLong eg: My Game Server")
	banSteamCmd.Flags().StringVarP(&host, "host", "H", "", "Server hostname/ip eg: us-1.myserver.com")
	banSteamCmd.Flags().IntVarP(&port, "port", "p", 0, "Server port")
	banSteamCmd.Flags().StringVarP(&rcon, "rcon", "r", "", "Server rcon password")

	banCIDRCmd.Flags().StringVarP(&serverId, "id", "i", "", "Existing server id to change: us-1")
	banCIDRCmd.Flags().StringVarP(&serverIdNew, "idnew", "I", "", "New server id eg: us-2")
	banCIDRCmd.Flags().StringVarP(&nameLong, "name", "n", "", "New nameLong eg: My Game Server")
	banCIDRCmd.Flags().StringVarP(&host, "host", "H", "", "New hostname/ip eg: us-1.myserver.com")
	banCIDRCmd.Flags().IntVarP(&port, "port", "p", 0, "New port")
	banCIDRCmd.Flags().StringVarP(&rcon, "rcon", "r", "", "New rcon password")

	banASNCmd.Flags().StringVarP(&asn, "asn", "a", "", "Autonomous Systems Number to ban eg: 10551")

	banCmd.AddCommand(banSteamCmd)
	banCmd.AddCommand(banCIDRCmd)
	banCmd.AddCommand(banASNCmd)

	rootCmd.AddCommand(banCmd)

}
