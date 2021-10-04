package cmd

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/golib"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"time"
)

var (
	serverId    = ""
	serverIdNew = ""
	nameLong    = ""
	host        = ""
	port        = 0
	rcon        = ""
)

// serverCmd represents the addserver command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Server functions",
	Long:  `Functionality for creating, or modifying server configurations`,
}

var serverListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all servers",
	Long:  `List all servers`,
	Run: func(cmd *cobra.Command, args []string) {
		db, err := store.New(config.DB.DSN)
		if err != nil {
			log.Fatalf("Failed to setup db connection: %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		servers, errFetch := db.GetServers(ctx, false)
		if errFetch != nil {
			if errors.Is(errFetch, store.ErrNoResult) {
				log.Infof("No servers")
				return
			}
			log.Fatalf("Failed to fetch servers: %v", errFetch)
		}
		var rows [][]string
		for _, srv := range servers {
			rows = append(rows, []string{
				fmt.Sprintf("%d", srv.ServerID),
				srv.ServerName,
				srv.ServerNameLong,
				srv.Addr(),
				srv.Region,
				srv.CC,
				fmt.Sprintf("%.4f %.4f", srv.Location.Latitude, srv.Location.Longitude),
			})
		}
		opts := golib.DefaultTableOpts()
		opts.Title = "Servers"
		opts.Headers = []string{"id", "name", "name_long", "address", "region", "country", "location"}
		fmt.Println(golib.ToTable(rows, opts))
	},
}
var serverCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "create a server",
	Long:  `Create a new server entry in the database`,
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

var serverDeleteCmd = &cobra.Command{
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

var serverUpdateCmd = &cobra.Command{
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
	serverCreateCmd.Flags().StringVarP(&serverId, "id", "i", "", "Short server ID eg: us-1")
	serverCreateCmd.Flags().StringVarP(&nameLong, "nameLong", "n", "", "Server nameLong eg: My Game Server")
	serverCreateCmd.Flags().StringVarP(&host, "host", "H", "", "Server hostname/ip eg: us-1.myserver.com")
	serverCreateCmd.Flags().IntVarP(&port, "port", "p", 0, "Server port")
	serverCreateCmd.Flags().StringVarP(&rcon, "rcon", "r", "", "Server rcon password")

	serverUpdateCmd.Flags().StringVarP(&serverId, "id", "i", "", "Existing server id to change: us-1")
	serverUpdateCmd.Flags().StringVarP(&serverIdNew, "idnew", "I", "", "New server id eg: us-2")
	serverUpdateCmd.Flags().StringVarP(&nameLong, "name", "n", "", "New nameLong eg: My Game Server")
	serverUpdateCmd.Flags().StringVarP(&host, "host", "H", "", "New hostname/ip eg: us-1.myserver.com")
	serverUpdateCmd.Flags().IntVarP(&port, "port", "p", 0, "New port")
	serverUpdateCmd.Flags().StringVarP(&rcon, "rcon", "r", "", "New rcon password")

	serverDeleteCmd.Flags().StringVarP(&serverId, "id", "i", "", "Short server ID eg: us-1")

	serverCmd.AddCommand(serverCreateCmd)
	serverCmd.AddCommand(serverDeleteCmd)
	serverCmd.AddCommand(serverUpdateCmd)
	serverCmd.AddCommand(serverListCmd)

	rootCmd.AddCommand(serverCmd)

}
